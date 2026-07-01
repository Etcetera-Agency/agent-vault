package server

import (
	"testing"

	"github.com/Infisical/agent-vault/internal/broker"
)

func TestServiceResponsesExposeMailProxyPolicy(t *testing.T) {
	responses := serviceResponses([]broker.Service{{
		Name: "gmail-mail",
		Host: "gmail.googleapis.com",
		Auth: broker.Auth{Type: "bearer", Token: "GOOGLE_MAIL_OAUTH"},
		MailProxy: &broker.MailProxyPolicy{
			Email:                   "agent@gmail.com",
			LocalPasswordCredential: "HERMES_MAIL_LOCAL_PASSWORD",
			IMAP:                    true,
			SMTP:                    true,
		},
	}})

	if len(responses) != 1 {
		t.Fatalf("responses len = %d", len(responses))
	}
	if responses[0].MailProxy == nil {
		t.Fatal("MailProxy is nil")
	}
	if !responses[0].MailProxy.IMAP || !responses[0].MailProxy.SMTP {
		t.Fatalf("MailProxy = %+v", responses[0].MailProxy)
	}
}
