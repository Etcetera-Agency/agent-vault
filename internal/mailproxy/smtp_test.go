package mailproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"net"
	"strings"
	"testing"
)

func TestSMTPEhloStartTLSAndAuthPlainRelays(t *testing.T) {
	upstreamServer, upstreamClient := net.Pipe()
	defer upstreamClient.Close()
	authCalls := 0

	client, done := startTestSMTPSession(t, func(opts *SMTPOptions) {
		opts.UpstreamDial = func(context.Context) (net.Conn, error) {
			return upstreamServer, nil
		}
		opts.UpstreamAuth = func(_ context.Context, conn net.Conn, email, token string) error {
			authCalls++
			if email != "agent@gmail.com" || token != "old-token" {
				t.Fatalf("upstream auth = %s/%s", email, token)
			}
			return nil
		}
	})
	reader := bufio.NewReader(client)

	readSMTPLine(t, reader, "220")
	writeLine(t, client, "EHLO localhost")
	readSMTPLine(t, reader, "250-localhost")
	readSMTPLine(t, reader, "250-STARTTLS")
	readSMTPLine(t, reader, "250 AUTH")
	writeLine(t, client, "STARTTLS")
	readSMTPLine(t, reader, "220")

	tlsClient := tls.Client(client, &tls.Config{
		RootCAs:    localSMTPRoots(t),
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsClient.Handshake(); err != nil {
		t.Fatalf("Handshake: %v", err)
	}
	reader = bufio.NewReader(tlsClient)
	writeLine(t, tlsClient, "EHLO localhost")
	readSMTPLine(t, reader, "250-localhost")
	readSMTPLine(t, reader, "250 AUTH")

	plain := base64.StdEncoding.EncodeToString([]byte("\x00agent@gmail.com\x00local-password"))
	writeLine(t, tlsClient, "AUTH PLAIN "+plain)
	readSMTPLine(t, reader, "235")
	writeLine(t, tlsClient, "MAIL FROM:<agent@gmail.com>")

	upstreamReader := bufio.NewReader(upstreamClient)
	if got := readRawLine(t, upstreamReader); got != "MAIL FROM:<agent@gmail.com>\r\n" {
		t.Fatalf("relayed upstream = %q", got)
	}
	_, _ = upstreamClient.Write([]byte("250 OK\r\n"))
	readSMTPLine(t, reader, "250")
	_ = tlsClient.Close()
	waitDone(t, done)
	if authCalls != 1 {
		t.Fatalf("auth calls = %d, want 1", authCalls)
	}
}

func TestSMTPAuthLoginBadLocalAuth(t *testing.T) {
	client, done := startTestSMTPSession(t, func(opts *SMTPOptions) {})
	defer client.Close()
	defer waitDone(t, done)
	reader := bufio.NewReader(client)

	readSMTPLine(t, reader, "220")
	writeLine(t, client, "STARTTLS")
	readSMTPLine(t, reader, "220")
	tlsClient := tls.Client(client, &tls.Config{
		RootCAs:    localSMTPRoots(t),
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsClient.Handshake(); err != nil {
		t.Fatalf("Handshake: %v", err)
	}
	reader = bufio.NewReader(tlsClient)
	writeLine(t, tlsClient, "AUTH LOGIN "+base64.StdEncoding.EncodeToString([]byte("agent@gmail.com")))
	readSMTPLine(t, reader, "334")
	writeLine(t, tlsClient, base64.StdEncoding.EncodeToString([]byte("wrong")))
	readSMTPLine(t, reader, "535 authentication failed")
	_ = tlsClient.Close()
}

func TestAuthenticateSMTPXOAUTH2RejectsNon235(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	restore := setTestSMTPUpstreamTLSConfig()
	defer restore()

	go fakeSMTPUpstream(t, client, "535 rejected")
	err := AuthenticateSMTPXOAUTH2(context.Background(), server, "agent@gmail.com", "token")
	if err != ErrXOAUTH2Rejected {
		t.Fatalf("err = %v, want ErrXOAUTH2Rejected", err)
	}
}

func startTestSMTPSession(t *testing.T, mutate func(*SMTPOptions)) (net.Conn, <-chan error) {
	t.Helper()
	tlsConfig, roots := localTLSConfigForTest(t)
	smtpTestRoots = roots
	authenticator, err := NewLocalAuthenticator("agent@gmail.com", []byte("local-password"))
	if err != nil {
		t.Fatalf("NewLocalAuthenticator: %v", err)
	}
	serverConn, clientConn := net.Pipe()
	opts := SMTPOptions{
		TLSConfig:     tlsConfig,
		Authenticator: authenticator,
		Email:         "agent@gmail.com",
		TokenProvider: &fakeTokenProvider{accessToken: "old-token", forcedToken: "new-token"},
		UpstreamDial: func(context.Context) (net.Conn, error) {
			server, _ := net.Pipe()
			return server, nil
		},
		UpstreamAuth: func(context.Context, net.Conn, string, string) error {
			return nil
		},
	}
	mutate(&opts)
	done := make(chan error, 1)
	go func() {
		done <- HandleSMTPSession(context.Background(), serverConn, opts)
	}()
	return clientConn, done
}

func fakeSMTPUpstream(t *testing.T, conn net.Conn, authResponse string) {
	t.Helper()
	defer conn.Close()
	reader := bufio.NewReader(conn)
	_, _ = conn.Write([]byte("220 smtp.gmail.com\r\n"))
	readSMTPLine(t, reader, "EHLO")
	_, _ = conn.Write([]byte("250-smtp.gmail.com\r\n250 STARTTLS\r\n"))
	readSMTPLine(t, reader, "STARTTLS")
	_, _ = conn.Write([]byte("220 ready\r\n"))
	tlsServer := tls.Server(conn, smtpTestServerTLSConfig(t))
	if err := tlsServer.Handshake(); err != nil {
		t.Errorf("server handshake: %v", err)
		return
	}
	reader = bufio.NewReader(tlsServer)
	readSMTPLine(t, reader, "EHLO")
	_, _ = tlsServer.Write([]byte("250-smtp.gmail.com\r\n250 AUTH XOAUTH2\r\n"))
	line := readRawLine(t, reader)
	if !strings.Contains(line, XOAUTH2Base64("agent@gmail.com", "token")) {
		t.Errorf("auth line = %q", line)
	}
	_, _ = tlsServer.Write([]byte(authResponse + "\r\n"))
}

func readSMTPLine(t *testing.T, reader *bufio.Reader, prefix string) string {
	t.Helper()
	line := readRawLine(t, reader)
	if !strings.HasPrefix(strings.TrimRight(line, "\r\n"), prefix) {
		t.Fatalf("line = %q, want prefix %q", line, prefix)
	}
	return line
}

var smtpTestRoots interface{}

func localSMTPRoots(t *testing.T) *x509.CertPool {
	t.Helper()
	roots, ok := smtpTestRoots.(*x509.CertPool)
	if !ok {
		t.Fatal("smtp roots not initialized")
	}
	return roots
}

func smtpTestServerTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	tlsConfig, _ := localTLSConfigForTest(t)
	return tlsConfig
}

func setTestSMTPUpstreamTLSConfig() func() {
	previous := smtpUpstreamTLSConfig
	smtpUpstreamTLSConfig = func() *tls.Config {
		return &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12} //nolint:gosec // test-only fake upstream
	}
	return func() {
		smtpUpstreamTLSConfig = previous
	}
}
