package mailproxy

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"testing"
)

type fakeTokenProvider struct {
	accessToken string
	forcedToken string
	accessCalls int
	forceCalls  int
}

func (f *fakeTokenProvider) AccessToken(context.Context) (string, error) {
	f.accessCalls++
	return f.accessToken, nil
}

func (f *fakeTokenProvider) ForceRefresh(context.Context) (string, error) {
	f.forceCalls++
	return f.forcedToken, nil
}

func TestXOAUTH2PayloadBytes(t *testing.T) {
	got := XOAUTH2Payload("agent@gmail.com", "ya29.token")
	want := []byte("user=agent@gmail.com\x01auth=Bearer ya29.token\x01\x01")
	if !slices.Equal(got, want) {
		t.Fatalf("payload = %q, want %q", got, want)
	}
}

func TestXOAUTH2Base64(t *testing.T) {
	got := XOAUTH2Base64("agent@gmail.com", "ya29.token")
	want := base64.StdEncoding.EncodeToString([]byte("user=agent@gmail.com\x01auth=Bearer ya29.token\x01\x01"))
	if got != want {
		t.Fatalf("base64 = %q, want %q", got, want)
	}
}

func TestWithForcedRefreshRetryRetriesOnceOnAuthRejection(t *testing.T) {
	provider := &fakeTokenProvider{accessToken: "old-token", forcedToken: "new-token"}
	var attempts []string

	err := WithForcedRefreshRetry(context.Background(), provider, func(token string) error {
		attempts = append(attempts, token)
		if token == "old-token" {
			return ErrXOAUTH2Rejected
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithForcedRefreshRetry: %v", err)
	}
	if !slices.Equal(attempts, []string{"old-token", "new-token"}) {
		t.Fatalf("attempts = %v", attempts)
	}
	if provider.accessCalls != 1 || provider.forceCalls != 1 {
		t.Fatalf("calls = access:%d force:%d, want 1/1", provider.accessCalls, provider.forceCalls)
	}
}

func TestWithForcedRefreshRetryDoesNotRetryOtherErrors(t *testing.T) {
	provider := &fakeTokenProvider{accessToken: "old-token", forcedToken: "new-token"}
	wantErr := errors.New("network")

	err := WithForcedRefreshRetry(context.Background(), provider, func(string) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want network", err)
	}
	if provider.forceCalls != 0 {
		t.Fatalf("force calls = %d, want 0", provider.forceCalls)
	}
}

func TestWithForcedRefreshRetryReturnsSecondRejection(t *testing.T) {
	provider := &fakeTokenProvider{accessToken: "old-token", forcedToken: "new-token"}

	err := WithForcedRefreshRetry(context.Background(), provider, func(string) error {
		return ErrXOAUTH2Rejected
	})
	if !errors.Is(err, ErrXOAUTH2Rejected) {
		t.Fatalf("err = %v, want ErrXOAUTH2Rejected", err)
	}
	if provider.forceCalls != 1 {
		t.Fatalf("force calls = %d, want 1", provider.forceCalls)
	}
}
