package mailproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/store"
)

const (
	DefaultIMAPListen   = "127.0.0.1:1993"
	DefaultSMTPListen   = "127.0.0.1:1587"
	DefaultIMAPUpstream = "imap.gmail.com:993"
	DefaultSMTPUpstream = "smtp.gmail.com:587"
)

type Config struct {
	VaultName       string
	ServiceName     string
	IMAPListen      string
	SMTPListen      string
	IMAPUpstream    string
	SMTPUpstream    string
	ShutdownTimeout time.Duration
}

type FlagValues struct {
	VaultName       string
	ServiceName     string
	IMAPListen      string
	SMTPListen      string
	IMAPUpstream    string
	SMTPUpstream    string
	ShutdownTimeout time.Duration
}

type Store interface {
	GetVault(ctx context.Context, name string) (*store.Vault, error)
	GetBrokerConfig(ctx context.Context, vaultID string) (*store.BrokerConfig, error)
	GetCredential(ctx context.Context, vaultID, key string) (*store.Credential, error)
	GetCredentialOAuth(ctx context.Context, vaultID, key string) (*store.CredentialOAuth, error)
}

type PreflightResult struct {
	VaultID string
	Service broker.Service
}

func LoadConfig(flags FlagValues, getenv func(string) string) Config {
	if getenv == nil {
		getenv = os.Getenv
	}

	cfg := Config{
		VaultName:       firstNonEmpty(flags.VaultName, getenv("AGENT_VAULT_MAIL_VAULT"), store.DefaultVault),
		ServiceName:     firstNonEmpty(flags.ServiceName, getenv("AGENT_VAULT_MAIL_SERVICE")),
		IMAPListen:      firstNonEmpty(flags.IMAPListen, getenv("AGENT_VAULT_MAIL_IMAP_LISTEN"), DefaultIMAPListen),
		SMTPListen:      firstNonEmpty(flags.SMTPListen, getenv("AGENT_VAULT_MAIL_SMTP_LISTEN"), DefaultSMTPListen),
		IMAPUpstream:    firstNonEmpty(flags.IMAPUpstream, getenv("AGENT_VAULT_MAIL_IMAP_UPSTREAM"), DefaultIMAPUpstream),
		SMTPUpstream:    firstNonEmpty(flags.SMTPUpstream, getenv("AGENT_VAULT_MAIL_SMTP_UPSTREAM"), DefaultSMTPUpstream),
		ShutdownTimeout: flags.ShutdownTimeout,
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}
	return cfg
}

func (c Config) Validate() error {
	if c.VaultName == "" {
		return fmt.Errorf("vault is required")
	}
	if c.ServiceName == "" {
		return fmt.Errorf("service is required")
	}
	if err := validateLoopbackListen(c.IMAPListen); err != nil {
		return fmt.Errorf("imap listen: %w", err)
	}
	if err := validateLoopbackListen(c.SMTPListen); err != nil {
		return fmt.Errorf("smtp listen: %w", err)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	return nil
}

func Preflight(ctx context.Context, store Store, cfg Config) (*PreflightResult, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	vault, err := store.GetVault(ctx, cfg.VaultName)
	if err != nil || vault == nil {
		return nil, fmt.Errorf("vault %q not found", cfg.VaultName)
	}

	brokerCfg, err := store.GetBrokerConfig(ctx, vault.ID)
	if err != nil || brokerCfg == nil || brokerCfg.ServicesJSON == "" {
		return nil, fmt.Errorf("service %q not found", cfg.ServiceName)
	}

	var services []broker.Service
	if err := json.Unmarshal([]byte(brokerCfg.ServicesJSON), &services); err != nil {
		return nil, fmt.Errorf("parse services: %w", err)
	}

	for i := range services {
		services[i].Host, services[i].Path, services[i].Port = broker.SplitInlineHost(services[i].Host, services[i].Path)
		if services[i].Name == cfg.ServiceName {
			if err := validateService(ctx, store, vault.ID, services[i]); err != nil {
				return nil, err
			}
			return &PreflightResult{VaultID: vault.ID, Service: services[i]}, nil
		}
	}

	return nil, fmt.Errorf("service %q not found", cfg.ServiceName)
}

func validateService(ctx context.Context, store Store, vaultID string, svc broker.Service) error {
	if !svc.IsEnabled() {
		return fmt.Errorf("service %q is disabled", svc.Name)
	}
	if svc.MailProxy == nil {
		return fmt.Errorf("service %q missing mail_proxy policy", svc.Name)
	}
	if !svc.MailProxy.IMAP && !svc.MailProxy.SMTP {
		return fmt.Errorf("service %q has no enabled mail proxy protocol", svc.Name)
	}
	if strings.TrimSpace(svc.MailProxy.Email) == "" {
		return fmt.Errorf("service %q missing mail_proxy.email", svc.Name)
	}
	if strings.TrimSpace(svc.MailProxy.LocalPasswordCredential) == "" {
		return fmt.Errorf("service %q missing mail_proxy.local_password_credential", svc.Name)
	}
	if svc.Auth.Type != "bearer" || strings.TrimSpace(svc.Auth.Token) == "" {
		return fmt.Errorf("service %q must use bearer auth token credential", svc.Name)
	}

	localPassword, err := store.GetCredential(ctx, vaultID, svc.MailProxy.LocalPasswordCredential)
	if err != nil || localPassword == nil || len(localPassword.Ciphertext) == 0 || localPassword.Type == "oauth" {
		return fmt.Errorf("local password credential %q not found or empty", svc.MailProxy.LocalPasswordCredential)
	}

	oauthCredential, err := store.GetCredential(ctx, vaultID, svc.Auth.Token)
	if err != nil || oauthCredential == nil || oauthCredential.Type != "oauth" || len(oauthCredential.Ciphertext) == 0 {
		return fmt.Errorf("oauth credential %q not found or not connected", svc.Auth.Token)
	}

	oauthCfg, err := store.GetCredentialOAuth(ctx, vaultID, svc.Auth.Token)
	if err != nil || oauthCfg == nil {
		return fmt.Errorf("oauth credential %q not found or not connected", svc.Auth.Token)
	}
	if len(oauthCfg.RefreshTokenCT) == 0 {
		return fmt.Errorf("oauth credential %q has no refresh token", svc.Auth.Token)
	}

	return nil
}

func validateLoopbackListen(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q", addr)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("address %q must bind to loopback", addr)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
