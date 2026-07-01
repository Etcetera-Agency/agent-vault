package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Infisical/agent-vault/internal/ca"
	"github.com/Infisical/agent-vault/internal/mailproxy"
	"github.com/Infisical/agent-vault/internal/store"
	"github.com/spf13/cobra"
)

var mailProxyCmd = &cobra.Command{
	Use:   "mail-proxy",
	Short: "Run local IMAP/SMTP proxy for Hermes Gmail OAuth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mailProxyConfigFromFlags(cmd)
		if err != nil {
			return err
		}

		db, cleanup, err := openMailProxyDB(cfg.databaseURL)
		if err != nil {
			return err
		}
		defer cleanup()

		preflight, err := mailproxy.Preflight(context.Background(), db, cfg.Config)
		if err != nil {
			return err
		}

		masterKey, err := unlockOrSetup(cmd, db, cfg.passwordStdin)
		if err != nil {
			return err
		}
		defer masterKey.Wipe()

		localPassword, err := mailproxy.LoadLocalPassword(
			context.Background(),
			db,
			preflight.VaultID,
			preflight.Service.MailProxy.LocalPasswordCredential,
			masterKey.Key(),
		)
		if err != nil {
			return err
		}
		if _, err := mailproxy.NewLocalAuthenticator(preflight.Service.MailProxy.Email, localPassword); err != nil {
			return err
		}

		caOpts := ca.Options{}
		if db.DialectName() == "postgres" {
			caOpts.Store = &caStoreAdapter{db: db}
		}
		caProvider, err := ca.New(masterKey.Key(), caOpts)
		if err != nil {
			return fmt.Errorf("initializing local CA: %w", err)
		}
		if _, err := mailproxy.LocalTLSConfig(caProvider); err != nil {
			return err
		}

		return fmt.Errorf("mail proxy listeners are not implemented yet")
	},
}

type mailProxyCommandConfig struct {
	mailproxy.Config
	databaseURL   string
	passwordStdin bool
}

func mailProxyConfigFromFlags(cmd *cobra.Command) (mailProxyCommandConfig, error) {
	vaultName := changedStringFlag(cmd, "vault")
	serviceName := changedStringFlag(cmd, "service")
	imapListen := changedStringFlag(cmd, "imap-listen")
	smtpListen := changedStringFlag(cmd, "smtp-listen")
	imapUpstream := changedStringFlag(cmd, "imap-upstream")
	smtpUpstream := changedStringFlag(cmd, "smtp-upstream")
	shutdownTimeout := changedDurationFlag(cmd, "shutdown-timeout")
	databaseURL, _ := cmd.Flags().GetString("database-url")
	passwordStdin, _ := cmd.Flags().GetBool("password-stdin")

	cfg := mailproxy.LoadConfig(mailproxy.FlagValues{
		VaultName:       vaultName,
		ServiceName:     serviceName,
		IMAPListen:      imapListen,
		SMTPListen:      smtpListen,
		IMAPUpstream:    imapUpstream,
		SMTPUpstream:    smtpUpstream,
		ShutdownTimeout: shutdownTimeout,
	}, os.Getenv)
	return mailProxyCommandConfig{
		Config:        cfg,
		databaseURL:   databaseURL,
		passwordStdin: passwordStdin,
	}, cfg.Validate()
}

func changedStringFlag(cmd *cobra.Command, name string) string {
	if !cmd.Flags().Changed(name) {
		return ""
	}
	value, _ := cmd.Flags().GetString(name)
	return value
}

func changedDurationFlag(cmd *cobra.Command, name string) time.Duration {
	if !cmd.Flags().Changed(name) {
		return 0
	}
	value, _ := cmd.Flags().GetDuration(name)
	return value
}

func openMailProxyDB(databaseURL string) (store.Store, func(), error) {
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL != "" {
		db, err := store.OpenStore(store.StoreConfig{DatabaseURL: databaseURL})
		if err != nil {
			return nil, nil, fmt.Errorf("opening store: %w", err)
		}
		return db, func() { _ = db.Close() }, nil
	}

	dbPath, err := store.DefaultDBPath()
	if err != nil {
		return nil, nil, fmt.Errorf("resolving db path: %w", err)
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening store: %w", err)
	}
	return db, func() { _ = db.Close() }, nil
}

func init() {
	mailProxyCmd.Flags().String("vault", "", "vault containing the Gmail mail service")
	mailProxyCmd.Flags().String("service", "", "service name carrying mail_proxy policy")
	mailProxyCmd.Flags().String("imap-listen", mailproxy.DefaultIMAPListen, "local IMAP implicit TLS listen address")
	mailProxyCmd.Flags().String("smtp-listen", mailproxy.DefaultSMTPListen, "local SMTP STARTTLS listen address")
	mailProxyCmd.Flags().String("imap-upstream", mailproxy.DefaultIMAPUpstream, "upstream Gmail IMAP address")
	mailProxyCmd.Flags().String("smtp-upstream", mailproxy.DefaultSMTPUpstream, "upstream Gmail SMTP address")
	mailProxyCmd.Flags().Duration("shutdown-timeout", 10*time.Second, "graceful shutdown timeout")
	mailProxyCmd.Flags().String("database-url", "", "database URL override")
	mailProxyCmd.Flags().Bool("password-stdin", false, "read master password from stdin")
	rootCmd.AddCommand(mailProxyCmd)
}
