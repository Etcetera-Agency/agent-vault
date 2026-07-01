package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Infisical/agent-vault/internal/broker"
	"github.com/Infisical/agent-vault/internal/session"
)

func TestServiceMailProxySetUpdatesExistingPolicyAndPreservesFields(t *testing.T) {
	resetServiceMailProxySetFlags(t)
	enabled := true
	services := []broker.Service{{
		Name:    "gmail-mail",
		Host:    "gmail.googleapis.com",
		Enabled: &enabled,
		Auth:    broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
		Methods: []string{"GET", "POST"},
		Substitutions: []broker.Substitution{{
			Key:         "THREAD_ID",
			Placeholder: "{thread_id}",
			In:          []string{"path"},
		}},
		MailProxy: &broker.MailProxyPolicy{
			Email:                   "agent@gmail.com",
			LocalPasswordCredential: "HERMES_MAIL_LOCAL_PASSWORD",
			IMAP:                    true,
			SMTP:                    true,
		},
	}}
	var updated []broker.Service
	server := newServiceMailProxyTestServer(t, services, &updated)
	defer server.Close()
	saveTestSession(t, server.URL)

	_, err := executeCommand("vault", "service", "mail-proxy", "set", "gmail-mail", "--imap=false")
	if err != nil {
		t.Fatalf("executeCommand: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("updated services = %d, want 1", len(updated))
	}
	got := updated[0]
	if got.MailProxy == nil {
		t.Fatal("MailProxy is nil")
	}
	if got.MailProxy.IMAP {
		t.Fatal("mail_proxy.imap = true, want false")
	}
	if !got.MailProxy.SMTP {
		t.Fatal("mail_proxy.smtp = false, want true")
	}
	if got.MailProxy.Email != "agent@gmail.com" {
		t.Fatalf("mail_proxy.email = %q", got.MailProxy.Email)
	}
	if got.MailProxy.LocalPasswordCredential != "HERMES_MAIL_LOCAL_PASSWORD" {
		t.Fatalf("local password credential = %q", got.MailProxy.LocalPasswordCredential)
	}
	if got.Host != services[0].Host || got.Auth.Token != services[0].Auth.Token || !got.IsEnabled() {
		t.Fatalf("unrelated fields changed: %+v", got)
	}
	if strings.Join(got.Methods, ",") != "GET,POST" || len(got.Substitutions) != 1 {
		t.Fatalf("policy metadata changed: %+v", got)
	}
}

func TestServiceMailProxySetCreatesPolicyOnExistingService(t *testing.T) {
	resetServiceMailProxySetFlags(t)
	services := []broker.Service{{
		Name: "gmail-mail",
		Host: "gmail.googleapis.com",
		Auth: broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
	}}
	var updated []broker.Service
	server := newServiceMailProxyTestServer(t, services, &updated)
	defer server.Close()
	saveTestSession(t, server.URL)

	_, err := executeCommand(
		"vault", "service", "mail-proxy", "set", "gmail-mail",
		"--imap=true",
		"--smtp=false",
		"--email", " agent@gmail.com ",
		"--local-password-credential", " HERMES_MAIL_LOCAL_PASSWORD ",
	)
	if err != nil {
		t.Fatalf("executeCommand: %v", err)
	}
	if len(updated) != 1 || updated[0].MailProxy == nil {
		t.Fatalf("updated service missing mail proxy: %+v", updated)
	}
	policy := updated[0].MailProxy
	if !policy.IMAP || policy.SMTP {
		t.Fatalf("protocol flags = imap:%v smtp:%v, want true/false", policy.IMAP, policy.SMTP)
	}
	if policy.Email != "agent@gmail.com" {
		t.Fatalf("email = %q, want trimmed agent@gmail.com", policy.Email)
	}
	if policy.LocalPasswordCredential != "HERMES_MAIL_LOCAL_PASSWORD" {
		t.Fatalf("local password credential = %q", policy.LocalPasswordCredential)
	}
}

func TestServiceMailProxySetRejectsNoop(t *testing.T) {
	resetServiceMailProxySetFlags(t)
	t.Setenv("HOME", t.TempDir())
	_, err := executeCommand("vault", "service", "mail-proxy", "set", "gmail-mail")
	if err == nil {
		t.Fatal("expected no-op error")
	}
	if !strings.Contains(err.Error(), "provide at least one mail proxy field") {
		t.Fatalf("error = %v", err)
	}
}

func TestServiceMailProxySetRejectsMissingService(t *testing.T) {
	resetServiceMailProxySetFlags(t)
	services := []broker.Service{{
		Name: "other-mail",
		Host: "gmail.googleapis.com",
		Auth: broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
	}}
	var updated []broker.Service
	server := newServiceMailProxyTestServer(t, services, &updated)
	defer server.Close()
	saveTestSession(t, server.URL)

	_, err := executeCommand("vault", "service", "mail-proxy", "set", "gmail-mail", "--imap=true")
	if err == nil {
		t.Fatal("expected missing service error")
	}
	if !strings.Contains(err.Error(), `service "gmail-mail" not found`) {
		t.Fatalf("error = %v", err)
	}
	if updated != nil {
		t.Fatalf("services updated on missing target: %+v", updated)
	}
}

func newServiceMailProxyTestServer(t *testing.T, services []broker.Service, updated *[]broker.Service) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"vault":    "default",
				"services": services,
			})
		case http.MethodPut:
			var req struct {
				Services []broker.Service `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			*updated = req.Services
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
}

func saveTestSession(t *testing.T, address string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if err := session.Save(&session.ClientSession{Token: "test-token", Address: address}); err != nil {
		t.Fatalf("session.Save: %v", err)
	}
}

func resetServiceMailProxySetFlags(t *testing.T) {
	t.Helper()
	defaults := map[string]string{
		"imap":                      "false",
		"smtp":                      "false",
		"email":                     "",
		"local-password-credential": "",
	}
	for name, value := range defaults {
		flag := serviceMailProxySetCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag %q not registered", name)
		}
		if err := flag.Value.Set(value); err != nil {
			t.Fatalf("reset flag %q: %v", name, err)
		}
		flag.Changed = false
	}
}
