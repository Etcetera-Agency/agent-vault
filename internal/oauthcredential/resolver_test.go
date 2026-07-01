package oauthcredential

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Infisical/agent-vault/internal/crypto"
	"github.com/Infisical/agent-vault/internal/oauth"
	"github.com/Infisical/agent-vault/internal/store"
)

type fakeOAuthStore struct {
	oauthCfg *store.CredentialOAuth

	updateAccessToken  string
	updateRefreshToken string
	updateExpiresAt    *time.Time
	updateCalls        int

	errorMessage string
	errorCalls   int
}

func (f *fakeOAuthStore) GetCredentialOAuth(_ context.Context, _, _ string) (*store.CredentialOAuth, error) {
	if f.oauthCfg == nil {
		return nil, errors.New("not found")
	}
	return f.oauthCfg, nil
}

func (f *fakeOAuthStore) UpdateCredentialOAuthTokens(_ context.Context, _, _ string, accessCT, accessNonce, refreshCT, refreshNonce []byte, expiresAt *time.Time) error {
	f.updateCalls++
	f.updateExpiresAt = expiresAt

	accessToken, err := crypto.Decrypt(accessCT, accessNonce, testKey())
	if err != nil {
		return err
	}
	f.updateAccessToken = string(accessToken)

	if refreshCT != nil {
		refreshToken, err := crypto.Decrypt(refreshCT, refreshNonce, testKey())
		if err != nil {
			return err
		}
		f.updateRefreshToken = string(refreshToken)
	}

	return nil
}

func (f *fakeOAuthStore) UpdateCredentialOAuthError(_ context.Context, _, _, errMsg string) error {
	f.errorCalls++
	f.errorMessage = errMsg
	return nil
}

func TestResolverResolve_ReturnsValidTokenWithoutRefresh(t *testing.T) {
	expiresAt := time.Now().Add(10 * time.Minute)
	store := &fakeOAuthStore{oauthCfg: &store.CredentialOAuth{TokenExpiresAt: &expiresAt}}

	resolver := NewResolver(store, testKey(), oauth.NewRefresher())
	token, err := resolver.Resolve(context.Background(), "vault", "GOOGLE", "current", ResolveOptions{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if token != "current" {
		t.Fatalf("token = %q, want current", token)
	}
	if store.updateCalls != 0 {
		t.Fatalf("refresh updates = %d, want 0", store.updateCalls)
	}
}

func TestResolverResolve_RefreshesWithinFiveMinuteBufferAndPersistsRotation(t *testing.T) {
	var refreshCalls int
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls++
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		assertFormValue(t, r.Form, "grant_type", "refresh_token")
		assertFormValue(t, r.Form, "refresh_token", "old-refresh")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	expiresAt := time.Now().Add(4 * time.Minute)
	store := &fakeOAuthStore{oauthCfg: oauthConfig(t, tokenServer.URL, expiresAt, "old-refresh")}

	resolver := NewResolver(store, testKey(), oauth.NewRefresher())
	token, err := resolver.Resolve(context.Background(), "vault", "GOOGLE", "old-access", ResolveOptions{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if token != "new-access" {
		t.Fatalf("token = %q, want new-access", token)
	}
	if refreshCalls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshCalls)
	}
	if store.updateAccessToken != "new-access" {
		t.Fatalf("stored access token = %q, want new-access", store.updateAccessToken)
	}
	if store.updateRefreshToken != "new-refresh" {
		t.Fatalf("stored refresh token = %q, want new-refresh", store.updateRefreshToken)
	}
	if store.updateExpiresAt == nil {
		t.Fatal("stored expiry is nil")
	}
}

func TestResolverResolve_PersistsRefreshError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	defer tokenServer.Close()

	expiresAt := time.Now().Add(-time.Minute)
	store := &fakeOAuthStore{oauthCfg: oauthConfig(t, tokenServer.URL, expiresAt, "old-refresh")}

	resolver := NewResolver(store, testKey(), oauth.NewRefresher())
	_, err := resolver.Resolve(context.Background(), "vault", "GOOGLE", "old-access", ResolveOptions{})
	if !errors.Is(err, ErrRefreshFailed) {
		t.Fatalf("err = %v, want ErrRefreshFailed", err)
	}
	if store.errorCalls != 1 {
		t.Fatalf("error updates = %d, want 1", store.errorCalls)
	}
	if store.errorMessage == "" {
		t.Fatal("stored refresh error is empty")
	}
}

func TestResolverResolve_ForcedRefreshIgnoresValidExpiry(t *testing.T) {
	var refreshCalls int
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		refreshCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "forced-access",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	expiresAt := time.Now().Add(10 * time.Minute)
	store := &fakeOAuthStore{oauthCfg: oauthConfig(t, tokenServer.URL, expiresAt, "old-refresh")}

	resolver := NewResolver(store, testKey(), oauth.NewRefresher())
	token, err := resolver.Resolve(context.Background(), "vault", "GOOGLE", "old-access", ResolveOptions{ForceRefresh: true})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if token != "forced-access" {
		t.Fatalf("token = %q, want forced-access", token)
	}
	if refreshCalls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshCalls)
	}
	if store.updateRefreshToken != "" {
		t.Fatalf("stored refresh token = %q, want empty when provider did not rotate", store.updateRefreshToken)
	}
}

func oauthConfig(t *testing.T, tokenURL string, expiresAt time.Time, refreshToken string) *store.CredentialOAuth {
	t.Helper()
	refreshCT, refreshNonce, err := crypto.Encrypt([]byte(refreshToken), testKey())
	if err != nil {
		t.Fatalf("encrypt refresh token: %v", err)
	}
	return &store.CredentialOAuth{
		TokenURL:          tokenURL,
		ClientID:          "client",
		RefreshTokenCT:    refreshCT,
		RefreshTokenNonce: refreshNonce,
		TokenExpiresAt:    &expiresAt,
	}
}

func assertFormValue(t *testing.T, form url.Values, key string, want string) {
	t.Helper()
	if got := form.Get(key); got != want {
		t.Fatalf("form[%s] = %q, want %q", key, got, want)
	}
}

func testKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x42
	}
	return key
}
