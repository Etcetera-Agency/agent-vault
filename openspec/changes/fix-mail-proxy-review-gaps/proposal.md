# Change: Fix mail proxy review gaps

## Why

The Hermes Gmail mail proxy implementation covers the MVP path, but review found three gaps before this should be treated as complete production behavior:

- controlled shutdown closes listeners but does not explicitly close active IMAP/SMTP relay sockets after `--shutdown-timeout`;
- SMTP upstream TLS verification uses a hardcoded `smtp.gmail.com` server name even when `--smtp-upstream` is configured;
- operators can store `mail_proxy` policy on a service record, but need a simple command that toggles IMAP/SMTP on an existing allowlist record without editing YAML or using Web UI configuration.

**Current state**:
- `agent-vault mail-proxy` accepts cancellation, closes listeners, and waits for sessions.
- SMTP upstream override changes the dial address but not the STARTTLS verification server name.
- `agent-vault vault service add` and service YAML can carry `mail_proxy`, and the Web UI preserves/displays it.

**Desired state**:
- shutdown closes active relay connections after the configured grace period;
- SMTP STARTTLS verifies the configured upstream host;
- an existing service record can have mail proxy fields changed in place by CLI, preserving unrelated fields.

## What Changes

- Track active local IMAP/SMTP sessions in `mailproxy.Proxy` and close them after `--shutdown-timeout`.
- Derive SMTP upstream TLS `ServerName` from the configured upstream address.
- Add a small `agent-vault vault service mail-proxy set <service>` CLI path for existing service records.
- Keep Web UI read-only for mail proxy policy: display/preserve only, no Web UI configuration.
- Clean whitespace issues in the concept patch so diff checks pass.

## Impact

### Affected Specifications

- `openspec/specs/mail-proxy/spec.md` - controlled shutdown, SMTP upstream STARTTLS verification, and service-record mail proxy management behavior.

### Affected Code

- `internal/mailproxy/proxy.go` - active session tracking and forced close after timeout.
- `internal/mailproxy/relay.go` - no major behavior change required; session lifecycle remains owned by `Proxy`.
- `internal/mailproxy/smtp.go` - upstream STARTTLS config takes server name from configured upstream host.
- `cmd/service.go` - new CLI subcommand for `mail_proxy` updates on existing service records.
- `cmd/*_test.go`, `internal/mailproxy/*_test.go` - focused TDD coverage.
- `openspec/agent-vault-hermes-gmail-imap-smtp-proxy-patch-v1.0.0.md` - whitespace cleanup only.

### User Impact

- Hermes operators get deterministic shutdown behavior.
- Custom SMTP upstream test environments and future Gmail endpoint changes do not fail TLS verification due to a stale hardcoded name.
- Operators can enable/disable IMAP/SMTP on an existing Gmail service record without touching the Web UI.

### API Changes

- No HTTP API change.
- New CLI subcommand:
  - `agent-vault vault service mail-proxy set <service> --imap=<bool> --smtp=<bool> [--email <addr>] [--local-password-credential <key>]`

### Migration Required

- [ ] Database migration
- [ ] API version bump
- [ ] User communication needed
- [x] Documentation updates

## Timeline Estimate

Small. Three focused implementation slices plus cleanup.

## Risks

- Active connection closing can race session cleanup. Mitigation: own registration in `Proxy`, protect map with mutex, make close idempotent.
- CLI partial updates can accidentally drop fields if implemented as replace. Mitigation: GET full service list, modify only selected service's `MailProxy`, PUT full list back.
