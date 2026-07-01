package mailproxy

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/Infisical/agent-vault/internal/crypto"
	"github.com/Infisical/agent-vault/internal/store"
)

type CredentialStore interface {
	GetCredential(ctx context.Context, vaultID, key string) (*store.Credential, error)
}

type LocalAuthenticator struct {
	email        string
	passwordHash [sha256.Size]byte
}

func LoadLocalPassword(ctx context.Context, store CredentialStore, vaultID, key string, encKey []byte) ([]byte, error) {
	credential, err := store.GetCredential(ctx, vaultID, key)
	if err != nil || credential == nil {
		return nil, fmt.Errorf("local password credential %q not found", key)
	}
	if credential.Type == "oauth" {
		return nil, fmt.Errorf("local password credential %q must be static", key)
	}

	password, err := crypto.Decrypt(credential.Ciphertext, credential.Nonce, encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt local password credential %q", key)
	}
	if len(password) == 0 {
		return nil, fmt.Errorf("local password credential %q is empty", key)
	}
	return password, nil
}

func LoadOAuthAccessToken(ctx context.Context, store CredentialStore, vaultID, key string, encKey []byte) (string, error) {
	credential, err := store.GetCredential(ctx, vaultID, key)
	if err != nil || credential == nil {
		return "", fmt.Errorf("oauth credential %q not found", key)
	}
	if credential.Type != "oauth" {
		return "", fmt.Errorf("credential %q must be oauth", key)
	}
	token, err := crypto.Decrypt(credential.Ciphertext, credential.Nonce, encKey)
	if err != nil {
		return "", fmt.Errorf("decrypt oauth credential %q", key)
	}
	if len(token) == 0 {
		return "", fmt.Errorf("oauth credential %q is not connected", key)
	}
	return string(token), nil
}

func NewLocalAuthenticator(email string, password []byte) (*LocalAuthenticator, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if len(password) == 0 {
		return nil, fmt.Errorf("local password is empty")
	}
	return &LocalAuthenticator{
		email:        email,
		passwordHash: sha256.Sum256(password),
	}, nil
}

func (a *LocalAuthenticator) Verify(email string, password []byte) bool {
	if a == nil || len(password) == 0 {
		return false
	}

	passwordHash := sha256.Sum256(password)
	emailOK := subtle.ConstantTimeCompare([]byte(email), []byte(a.email)) == 1
	passwordOK := subtle.ConstantTimeCompare(passwordHash[:], a.passwordHash[:]) == 1
	return emailOK && passwordOK
}

func GenericAuthFailure() string {
	return "authentication failed"
}
