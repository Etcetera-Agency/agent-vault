package mailproxy

import (
	"context"
	"strings"
	"testing"

	"github.com/Infisical/agent-vault/internal/crypto"
	"github.com/Infisical/agent-vault/internal/store"
)

type fakeCredentialStore struct {
	credential *store.Credential
}

func (f *fakeCredentialStore) GetCredential(_ context.Context, _, _ string) (*store.Credential, error) {
	return f.credential, nil
}

func TestLoadLocalPasswordDecryptsStaticCredential(t *testing.T) {
	encKey := localAuthTestKey()
	ct, nonce, err := crypto.Encrypt([]byte("local-password"), encKey)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	store := &fakeCredentialStore{credential: &store.Credential{Type: "static", Ciphertext: ct, Nonce: nonce}}

	password, err := LoadLocalPassword(context.Background(), store, "vault", "LOCAL_PASSWORD", encKey)
	if err != nil {
		t.Fatalf("LoadLocalPassword: %v", err)
	}
	if string(password) != "local-password" {
		t.Fatalf("password = %q", password)
	}
}

func TestLoadLocalPasswordRejectsEmptyPassword(t *testing.T) {
	encKey := localAuthTestKey()
	ct, nonce, err := crypto.Encrypt(nil, encKey)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	store := &fakeCredentialStore{credential: &store.Credential{Type: "static", Ciphertext: ct, Nonce: nonce}}

	_, err = LoadLocalPassword(context.Background(), store, "vault", "LOCAL_PASSWORD", encKey)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("err = %v, want empty password rejection", err)
	}
}

func TestLocalAuthenticatorVerify(t *testing.T) {
	authenticator, err := NewLocalAuthenticator("agent@gmail.com", []byte("local-password"))
	if err != nil {
		t.Fatalf("NewLocalAuthenticator: %v", err)
	}
	if !authenticator.Verify("agent@gmail.com", []byte("local-password")) {
		t.Fatal("valid local credentials rejected")
	}
	if authenticator.Verify("agent@gmail.com", []byte("wrong")) {
		t.Fatal("invalid password accepted")
	}
	if authenticator.Verify("other@gmail.com", []byte("local-password")) {
		t.Fatal("invalid email accepted")
	}
	if GenericAuthFailure() != "authentication failed" {
		t.Fatalf("generic failure = %q", GenericAuthFailure())
	}
}

func TestNewLocalAuthenticatorRejectsEmptyPassword(t *testing.T) {
	_, err := NewLocalAuthenticator("agent@gmail.com", nil)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("err = %v, want empty password rejection", err)
	}
}

func localAuthTestKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x24
	}
	return key
}
