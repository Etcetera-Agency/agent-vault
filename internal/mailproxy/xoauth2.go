package mailproxy

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/Infisical/agent-vault/internal/oauthcredential"
)

var ErrXOAUTH2Rejected = errors.New("xoauth2 authentication rejected")

type TokenProvider interface {
	AccessToken(ctx context.Context) (string, error)
	ForceRefresh(ctx context.Context) (string, error)
}

type VaultOAuthTokenProvider struct {
	Resolver      *oauthcredential.Resolver
	VaultID       string
	CredentialKey string
	CurrentToken  string
}

func (p *VaultOAuthTokenProvider) AccessToken(ctx context.Context) (string, error) {
	token, err := p.resolve(ctx, false)
	if err != nil {
		return "", err
	}
	p.CurrentToken = token
	return token, nil
}

func (p *VaultOAuthTokenProvider) ForceRefresh(ctx context.Context) (string, error) {
	token, err := p.resolve(ctx, true)
	if err != nil {
		return "", err
	}
	p.CurrentToken = token
	return token, nil
}

func (p *VaultOAuthTokenProvider) resolve(ctx context.Context, force bool) (string, error) {
	return p.Resolver.Resolve(ctx, p.VaultID, p.CredentialKey, p.CurrentToken, oauthcredential.ResolveOptions{
		ForceRefresh: force,
	})
}

func XOAUTH2Payload(email, accessToken string) []byte {
	return []byte("user=" + email + "\x01auth=Bearer " + accessToken + "\x01\x01")
}

func XOAUTH2Base64(email, accessToken string) string {
	return base64.StdEncoding.EncodeToString(XOAUTH2Payload(email, accessToken))
}

func WithForcedRefreshRetry(ctx context.Context, provider TokenProvider, authenticate func(token string) error) error {
	token, err := provider.AccessToken(ctx)
	if err != nil {
		return err
	}

	err = authenticate(token)
	if !errors.Is(err, ErrXOAUTH2Rejected) {
		return err
	}

	token, err = provider.ForceRefresh(ctx)
	if err != nil {
		return err
	}
	return authenticate(token)
}
