package brokercore

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Infisical/agent-vault/internal/broker"
	"gopkg.in/yaml.v3"
)

// fork-local: Agent profile acceptance tests live outside upstream core files.

func loadAgentServiceProfile(t *testing.T) []broker.Service {
	t.Helper()

	path := filepath.Join("..", "..", "examples", "agent-service-profile", "services.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read agent profile: %v", err)
	}

	var cfg broker.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal agent profile: %v", err)
	}
	if cfg.Vault != "agent" {
		t.Fatalf("profile vault = %q, want agent", cfg.Vault)
	}
	if len(cfg.Services) != 4 {
		t.Fatalf("profile services = %d, want 4", len(cfg.Services))
	}
	return cfg.Services
}

func newAgentProfileProvider(t *testing.T) *StoreCredentialProvider {
	t.Helper()

	key32 := make32(0xA7)
	store := newFakeCredStore()
	store.policy = PolicyDeny
	store.setServices(t, "agent-vault-id", loadAgentServiceProfile(t))
	store.setCred(t, key32, "agent-vault-id", "GOOGLE_ACCESS_TOKEN", "google-real-token")
	store.setCred(t, key32, "agent-vault-id", "DISCORD_BOT_TOKEN", "discord-real-token")
	store.setCred(t, key32, "agent-vault-id", "TELEGRAM_BOT_TOKEN", "123456:telegram-real-token")

	return NewStoreCredentialProvider(store, key32)
}

func TestAgentServiceProfileAllowedRoutes(t *testing.T) {
	provider := newAgentProfileProvider(t)

	cases := []struct {
		name        string
		method      string
		host        string
		path        string
		service     string
		authHeader  string
		credential  string
		credentials []string
	}{
		{
			name:        "gmail read",
			method:      "GET",
			host:        "gmail.googleapis.com",
			path:        "/gmail/v1/users/me/messages",
			service:     "gmail-list-read",
			authHeader:  "Bearer google-real-token",
			credential:  "GOOGLE_ACCESS_TOKEN",
			credentials: []string{"GOOGLE_ACCESS_TOKEN"},
		},
		{
			name:        "calendar read",
			method:      "GET",
			host:        "www.googleapis.com",
			path:        "/calendar/v3/calendars/primary/events",
			service:     "calendar-events-read",
			authHeader:  "Bearer google-real-token",
			credential:  "GOOGLE_ACCESS_TOKEN",
			credentials: []string{"GOOGLE_ACCESS_TOKEN"},
		},
		{
			name:        "discord send",
			method:      "POST",
			host:        "discord.com",
			path:        "/api/v10/channels/123/messages",
			service:     "discord-channel-messages",
			authHeader:  "Bearer discord-real-token",
			credential:  "DISCORD_BOT_TOKEN",
			credentials: []string{"DISCORD_BOT_TOKEN"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := provider.Inject(context.Background(), "agent-vault-id", tc.host, 443, tc.method, tc.path)
			if err != nil {
				t.Fatalf("Inject: %v", err)
			}
			if res.MatchedName != tc.service {
				t.Fatalf("MatchedName = %q, want %q", res.MatchedName, tc.service)
			}
			if got := res.Headers["Authorization"]; got != tc.authHeader {
				t.Fatalf("Authorization = %q, want %q", got, tc.authHeader)
			}
			if strings.Contains(res.Headers["Authorization"], tc.credential) {
				t.Fatalf("Authorization leaked key name instead of resolved value: %q", res.Headers["Authorization"])
			}
			if strings.Join(res.CredentialKeys, ",") != strings.Join(tc.credentials, ",") {
				t.Fatalf("CredentialKeys = %v, want %v", res.CredentialKeys, tc.credentials)
			}
		})
	}
}

func TestAgentServiceProfileDeniedRoutes(t *testing.T) {
	provider := newAgentProfileProvider(t)

	cases := []struct {
		name   string
		method string
		host   string
		path   string
		want   error
	}{
		{
			name:   "gmail write denied",
			method: "POST",
			host:   "gmail.googleapis.com",
			path:   "/gmail/v1/users/me/messages",
			want:   ErrServiceMethodDenied,
		},
		{
			name:   "calendar write denied",
			method: "POST",
			host:   "www.googleapis.com",
			path:   "/calendar/v3/calendars/primary/events",
			want:   ErrServiceMethodDenied,
		},
		{
			name:   "gmail sibling path denied",
			method: "GET",
			host:   "gmail.googleapis.com",
			path:   "/gmail/v1/users/me/settings",
			want:   ErrServiceNotFound,
		},
		{
			name:   "calendar sibling path denied",
			method: "GET",
			host:   "www.googleapis.com",
			path:   "/calendar/v3/users/me/calendarList",
			want:   ErrServiceNotFound,
		},
		{
			name:   "discord sibling path denied",
			method: "GET",
			host:   "discord.com",
			path:   "/api/v10/guilds/123",
			want:   ErrServiceNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := provider.Inject(context.Background(), "agent-vault-id", tc.host, 443, tc.method, tc.path)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Inject err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestAgentServiceProfileTelegramSubstitutionAndLogRedaction(t *testing.T) {
	provider := newAgentProfileProvider(t)

	res, err := provider.Inject(context.Background(), "agent-vault-id", "api.telegram.org", 443, "POST", "/bot__bot_token__/sendMessage")
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}
	if res.MatchedName != "telegram-bot-api" {
		t.Fatalf("MatchedName = %q, want telegram-bot-api", res.MatchedName)
	}
	if len(res.Headers) != 0 {
		t.Fatalf("telegram profile should use passthrough auth, got headers %+v", res.Headers)
	}
	if len(res.Substitutions) != 1 {
		t.Fatalf("Substitutions = %+v, want one entry", res.Substitutions)
	}

	rawURL := "https://api.telegram.org/bot__bot_token__/sendMessage"
	upstreamURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if err := ApplySubstitutions(upstreamURL, nil, res.Substitutions); err != nil {
		t.Fatalf("ApplySubstitutions: %v", err)
	}
	if strings.Contains(upstreamURL.String(), "__bot_token__") {
		t.Fatalf("placeholder was not substituted: %s", upstreamURL.String())
	}
	if !strings.Contains(upstreamURL.EscapedPath(), url.PathEscape("123456:telegram-real-token")) {
		t.Fatalf("substituted path missing escaped token: %s", upstreamURL.String())
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	LogProxyEvent(logger, ProxyEvent{
		Ingress:        IngressMITM,
		Method:         "POST",
		Host:           "api.telegram.org",
		Path:           "/bot__bot_token__/sendMessage",
		MatchedService: res.MatchedName,
		MatchedHost:    res.MatchedHost,
		MatchedPath:    res.MatchedPath,
		CredentialKeys: res.CredentialKeys,
		Status:         200,
		TotalMs:        17,
	})

	out := buf.String()
	for _, needle := range []string{
		"method=POST",
		"host=api.telegram.org",
		"path=/bot__bot_token__/sendMessage",
		"matched_service=telegram-bot-api",
		"status=200",
		"total_ms=17",
		"TELEGRAM_BOT_TOKEN",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("log missing %q: %s", needle, out)
		}
	}
	if strings.Contains(out, "123456:telegram-real-token") {
		t.Fatalf("request log leaked Telegram token: %s", out)
	}
}
