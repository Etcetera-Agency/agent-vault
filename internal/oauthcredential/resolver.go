package oauthcredential

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Infisical/agent-vault/internal/crypto"
	"github.com/Infisical/agent-vault/internal/oauth"
	"github.com/Infisical/agent-vault/internal/store"
)

const refreshBuffer = 5 * time.Minute

var ErrRefreshFailed = errors.New("oauthcredential: oauth token refresh failed")

type Store interface {
	GetCredentialOAuth(ctx context.Context, vaultID, key string) (*store.CredentialOAuth, error)
	UpdateCredentialOAuthTokens(ctx context.Context, vaultID, key string, accessCT, accessNonce, refreshCT, refreshNonce []byte, expiresAt *time.Time) error
	UpdateCredentialOAuthError(ctx context.Context, vaultID, key string, errMsg string) error
}

type ResolveOptions struct {
	ForceRefresh bool
}

type Resolver struct {
	store     Store
	encKey    []byte
	refresher *oauth.Refresher
}

func NewResolver(store Store, encKey []byte, refresher *oauth.Refresher) *Resolver {
	return &Resolver{
		store:     store,
		encKey:    encKey,
		refresher: refresher,
	}
}

func (r *Resolver) Resolve(ctx context.Context, vaultID, key, currentToken string, opts ResolveOptions) (string, error) {
	if r == nil || r.store == nil || r.refresher == nil {
		return currentToken, nil
	}

	oauthCfg, err := r.store.GetCredentialOAuth(ctx, vaultID, key)
	if err != nil {
		return currentToken, nil
	}

	if !opts.ForceRefresh && tokenFresh(oauthCfg.TokenExpiresAt) {
		return currentToken, nil
	}
	if len(oauthCfg.RefreshTokenCT) == 0 {
		return currentToken, nil
	}

	sfKey := vaultID + "|" + key
	result := r.refresher.Do(sfKey, func() oauth.RefreshResult {
		return r.refresh(ctx, vaultID, key, oauthCfg)
	})
	if result.Err != nil {
		return "", result.Err
	}
	if result.Refreshed {
		return result.AccessToken, nil
	}
	return currentToken, nil
}

func tokenFresh(expiresAt *time.Time) bool {
	return expiresAt == nil || time.Until(*expiresAt) > refreshBuffer
}

func (r *Resolver) refresh(ctx context.Context, vaultID, key string, oauthCfg *store.CredentialOAuth) oauth.RefreshResult {
	refreshToken, err := crypto.Decrypt(oauthCfg.RefreshTokenCT, oauthCfg.RefreshTokenNonce, r.encKey)
	if err != nil {
		return oauth.RefreshResult{Err: fmt.Errorf("%w: decrypt refresh token: %v", ErrRefreshFailed, err)}
	}

	clientSecret, err := r.decryptClientSecret(oauthCfg)
	if err != nil {
		return oauth.RefreshResult{Err: err}
	}

	tok, err := oauth.Refresh(ctx, oauth.RefreshConfig{
		TokenURL:        oauthCfg.TokenURL,
		ClientID:        oauthCfg.ClientID,
		ClientSecret:    clientSecret,
		RefreshToken:    string(refreshToken),
		Scopes:          oauthCfg.Scopes,
		ScopeSeparator:  oauthCfg.ScopeSeparator,
		TokenAuthMethod: oauthCfg.TokenAuthMethod,
	})
	if err != nil {
		_ = r.store.UpdateCredentialOAuthError(ctx, vaultID, key, err.Error())
		return oauth.RefreshResult{Err: fmt.Errorf("%w: %v", ErrRefreshFailed, err)}
	}

	accessCT, accessNonce, err := crypto.Encrypt([]byte(tok.AccessToken), r.encKey)
	if err != nil {
		return oauth.RefreshResult{Err: fmt.Errorf("%w: encrypt access token: %v", ErrRefreshFailed, err)}
	}

	var refreshCT, refreshNonce []byte
	if tok.RefreshToken != "" {
		refreshCT, refreshNonce, err = crypto.Encrypt([]byte(tok.RefreshToken), r.encKey)
		if err != nil {
			return oauth.RefreshResult{Err: fmt.Errorf("%w: encrypt refresh token: %v", ErrRefreshFailed, err)}
		}
	}

	var expiresAt *time.Time
	if !tok.ExpiresAt.IsZero() {
		expiresAt = &tok.ExpiresAt
	}

	if err := r.store.UpdateCredentialOAuthTokens(ctx, vaultID, key, accessCT, accessNonce, refreshCT, refreshNonce, expiresAt); err != nil {
		return oauth.RefreshResult{Err: fmt.Errorf("%w: store tokens: %v", ErrRefreshFailed, err)}
	}

	return oauth.RefreshResult{AccessToken: tok.AccessToken, Refreshed: true}
}

func (r *Resolver) decryptClientSecret(oauthCfg *store.CredentialOAuth) (string, error) {
	if len(oauthCfg.ClientSecretCT) == 0 {
		return "", nil
	}

	clientSecret, err := crypto.Decrypt(oauthCfg.ClientSecretCT, oauthCfg.ClientSecretNonce, r.encKey)
	if err != nil {
		return "", fmt.Errorf("%w: decrypt client secret: %v", ErrRefreshFailed, err)
	}
	return string(clientSecret), nil
}
