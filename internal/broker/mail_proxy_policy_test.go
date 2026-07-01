package broker

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestServiceMailProxyJSONRoundTrip(t *testing.T) {
	input := []byte(`{
		"name":"gmail-mail",
		"host":"gmail.googleapis.com/gmail/v1/users/me/messages*",
		"auth":{"type":"bearer","token":"GOOGLE_MAIL_OAUTH"},
		"mail_proxy":{
			"email":"agent@gmail.com",
			"local_password_credential":"HERMES_MAIL_LOCAL_PASSWORD",
			"imap":true,
			"smtp":false
		}
	}`)

	var svc Service
	if err := json.Unmarshal(input, &svc); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if svc.MailProxy == nil {
		t.Fatal("MailProxy is nil")
	}
	if !svc.MailProxy.IMAP || svc.MailProxy.SMTP {
		t.Fatalf("mail proxy protocol flags = imap:%v smtp:%v, want imap:true smtp:false", svc.MailProxy.IMAP, svc.MailProxy.SMTP)
	}
	if svc.MailProxy.LocalPasswordCredential != "HERMES_MAIL_LOCAL_PASSWORD" {
		t.Fatalf("local password credential = %q", svc.MailProxy.LocalPasswordCredential)
	}

	out, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), `"mail_proxy"`) {
		t.Fatalf("marshaled service dropped mail_proxy: %s", out)
	}
}

func TestServiceMailProxyYAMLRoundTrip(t *testing.T) {
	input := []byte(`
name: gmail-mail
host: gmail.googleapis.com/gmail/v1/users/me/messages*
auth:
  type: bearer
  token: GOOGLE_MAIL_OAUTH
mail_proxy:
  email: agent@gmail.com
  local_password_credential: HERMES_MAIL_LOCAL_PASSWORD
  imap: true
  smtp: true
`)

	var svc Service
	if err := yaml.Unmarshal(input, &svc); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if svc.MailProxy == nil || !svc.MailProxy.IMAP || !svc.MailProxy.SMTP {
		t.Fatalf("mail proxy policy = %+v", svc.MailProxy)
	}

	out, err := yaml.Marshal(svc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), "mail_proxy:") {
		t.Fatalf("marshaled service dropped mail_proxy:\n%s", out)
	}
}
