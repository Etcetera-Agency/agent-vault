package mailproxy

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/Infisical/agent-vault/internal/ca"
)

func TestLocalTLSConfigUsesAgentVaultCAForLoopbackLeaf(t *testing.T) {
	provider, err := ca.New(localTLSTestKey(), ca.Options{Dir: t.TempDir(), LeafTTL: time.Hour})
	if err != nil {
		t.Fatalf("ca.New: %v", err)
	}

	tlsConfig, err := LocalTLSConfig(provider)
	if err != nil {
		t.Fatalf("LocalTLSConfig: %v", err)
	}
	if len(tlsConfig.Certificates) != 1 {
		t.Fatalf("certificates = %d, want 1", len(tlsConfig.Certificates))
	}
	cert := tlsConfig.Certificates[0].Leaf
	if cert == nil {
		t.Fatal("leaf certificate is nil")
	}
	if !slices.ContainsFunc(cert.IPAddresses, func(ip net.IP) bool {
		return ip.Equal(net.ParseIP("127.0.0.1"))
	}) {
		t.Fatalf("IP SANs = %v, want 127.0.0.1", cert.IPAddresses)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(provider.RootPEM()) {
		t.Fatal("AppendCertsFromPEM failed")
	}
	if _, err := cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSName:   "127.0.0.1",
	}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestSMTPLocalCapabilitiesAdvertiseStartTLS(t *testing.T) {
	capabilities := SMTPLocalCapabilities()
	if !slices.Contains(capabilities, "STARTTLS") {
		t.Fatalf("capabilities = %v, want STARTTLS", capabilities)
	}
}

func TestWrapImplicitTLSCompletesHandshake(t *testing.T) {
	tlsConfig, roots := localTLSConfigForTest(t)
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- WrapImplicitTLS(serverConn, tlsConfig).Handshake()
	}()

	client := tls.Client(clientConn, &tls.Config{
		RootCAs:    roots,
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS12,
	})
	if err := client.Handshake(); err != nil {
		t.Fatalf("client Handshake: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server Handshake: %v", err)
	}
}

func TestUpgradeStartTLSCompletesHandshake(t *testing.T) {
	tlsConfig, roots := localTLSConfigForTest(t)
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- UpgradeStartTLS(serverConn, tlsConfig).Handshake()
	}()

	client := tls.Client(clientConn, &tls.Config{
		RootCAs:    roots,
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS12,
	})
	if err := client.Handshake(); err != nil {
		t.Fatalf("client Handshake: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("server Handshake: %v", err)
	}
}

func localTLSConfigForTest(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	provider, err := ca.New(localTLSTestKey(), ca.Options{Dir: t.TempDir(), LeafTTL: time.Hour})
	if err != nil {
		t.Fatalf("ca.New: %v", err)
	}
	tlsConfig, err := LocalTLSConfig(provider)
	if err != nil {
		t.Fatalf("LocalTLSConfig: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(provider.RootPEM()) {
		t.Fatal("AppendCertsFromPEM failed")
	}
	return tlsConfig, roots
}

func localTLSTestKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x36
	}
	return key
}
