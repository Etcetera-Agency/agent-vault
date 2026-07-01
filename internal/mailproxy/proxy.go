package mailproxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type Proxy struct {
	Config        Config
	Policy        *MailProxyPolicy
	TLSConfig     *tls.Config
	Authenticator *LocalAuthenticator
	TokenProvider TokenProvider
}

type MailProxyPolicy struct {
	IMAP bool
	SMTP bool
}

func (p *Proxy) Run(ctx context.Context) error {
	if p.Policy == nil || (!p.Policy.IMAP && !p.Policy.SMTP) {
		return fmt.Errorf("no enabled mail proxy protocol")
	}

	var listeners []net.Listener
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	if p.Policy.IMAP {
		listener, err := net.Listen("tcp", p.Config.IMAPListen)
		if err != nil {
			return fmt.Errorf("listen imap: %w", err)
		}
		listeners = append(listeners, listener)
		wg.Add(1)
		go p.acceptIMAP(ctx, listener, &wg, errCh)
	}
	if p.Policy.SMTP {
		listener, err := net.Listen("tcp", p.Config.SMTPListen)
		if err != nil {
			closeListeners(listeners)
			return fmt.Errorf("listen smtp: %w", err)
		}
		listeners = append(listeners, listener)
		wg.Add(1)
		go p.acceptSMTP(ctx, listener, &wg, errCh)
	}

	select {
	case <-ctx.Done():
	case err := <-errCh:
		closeListeners(listeners)
		return err
	}

	closeListeners(listeners)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timeout := p.Config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("mail proxy shutdown timeout")
	}
}

func (p *Proxy) acceptIMAP(ctx context.Context, listener net.Listener, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			errCh <- err
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = HandleIMAPSession(ctx, conn, IMAPOptions{
				TLSConfig:     p.TLSConfig,
				Authenticator: p.Authenticator,
				Email:         p.ConfiguredEmail(),
				TokenProvider: p.TokenProvider,
				UpstreamDial: func(ctx context.Context) (net.Conn, error) {
					return DialIMAPUpstream(ctx, p.Config.IMAPUpstream)
				},
			})
		}()
	}
}

func (p *Proxy) acceptSMTP(ctx context.Context, listener net.Listener, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			errCh <- err
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = HandleSMTPSession(ctx, conn, SMTPOptions{
				TLSConfig:     p.TLSConfig,
				Authenticator: p.Authenticator,
				Email:         p.ConfiguredEmail(),
				TokenProvider: p.TokenProvider,
				UpstreamDial: func(ctx context.Context) (net.Conn, error) {
					return DialSMTPUpstream(ctx, p.Config.SMTPUpstream)
				},
			})
		}()
	}
}

func (p *Proxy) ConfiguredEmail() string {
	if p.Authenticator == nil {
		return ""
	}
	return p.Authenticator.email
}

func closeListeners(listeners []net.Listener) {
	for _, listener := range listeners {
		_ = listener.Close()
	}
}
