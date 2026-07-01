package mailproxy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/store"
)

type fakePreflightStore struct {
	vault     *store.Vault
	brokerCfg *store.BrokerConfig
	creds     map[string]*store.Credential
	oauth     *store.CredentialOAuth
}

func (f *fakePreflightStore) GetVault(_ context.Context, _ string) (*store.Vault, error) {
	return f.vault, nil
}

func (f *fakePreflightStore) GetBrokerConfig(_ context.Context, _ string) (*store.BrokerConfig, error) {
	return f.brokerCfg, nil
}

func (f *fakePreflightStore) GetCredential(_ context.Context, _, key string) (*store.Credential, error) {
	return f.creds[key], nil
}

func (f *fakePreflightStore) GetCredentialOAuth(_ context.Context, _, _ string) (*store.CredentialOAuth, error) {
	return f.oauth, nil
}

func TestLoadConfigFlagsOverrideEnv(t *testing.T) {
	env := map[string]string{
		"AGENT_VAULT_MAIL_VAULT":       "env-vault",
		"AGENT_VAULT_MAIL_SERVICE":     "env-service",
		"AGENT_VAULT_MAIL_IMAP_LISTEN": "127.0.0.1:2993",
	}
	cfg := LoadConfig(FlagValues{
		VaultName:       "flag-vault",
		ServiceName:     "flag-service",
		IMAPListen:      "127.0.0.1:3993",
		ShutdownTimeout: 5 * time.Second,
	}, func(key string) string {
		return env[key]
	})

	if cfg.VaultName != "flag-vault" || cfg.ServiceName != "flag-service" {
		t.Fatalf("flag precedence failed: %+v", cfg)
	}
	if cfg.IMAPListen != "127.0.0.1:3993" {
		t.Fatalf("IMAPListen = %q", cfg.IMAPListen)
	}
	if cfg.SMTPListen != DefaultSMTPListen {
		t.Fatalf("SMTPListen = %q, want default", cfg.SMTPListen)
	}
}

func TestConfigValidateRejectsMissingService(t *testing.T) {
	cfg := LoadConfig(FlagValues{VaultName: "default"}, nil)
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "service is required") {
		t.Fatalf("err = %v, want service required", err)
	}
}

func TestConfigValidateRejectsNonLoopbackListeners(t *testing.T) {
	cfg := LoadConfig(FlagValues{
		VaultName:   "default",
		ServiceName: "gmail-mail",
		IMAPListen:  "0.0.0.0:1993",
	}, nil)
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("err = %v, want loopback rejection", err)
	}
}

func TestPreflightAcceptsValidService(t *testing.T) {
	store := validPreflightStore(t, broker.Service{
		Name: "gmail-mail",
		Host: "gmail.googleapis.com",
		Auth: broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
		MailProxy: &broker.MailProxyPolicy{
			Email:                   "agent@gmail.com",
			LocalPasswordCredential: "HERMES_MAIL_LOCAL_PASSWORD",
			IMAP:                    true,
		},
	})

	result, err := Preflight(context.Background(), store, validConfig())
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if result.VaultID != "vault-id" || result.Service.Name != "gmail-mail" {
		t.Fatalf("result = %+v", result)
	}
}

func TestPreflightRejectsBothProtocolsDisabled(t *testing.T) {
	store := validPreflightStore(t, broker.Service{
		Name: "gmail-mail",
		Host: "gmail.googleapis.com",
		Auth: broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
		MailProxy: &broker.MailProxyPolicy{
			Email:                   "agent@gmail.com",
			LocalPasswordCredential: "HERMES_MAIL_LOCAL_PASSWORD",
		},
	})

	_, err := Preflight(context.Background(), store, validConfig())
	if err == nil || !strings.Contains(err.Error(), "no enabled mail proxy protocol") {
		t.Fatalf("err = %v, want disabled protocols rejection", err)
	}
}

func TestPreflightRejectsDisabledService(t *testing.T) {
	enabled := false
	store := validPreflightStore(t, broker.Service{
		Name:    "gmail-mail",
		Host:    "gmail.googleapis.com",
		Enabled: &enabled,
		Auth:    broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
		MailProxy: &broker.MailProxyPolicy{
			Email:                   "agent@gmail.com",
			LocalPasswordCredential: "HERMES_MAIL_LOCAL_PASSWORD",
			IMAP:                    true,
		},
	})

	_, err := Preflight(context.Background(), store, validConfig())
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("err = %v, want disabled service rejection", err)
	}
}

func TestPreflightRejectsOAuthCredentialWithoutRefreshToken(t *testing.T) {
	store := validPreflightStore(t, broker.Service{
		Name: "gmail-mail",
		Host: "gmail.googleapis.com",
		Auth: broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
		MailProxy: &broker.MailProxyPolicy{
			Email:                   "agent@gmail.com",
			LocalPasswordCredential: "HERMES_MAIL_LOCAL_PASSWORD",
			IMAP:                    true,
		},
	})
	store.oauth.RefreshTokenCT = nil

	_, err := Preflight(context.Background(), store, validConfig())
	if err == nil || !strings.Contains(err.Error(), "no refresh token") {
		t.Fatalf("err = %v, want refresh token rejection", err)
	}
}

func validPreflightStore(t *testing.T, svc broker.Service) *fakePreflightStore {
	t.Helper()
	servicesJSON, err := json.Marshal([]broker.Service{svc})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return &fakePreflightStore{
		vault:     &store.Vault{ID: "vault-id", Name: "default"},
		brokerCfg: &store.BrokerConfig{VaultID: "vault-id", ServicesJSON: string(servicesJSON)},
		creds: map[string]*store.Credential{
			"HERMES_MAIL_LOCAL_PASSWORD": {Type: "static", Ciphertext: []byte("encrypted-local-password")},
			"GOOGLE_MAIL_OAUTH":          {Type: "oauth", Ciphertext: []byte("encrypted-access-token")},
		},
		oauth: &store.CredentialOAuth{RefreshTokenCT: []byte("encrypted-refresh")},
	}
}

func validConfig() Config {
	return Config{
		VaultName:       "default",
		ServiceName:     "gmail-mail",
		IMAPListen:      DefaultIMAPListen,
		SMTPListen:      DefaultSMTPListen,
		IMAPUpstream:    DefaultIMAPUpstream,
		SMTPUpstream:    DefaultSMTPUpstream,
		ShutdownTimeout: time.Second,
	}
}
