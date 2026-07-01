package mailproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
)

type IMAPOptions struct {
	TLSConfig     *tls.Config
	Authenticator *LocalAuthenticator
	Email         string
	TokenProvider TokenProvider
	UpstreamDial  func(context.Context) (net.Conn, error)
	UpstreamAuth  func(context.Context, net.Conn, string, string) error
}

func HandleIMAPSession(ctx context.Context, conn net.Conn, opts IMAPOptions) error {
	local := WrapImplicitTLS(conn, opts.TLSConfig)
	if err := local.Handshake(); err != nil {
		_ = local.Close()
		return err
	}

	reader := bufio.NewReader(local)
	writer := bufio.NewWriter(local)
	if err := writeIMAP(writer, "* OK Agent Vault IMAP proxy ready"); err != nil {
		return err
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		tag, command, args, ok := parseIMAPCommand(line)
		if !ok {
			if err := writeIMAP(writer, "* BAD malformed command"); err != nil {
				return err
			}
			continue
		}

		switch command {
		case "CAPABILITY":
			if err := writeIMAP(writer, "* CAPABILITY IMAP4rev1 AUTH=XOAUTH2"); err != nil {
				return err
			}
			if err := writeIMAP(writer, tag+" OK CAPABILITY completed"); err != nil {
				return err
			}
		case "NOOP":
			if err := writeIMAP(writer, tag+" OK NOOP completed"); err != nil {
				return err
			}
		case "LOGOUT":
			if err := writeIMAP(writer, "* BYE Agent Vault IMAP proxy closing"); err != nil {
				return err
			}
			return writeIMAP(writer, tag+" OK LOGOUT completed")
		case "LOGIN":
			if len(args) < 2 {
				if err := writeIMAP(writer, tag+" BAD malformed LOGIN"); err != nil {
					return err
				}
				continue
			}
			if !opts.Authenticator.Verify(unquoteIMAPAtom(args[0]), []byte(unquoteIMAPAtom(args[1]))) {
				if err := writeIMAP(writer, tag+" NO "+GenericAuthFailure()); err != nil {
					return err
				}
				continue
			}
			upstreamDial := opts.UpstreamDial
			if upstreamDial == nil {
				upstreamDial = func(ctx context.Context) (net.Conn, error) {
					return DialIMAPUpstream(ctx, DefaultIMAPUpstream)
				}
			}
			upstream, err := upstreamDial(ctx)
			if err != nil {
				return err
			}
			authFunc := opts.UpstreamAuth
			if authFunc == nil {
				authFunc = AuthenticateIMAPXOAUTH2
			}
			if err := WithForcedRefreshRetry(ctx, opts.TokenProvider, func(token string) error {
				return authFunc(ctx, upstream, opts.Email, token)
			}); err != nil {
				_ = upstream.Close()
				return err
			}
			if err := writeIMAP(writer, tag+" OK LOGIN completed"); err != nil {
				_ = upstream.Close()
				return err
			}
			Relay(local, upstream)
			return nil
		default:
			if err := writeIMAP(writer, tag+" BAD unsupported command"); err != nil {
				return err
			}
		}
	}
}

func DialIMAPUpstream(ctx context.Context, address string) (net.Conn, error) {
	dialer := tls.Dialer{
		Config: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: strings.Split(address, ":")[0],
		},
	}
	return dialer.DialContext(ctx, "tcp", address)
}

func AuthenticateIMAPXOAUTH2(_ context.Context, conn net.Conn, email, token string) error {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		return err
	}
	if err := writeIMAP(writer, "A1 AUTHENTICATE XOAUTH2 "+XOAUTH2Base64(email, token)); err != nil {
		return err
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	upper := strings.ToUpper(line)
	if strings.Contains(upper, " OK ") || strings.HasPrefix(upper, "A1 OK") {
		return nil
	}
	return ErrXOAUTH2Rejected
}

func parseIMAPCommand(line string) (tag string, command string, args []string, ok bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return "", "", nil, false
	}
	return fields[0], strings.ToUpper(fields[1]), fields[2:], true
}

func unquoteIMAPAtom(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`)
	}
	return value
}

func writeIMAP(writer *bufio.Writer, line string) error {
	if _, err := fmt.Fprintf(writer, "%s\r\n", line); err != nil {
		return err
	}
	return writer.Flush()
}
