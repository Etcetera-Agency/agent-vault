package cmd

import (
	"testing"
	"time"

	"github.com/Infisical/agent-vault/internal/mailproxy"
	"github.com/spf13/cobra"
)

func TestMailProxyConfigFromFlagsUsesEnvWhenFlagUnchanged(t *testing.T) {
	t.Setenv("AGENT_VAULT_MAIL_VAULT", "env-vault")
	t.Setenv("AGENT_VAULT_MAIL_SERVICE", "env-service")
	t.Setenv("AGENT_VAULT_MAIL_IMAP_LISTEN", "127.0.0.1:2993")

	cmd := newMailProxyConfigTestCommand()
	cfg, err := mailProxyConfigFromFlags(cmd)
	if err != nil {
		t.Fatalf("mailProxyConfigFromFlags: %v", err)
	}
	if cfg.VaultName != "env-vault" || cfg.ServiceName != "env-service" {
		t.Fatalf("cfg = %+v, want env vault/service", cfg.Config)
	}
	if cfg.IMAPListen != "127.0.0.1:2993" {
		t.Fatalf("IMAPListen = %q, want env value", cfg.IMAPListen)
	}
	if cfg.SMTPListen != mailproxy.DefaultSMTPListen {
		t.Fatalf("SMTPListen = %q, want default", cfg.SMTPListen)
	}
}

func TestMailProxyConfigFromFlagsPreferChangedFlags(t *testing.T) {
	t.Setenv("AGENT_VAULT_MAIL_VAULT", "env-vault")
	t.Setenv("AGENT_VAULT_MAIL_SERVICE", "env-service")

	cmd := newMailProxyConfigTestCommand()
	if err := cmd.Flags().Set("vault", "flag-vault"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("service", "flag-service"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("shutdown-timeout", "3s"); err != nil {
		t.Fatal(err)
	}

	cfg, err := mailProxyConfigFromFlags(cmd)
	if err != nil {
		t.Fatalf("mailProxyConfigFromFlags: %v", err)
	}
	if cfg.VaultName != "flag-vault" || cfg.ServiceName != "flag-service" {
		t.Fatalf("cfg = %+v, want flag vault/service", cfg.Config)
	}
	if cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 3s", cfg.ShutdownTimeout)
	}
}

func newMailProxyConfigTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("vault", "", "")
	cmd.Flags().String("service", "", "")
	cmd.Flags().String("imap-listen", mailproxy.DefaultIMAPListen, "")
	cmd.Flags().String("smtp-listen", mailproxy.DefaultSMTPListen, "")
	cmd.Flags().String("imap-upstream", mailproxy.DefaultIMAPUpstream, "")
	cmd.Flags().String("smtp-upstream", mailproxy.DefaultSMTPUpstream, "")
	cmd.Flags().Duration("shutdown-timeout", 10*time.Second, "")
	cmd.Flags().String("database-url", "", "")
	cmd.Flags().Bool("password-stdin", false, "")
	return cmd
}
