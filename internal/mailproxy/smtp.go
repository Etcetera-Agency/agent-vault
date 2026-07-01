package mailproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
)

type SMTPOptions struct {
	TLSConfig     *tls.Config
	Authenticator *LocalAuthenticator
	Email         string
	TokenProvider TokenProvider
	UpstreamDial  func(context.Context) (net.Conn, error)
	UpstreamAuth  func(context.Context, net.Conn, string, string) error
}

func HandleSMTPSession(ctx context.Context, conn net.Conn, opts SMTPOptions) error {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	tlsActive := false

	if err := writeSMTP(writer, "220 Agent Vault SMTP proxy ready"); err != nil {
		return err
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		command, args := parseSMTPCommand(line)

		switch command {
		case "EHLO", "HELO":
			if err := writeSMTPCapabilities(writer, tlsActive); err != nil {
				return err
			}
		case "NOOP", "RSET":
			if err := writeSMTP(writer, "250 OK"); err != nil {
				return err
			}
		case "QUIT":
			return writeSMTP(writer, "221 Bye")
		case "STARTTLS":
			if tlsActive {
				if err := writeSMTP(writer, "503 TLS already active"); err != nil {
					return err
				}
				continue
			}
			if err := writeSMTP(writer, "220 Ready to start TLS"); err != nil {
				return err
			}
			tlsConn := UpgradeStartTLS(conn, opts.TLSConfig)
			if err := tlsConn.Handshake(); err != nil {
				return err
			}
			conn = tlsConn
			reader = bufio.NewReader(tlsConn)
			writer = bufio.NewWriter(tlsConn)
			tlsActive = true
		case "AUTH":
			if !tlsActive {
				if err := writeSMTP(writer, "530 Must issue STARTTLS first"); err != nil {
					return err
				}
				continue
			}
			if err := handleSMTPAuth(ctx, conn, reader, writer, opts, args); err != nil {
				return err
			}
			return nil
		default:
			if err := writeSMTP(writer, "502 Command not implemented"); err != nil {
				return err
			}
		}
	}
}

func handleSMTPAuth(ctx context.Context, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer, opts SMTPOptions, args string) error {
	email, password, ok := parseSMTPAuth(args, reader, writer)
	if !ok || !opts.Authenticator.Verify(email, []byte(password)) {
		return writeSMTP(writer, "535 "+GenericAuthFailure())
	}

	upstreamDial := opts.UpstreamDial
	if upstreamDial == nil {
		upstreamDial = func(ctx context.Context) (net.Conn, error) {
			return DialSMTPUpstream(ctx, DefaultSMTPUpstream)
		}
	}
	upstream, err := upstreamDial(ctx)
	if err != nil {
		return err
	}
	authFunc := opts.UpstreamAuth
	if authFunc == nil {
		authFunc = AuthenticateSMTPXOAUTH2
	}
	if err := WithForcedRefreshRetry(ctx, opts.TokenProvider, func(token string) error {
		return authFunc(ctx, upstream, opts.Email, token)
	}); err != nil {
		_ = upstream.Close()
		return err
	}
	if err := writeSMTP(writer, "235 Authentication successful"); err != nil {
		_ = upstream.Close()
		return err
	}
	Relay(conn, upstream)
	return nil
}

func AuthenticateSMTPXOAUTH2(_ context.Context, conn net.Conn, email, token string) error {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	if _, err := reader.ReadString('\n'); err != nil {
		return err
	}
	if err := writeSMTP(writer, "EHLO localhost"); err != nil {
		return err
	}
	if err := readSMTPPositive(reader, "250"); err != nil {
		return err
	}
	if err := writeSMTP(writer, "STARTTLS"); err != nil {
		return err
	}
	if err := readSMTPPositive(reader, "220"); err != nil {
		return err
	}

	tlsConn := tls.Client(conn, smtpUpstreamTLSConfig())
	reader = bufio.NewReader(tlsConn)
	writer = bufio.NewWriter(tlsConn)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	if err := writeSMTP(writer, "EHLO localhost"); err != nil {
		return err
	}
	if err := readSMTPPositive(reader, "250"); err != nil {
		return err
	}
	if err := writeSMTP(writer, "AUTH XOAUTH2 "+XOAUTH2Base64(email, token)); err != nil {
		return err
	}
	if err := readSMTPPositive(reader, "235"); err != nil {
		return ErrXOAUTH2Rejected
	}
	return nil
}

func DialSMTPUpstream(ctx context.Context, address string) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, "tcp", address)
}

func parseSMTPCommand(line string) (command string, args string) {
	line = strings.TrimRight(line, "\r\n")
	parts := strings.SplitN(line, " ", 2)
	command = strings.ToUpper(parts[0])
	if len(parts) == 2 {
		args = strings.TrimSpace(parts[1])
	}
	return command, args
}

func parseSMTPAuth(args string, reader *bufio.Reader, writer *bufio.Writer) (email string, password string, ok bool) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "", "", false
	}

	switch strings.ToUpper(parts[0]) {
	case "PLAIN":
		if len(parts) < 2 {
			return "", "", false
		}
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", "", false
		}
		fields := strings.Split(string(decoded), "\x00")
		if len(fields) != 3 {
			return "", "", false
		}
		return fields[1], fields[2], true
	case "LOGIN":
		var user string
		if len(parts) >= 2 {
			decoded, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				return "", "", false
			}
			user = string(decoded)
		} else {
			if err := writeSMTP(writer, "334 VXNlcm5hbWU6"); err != nil {
				return "", "", false
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				return "", "", false
			}
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(line))
			if err != nil {
				return "", "", false
			}
			user = string(decoded)
		}
		if err := writeSMTP(writer, "334 UGFzc3dvcmQ6"); err != nil {
			return "", "", false
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", "", false
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(line))
		if err != nil {
			return "", "", false
		}
		return user, string(decoded), true
	default:
		return "", "", false
	}
}

func writeSMTPCapabilities(writer *bufio.Writer, tlsActive bool) error {
	if err := writeSMTP(writer, "250-localhost"); err != nil {
		return err
	}
	if !tlsActive {
		if err := writeSMTP(writer, "250-STARTTLS"); err != nil {
			return err
		}
	}
	return writeSMTP(writer, "250 AUTH PLAIN LOGIN")
}

func readSMTPPositive(reader *bufio.Reader, code string) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, code+" ") {
			return nil
		}
		if !strings.HasPrefix(line, code+"-") {
			return fmt.Errorf("smtp upstream returned %q", strings.TrimSpace(line))
		}
	}
}

func writeSMTP(writer *bufio.Writer, line string) error {
	if _, err := fmt.Fprintf(writer, "%s\r\n", line); err != nil {
		return err
	}
	return writer.Flush()
}

var smtpUpstreamTLSConfig = func() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: "smtp.gmail.com",
	}
}
