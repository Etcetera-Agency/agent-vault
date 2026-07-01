package mailproxy

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/Infisical/agent-vault/internal/ca"
)

func LocalTLSConfig(provider ca.Provider) (*tls.Config, error) {
	if provider == nil {
		return nil, fmt.Errorf("CA provider is required")
	}

	cert, err := provider.MintLeaf("127.0.0.1")
	if err != nil {
		return nil, fmt.Errorf("mint local TLS certificate: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{*cert},
	}, nil
}

func SMTPLocalCapabilities() []string {
	return []string{
		"STARTTLS",
		"AUTH PLAIN LOGIN",
	}
}

func WrapImplicitTLS(conn net.Conn, config *tls.Config) *tls.Conn {
	return tls.Server(conn, config)
}

func UpgradeStartTLS(conn net.Conn, config *tls.Config) *tls.Conn {
	return tls.Server(conn, config)
}
