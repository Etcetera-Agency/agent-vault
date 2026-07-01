package mailproxy

import (
	"context"
	"crypto/tls"
	"testing"
	"time"
)

func TestProxyRunRejectsBothProtocolsDisabled(t *testing.T) {
	proxy := &Proxy{Policy: &MailProxyPolicy{}, Config: Config{ShutdownTimeout: time.Millisecond}}
	err := proxy.Run(context.Background())
	if err == nil {
		t.Fatal("expected disabled protocol error")
	}
}

func TestProxyRunClosesListenerOnCancellation(t *testing.T) {
	authenticator, err := NewLocalAuthenticator("agent@gmail.com", []byte("local-password"))
	if err != nil {
		t.Fatalf("NewLocalAuthenticator: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	proxy := &Proxy{
		Config: Config{
			IMAPListen:      "127.0.0.1:0",
			IMAPUpstream:    DefaultIMAPUpstream,
			ShutdownTimeout: time.Second,
		},
		Policy:        &MailProxyPolicy{IMAP: true},
		TLSConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		Authenticator: authenticator,
		TokenProvider: &fakeTokenProvider{accessToken: "old-token"},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.Run(ctx)
	}()
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("proxy did not stop after cancellation")
	}
}
