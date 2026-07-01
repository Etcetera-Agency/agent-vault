package mailproxy

import (
	"bufio"
	"context"
	"net"
	"strings"
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
	proxy := newTestProxy(t, &MailProxyPolicy{IMAP: true}, Config{
		IMAPListen:      freeLoopbackAddr(t),
		IMAPUpstream:    DefaultIMAPUpstream,
		ShutdownTimeout: time.Second,
	})
	ctx, cancel := context.WithCancel(context.Background())

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

func TestProxyRunClosesActiveIMAPAfterShutdownTimeout(t *testing.T) {
	proxy := newTestProxy(t, &MailProxyPolicy{IMAP: true}, Config{
		IMAPListen:      freeLoopbackAddr(t),
		IMAPUpstream:    DefaultIMAPUpstream,
		ShutdownTimeout: 20 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.Run(ctx)
	}()

	conn := dialUntilReady(t, proxy.Config.IMAPListen)
	defer conn.Close()
	waitForActiveConnection(t, proxy)

	cancel()
	err := waitProxyDone(t, errCh)
	if err == nil || !strings.Contains(err.Error(), "shutdown timeout") {
		t.Fatalf("Run error = %v, want shutdown timeout", err)
	}
}

func TestProxyRunClosesActiveSMTPAfterShutdownTimeout(t *testing.T) {
	proxy := newTestProxy(t, &MailProxyPolicy{SMTP: true}, Config{
		SMTPListen:      freeLoopbackAddr(t),
		SMTPUpstream:    DefaultSMTPUpstream,
		ShutdownTimeout: 20 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.Run(ctx)
	}()

	conn := dialUntilReady(t, proxy.Config.SMTPListen)
	defer conn.Close()
	readSMTPLine(t, bufio.NewReader(conn), "220")

	cancel()
	err := waitProxyDone(t, errCh)
	if err == nil || !strings.Contains(err.Error(), "shutdown timeout") {
		t.Fatalf("Run error = %v, want shutdown timeout", err)
	}
}

func newTestProxy(t *testing.T, policy *MailProxyPolicy, cfg Config) *Proxy {
	t.Helper()
	authenticator, err := NewLocalAuthenticator("agent@gmail.com", []byte("local-password"))
	if err != nil {
		t.Fatalf("NewLocalAuthenticator: %v", err)
	}
	tlsConfig, _ := localTLSConfigForTest(t)
	return &Proxy{
		Config:        cfg,
		Policy:        policy,
		TLSConfig:     tlsConfig,
		Authenticator: authenticator,
		TokenProvider: &fakeTokenProvider{accessToken: "old-token"},
	}
}

func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close free port listener: %v", err)
	}
	return addr
}

func dialUntilReady(t *testing.T, addr string) net.Conn {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			return conn
		}
		if time.Now().After(deadline) {
			t.Fatalf("dial %s: %v", addr, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func waitProxyDone(t *testing.T, errCh <-chan error) error {
	t.Helper()
	select {
	case err := <-errCh:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("proxy did not stop")
		return nil
	}
}

func waitForActiveConnection(t *testing.T, proxy *Proxy) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		proxy.mu.Lock()
		active := len(proxy.active)
		proxy.mu.Unlock()
		if active > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("proxy did not track active connection")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
