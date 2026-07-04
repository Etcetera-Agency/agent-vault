package brokercore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/crypto"
	"github.com/Infisical/agent-vault/internal/egressquota"
	"github.com/Infisical/agent-vault/internal/oauth"
	"github.com/Infisical/agent-vault/internal/oauthcredential"
	"github.com/Infisical/agent-vault/internal/store"
)

// UnmatchedHostPolicy controls what happens when a request's target host
// does not match any configured broker service. PolicyPassthrough is the
// system-wide default; PolicyDeny is the opt-in strict mode.
type UnmatchedHostPolicy string

const (
	PolicyPassthrough UnmatchedHostPolicy = "passthrough"
	PolicyDeny        UnmatchedHostPolicy = "deny"
)

func IsValidUnmatchedHostPolicy(p UnmatchedHostPolicy) bool {
	return p == PolicyPassthrough || p == PolicyDeny
}

// InjectResult is the outcome of matching (host, path) and resolving
// credentials to ready-to-attach HTTP headers.
type InjectResult struct {
	// Headers carries SECRET values — never log. Caller must Set (not
	// Add) so injected values win over client-supplied duplicates.
	// Nil for passthrough services.
	Headers map[string]string

	// MatchedName/Host/Path/Port describe the matched service. Safe to log.
	// Empty under unmatched-host passthrough.
	MatchedName string
	MatchedHost string
	MatchedPath string
	MatchedPort *int

	// CredentialKeys are the key names referenced by the matched
	// service. Populated before resolution so credential-missing
	// errors still carry diagnostic context. Safe to log.
	CredentialKeys []string

	// AccountID is the selected service account identity for quota
	// state. Safe to log; never a credential value.
	AccountID string

	// Substitutions are resolved placeholder rewrites; each entry
	// carries a SECRET Value — never log placeholder values.
	Substitutions []ResolvedSubstitution

	// Passthrough is set when no service matched but the unmatched-host
	// policy permitted forwarding.
	Passthrough bool

	QuotaReservation *egressquota.Reservation
}

// CredentialProvider resolves a service for (targetHost, targetMethod,
// targetPath) in vaultID and returns the headers to attach. targetPath must
// be the URL path only — no query, no fragment.
type CredentialProvider interface {
	Inject(ctx context.Context, vaultID, targetHost string, targetPort int, targetMethod, targetPath string) (*InjectResult, error)
}

// CredentialStore is the minimal store surface used by StoreCredentialProvider.
type CredentialStore interface {
	GetBrokerConfig(ctx context.Context, vaultID string) (*store.BrokerConfig, error)
	GetCredential(ctx context.Context, vaultID, key string) (*store.Credential, error)
	UnmatchedHostPolicy(ctx context.Context, vaultID string) (UnmatchedHostPolicy, error)
}

// OAuthStore is the store surface for OAuth token refresh.
// Passed separately to StoreCredentialProvider to keep CredentialStore minimal.
type OAuthStore interface {
	oauthcredential.Store
}

// DynamicCredentialResolver resolves credential keys that are not stored
// statically — e.g. Infisical dynamic-secret leases minted on demand. ok=false
// means "not a dynamic credential" (the caller keeps its not-found error); a
// non-nil error is a real failure. Implemented outside brokercore (infisical)
// and injected, so brokercore takes no dependency on it.
type DynamicCredentialResolver interface {
	Resolve(ctx context.Context, vaultID, key string) (value string, ok bool, err error)
}

// StoreCredentialProvider injects credentials using a CredentialStore and a
// 32-byte AES-256-GCM key held in memory for the lifetime of the process.
type StoreCredentialProvider struct {
	Store      CredentialStore
	OAuthStore OAuthStore // nil = no OAuth refresh
	EncKey     []byte
	Refresher  *oauth.Refresher          // nil = no OAuth refresh
	Dynamic    DynamicCredentialResolver // nil = no dynamic-secret resolution
	Quota      *egressquota.Registry     // nil = no egress quota enforcement
}

// NewStoreCredentialProvider constructs a provider. encKey must be 32 bytes.
func NewStoreCredentialProvider(s CredentialStore, encKey []byte) *StoreCredentialProvider {
	return &StoreCredentialProvider{Store: s, EncKey: encKey}
}

// Inject matches (targetHost, targetPath) and resolves the matched
// service's auth into HTTP headers. targetHost may include a port —
// stripped before matching. Pass "/" for targetPath when no path is
// meaningful.
func (p *StoreCredentialProvider) Inject(ctx context.Context, vaultID, targetHost string, targetPort int, targetMethod, targetPath string) (*InjectResult, error) {
	// A missing row is equivalent to an empty services list — fall
	// through to the unmatched-host policy. Any other error fails closed
	// so a transient store failure can't silently strip enforcement.
	cfg, err := p.Store.GetBrokerConfig(ctx, vaultID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, ErrServiceNotFound
	}

	var services []broker.Service
	if cfg != nil && cfg.ServicesJSON != "" {
		if err := json.Unmarshal([]byte(cfg.ServicesJSON), &services); err != nil {
			return nil, fmt.Errorf("brokercore: parsing broker services: %w", err)
		}
	}
	// MarshalJSON persists Host in joined-inline form; the matcher
	// requires Host without "/", so split before matching.
	for i := range services {
		services[i].Host, services[i].Path, services[i].Port = broker.SplitInlineHost(services[i].Host, services[i].Path)
	}
	// Heal legacy unnamed entries so MatchedName (which lands in the
	// request log and the X-Vault-Service header) is never blank for a
	// matched service — the documented `?service=<name>` log filter
	// depends on it.
	broker.AssignSlugNames(services)

	matchHost := targetHost
	if h, _, err := net.SplitHostPort(targetHost); err == nil {
		matchHost = h
	}
	if targetPath == "" {
		targetPath = "/"
	}
	// fork-local: Gate the upstream host/path match with stored method policy before credential injection.
	matched, score, matchStatus := broker.MatchServiceWithMethodPolicy(matchHost, targetPort, targetPath, targetMethod, services)
	if matchStatus == broker.MethodMatchDenied {
		return nil, ErrServiceMethodDenied
	}
	if matched == nil {
		// Fail closed on policy lookup errors so a transient store
		// failure can't silently strip enforcement.
		policy, err := p.Store.UnmatchedHostPolicy(ctx, vaultID)
		if err != nil || policy == PolicyDeny {
			return nil, ErrServiceNotFound
		}
		return &InjectResult{Passthrough: true}, nil
	}
	if !matched.IsEnabled() {
		return nil, ErrServiceDisabled
	}
	slog.Default().Debug("broker matched",
		slog.String("vault", vaultID),
		slog.String("service", matched.Name),
		slog.String("host", matched.Host),
		slog.String("path", matched.Path),
		slog.String("host_tier", score.HostTierName()),
		slog.Int("path_prefix_len", score.PathLiteralLen),
		slog.Int("decl_order", score.DeclOrder),
	)

	var quotaReservation *egressquota.Reservation
	selectedAccountID := ""
	selectedCredentialKey := ""
	if p.Quota != nil {
		reservation, denial := p.Quota.Reserve(ctx, vaultID, *matched)
		if denial != nil {
			return &InjectResult{
				MatchedName:    matched.Name,
				MatchedHost:    matched.Host,
				MatchedPath:    matched.Path,
				MatchedPort:    matched.Port,
				CredentialKeys: matched.CredentialKeys(),
			}, &ErrEgressQuotaExceeded{Decision: denial}
		}
		quotaReservation = reservation
		if reservation != nil {
			selectedAccountID = reservation.AccountID()
			selectedCredentialKey = reservation.CredentialKey()
		}
		if selectedCredentialKey != "" {
			copySvc := *matched
			copySvc.Auth = accountAuth(copySvc.Auth, selectedCredentialKey)
			matched = &copySvc
		}
	}

	// Memoize per-key lookups so a credential shared by auth and a
	// substitution decrypts only once.
	cache := make(map[string]string)
	getCredential := func(key string) (string, error) {
		if v, ok := cache[key]; ok {
			return v, nil
		}
		cred, err := p.Store.GetCredential(ctx, vaultID, key)
		if err != nil || cred == nil {
			// No static credential: try resolving it as a dynamic-secret field.
			if p.Dynamic != nil {
				if val, ok, derr := p.Dynamic.Resolve(ctx, vaultID, key); derr != nil {
					return "", derr
				} else if ok {
					cache[key] = val
					return val, nil
				}
			}
			return "", fmt.Errorf("credential %q not found", key)
		}

		plaintext, err := crypto.Decrypt(cred.Ciphertext, cred.Nonce, p.EncKey)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt credential %q", key)
		}
		s := string(plaintext)

		if cred.Type == "oauth" && s == "" {
			return "", fmt.Errorf("%w: credential %q", ErrOAuthNotConnected, key)
		}

		if cred.Type == "oauth" && p.Refresher != nil && p.OAuthStore != nil {
			// fork-local: shared OAuth resolver is reused by HTTP broker and mail proxy.
			s, err = p.maybeRefreshOAuth(ctx, vaultID, key, s)
			if err != nil {
				return "", err
			}
		}

		cache[key] = s
		return s, nil
	}

	// Capture non-secret metadata up front so a downstream credential-missing
	// error still carries it for diagnostic logging.
	result := &InjectResult{
		MatchedName:      matched.Name,
		MatchedHost:      matched.Host,
		MatchedPath:      matched.Path,
		MatchedPort:      matched.Port,
		CredentialKeys:   selectedCredentialKeys(matched.CredentialKeys(), selectedCredentialKey),
		AccountID:        selectedAccountID,
		QuotaReservation: quotaReservation,
	}

	// Resolve substitutions before auth so passthrough services (which
	// skip the auth branch) still surface ErrCredentialMissing here.
	// Hold locally and attach only on success — error returns must not
	// expose resolved secret values via result.
	var resolvedSubs []ResolvedSubstitution
	if len(matched.Substitutions) > 0 {
		resolvedSubs = make([]ResolvedSubstitution, 0, len(matched.Substitutions))
		for _, sub := range matched.Substitutions {
			val, err := getCredential(sub.Key)
			if err != nil {
				return result, fmt.Errorf("%w: %w", ErrCredentialMissing, err)
			}
			resolvedSubs = append(resolvedSubs, ResolvedSubstitution{
				Placeholder: sub.Placeholder,
				Value:       val,
				In:          sub.NormalizedIn(),
			})
		}
	}

	if matched.Auth.Type == "passthrough" {
		result.Substitutions = resolvedSubs
		return result, nil
	}

	headers, err := matched.Auth.Resolve(getCredential)
	if err != nil {
		return result, fmt.Errorf("%w: %w", ErrCredentialMissing, err)
	}

	result.Headers = headers
	result.Substitutions = resolvedSubs
	return result, nil
}

func (p *StoreCredentialProvider) maybeRefreshOAuth(ctx context.Context, vaultID, key, currentToken string) (string, error) {
	resolver := oauthcredential.NewResolver(p.OAuthStore, p.EncKey, p.Refresher)
	token, err := resolver.Resolve(ctx, vaultID, key, currentToken, oauthcredential.ResolveOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthRefreshFailed, err)
	}
	return token, nil
}

func accountAuth(auth broker.Auth, accountKey string) broker.Auth {
	if accountKey == "" {
		return auth
	}
	switch auth.Type {
	case "bearer":
		auth.Token = accountKey
	case "api-key":
		auth.Key = accountKey
	case "basic":
		auth.Username = accountKey
	}
	return auth
}

func selectedCredentialKeys(keys []string, selected string) []string {
	if selected == "" {
		return keys
	}
	out := []string{selected}
	for _, key := range keys {
		if key != selected {
			out = append(out, key)
		}
	}
	return out
}
