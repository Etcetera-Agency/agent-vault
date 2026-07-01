package mailproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"strings"
	"testing"
)

func TestIMAPCapabilityAndNoop(t *testing.T) {
	client, done := startTestIMAPSession(t, func(opts *IMAPOptions) {})
	defer client.Close()
	defer waitDone(t, done)
	reader := bufio.NewReader(client)

	readLine(t, reader, "* OK")
	writeLine(t, client, "A1 CAPABILITY")
	readLine(t, reader, "* CAPABILITY")
	readLine(t, reader, "A1 OK")
	writeLine(t, client, "A2 NOOP")
	readLine(t, reader, "A2 OK")
	writeLine(t, client, "A3 LOGOUT")
	readLine(t, reader, "* BYE")
	readLine(t, reader, "A3 OK")
}

func TestIMAPMalformedCommand(t *testing.T) {
	client, done := startTestIMAPSession(t, func(opts *IMAPOptions) {})
	defer client.Close()
	defer waitDone(t, done)
	reader := bufio.NewReader(client)

	readLine(t, reader, "* OK")
	writeLine(t, client, "BAD")
	readLine(t, reader, "* BAD malformed command")
	writeLine(t, client, "A1 LOGOUT")
	readLine(t, reader, "* BYE")
	readLine(t, reader, "A1 OK")
}

func TestIMAPBadLocalAuth(t *testing.T) {
	client, done := startTestIMAPSession(t, func(opts *IMAPOptions) {})
	defer client.Close()
	defer waitDone(t, done)
	reader := bufio.NewReader(client)

	readLine(t, reader, "* OK")
	writeLine(t, client, "A1 LOGIN agent@gmail.com wrong")
	readLine(t, reader, "A1 NO authentication failed")
	writeLine(t, client, "A2 LOGOUT")
	readLine(t, reader, "* BYE")
	readLine(t, reader, "A2 OK")
}

func TestIMAPLoginAuthenticatesUpstreamAndRelays(t *testing.T) {
	upstreamServer, upstreamClient := net.Pipe()
	defer upstreamClient.Close()
	authCalls := 0

	client, done := startTestIMAPSession(t, func(opts *IMAPOptions) {
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
	defer client.Close()
	reader := bufio.NewReader(client)

	readLine(t, reader, "* OK")
	writeLine(t, client, "A1 LOGIN agent@gmail.com local-password")
	readLine(t, reader, "A1 OK")
	writeLine(t, client, "A2 SELECT INBOX")
	readClient := bufio.NewReader(upstreamClient)
	if got := readRawLine(t, readClient); got != "A2 SELECT INBOX\r\n" {
		t.Fatalf("relayed upstream = %q", got)
	}
	_, _ = upstreamClient.Write([]byte("* 0 EXISTS\r\nA2 OK SELECT completed\r\n"))
	readLine(t, reader, "* 0 EXISTS")
	readLine(t, reader, "A2 OK")
	_ = client.Close()
	waitDone(t, done)
	if authCalls != 1 {
		t.Fatalf("auth calls = %d, want 1", authCalls)
	}
}

func TestAuthenticateIMAPXOAUTH2(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	errCh := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(client)
		_, _ = client.Write([]byte("* OK Gmail ready\r\n"))
		line := readRawLine(t, reader)
		if !strings.Contains(line, XOAUTH2Base64("agent@gmail.com", "token")) {
			t.Errorf("auth line = %q", line)
		}
		_, _ = client.Write([]byte("A1 OK success\r\n"))
	}()
	go func() {
		errCh <- AuthenticateIMAPXOAUTH2(context.Background(), server, "agent@gmail.com", "token")
	}()
	if err := <-errCh; err != nil {
		t.Fatalf("AuthenticateIMAPXOAUTH2: %v", err)
	}
}

func startTestIMAPSession(t *testing.T, mutate func(*IMAPOptions)) (net.Conn, <-chan error) {
	t.Helper()
	tlsConfig, roots := localTLSConfigForTest(t)
	authenticator, err := NewLocalAuthenticator("agent@gmail.com", []byte("local-password"))
	if err != nil {
		t.Fatalf("NewLocalAuthenticator: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	opts := IMAPOptions{
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
		done <- HandleIMAPSession(context.Background(), serverConn, opts)
	}()

	client := tls.Client(clientConn, &tls.Config{
		RootCAs:    roots,
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS12,
	})
	if err := client.Handshake(); err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	return client, done
}

func readLine(t *testing.T, reader *bufio.Reader, prefix string) string {
	t.Helper()
	line := readRawLine(t, reader)
	if !strings.HasPrefix(strings.TrimRight(line, "\r\n"), prefix) {
		t.Fatalf("line = %q, want prefix %q", line, prefix)
	}
	return line
}

func readRawLine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	return line
}

func writeLine(t *testing.T, conn net.Conn, line string) {
	t.Helper()
	if _, err := conn.Write([]byte(line + "\r\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func waitDone(t *testing.T, done <-chan error) {
	t.Helper()
	err := <-done
	if err != nil && !strings.Contains(err.Error(), "closed") && !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("session: %v", err)
	}
}
