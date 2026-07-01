# Change: Add Hermes Gmail mail proxy

## Why

Hermes Email Gateway polls an IMAP mailbox and sends replies through SMTP, but Gmail no longer fits the simple password model for personal accounts. Hermes currently expects `EMAIL_ADDRESS`, `EMAIL_PASSWORD`, `EMAIL_IMAP_HOST`, and `EMAIL_SMTP_HOST`; exposing a Google OAuth refresh token or app password to Hermes breaks the Agent Vault security boundary.

This change adds a small local Agent Vault mail proxy so unmodified Hermes can keep using IMAP/SMTP while Agent Vault keeps Google OAuth tokens encrypted and performs Gmail XOAUTH2 authentication upstream.

## What Changes

- Add a separate `agent-vault mail-proxy` command.
- Reuse existing Agent Vault store opening, master-key unlock, encrypted credential storage, OAuth refresh, and CA minting.
- Use an existing Gmail service allowlist record in the target Agent Vault vault; the proxy selects that record by name.
- Add optional `mail_proxy` settings to that service allowlist record so IMAP and SMTP can be enabled or disabled per existing Gmail record.
- Expose loopback-only IMAP and SMTP listeners for Hermes.
- Terminate local TLS with a certificate minted by the existing Agent Vault CA.
- Authenticate Hermes locally with a separate Agent Vault credential password.
- Authenticate to Gmail IMAP/SMTP upstream with OAuth XOAUTH2.
- Relay IMAP/SMTP bytes after local auth and upstream XOAUTH2 succeed.
- Document the Hermes environment variables and Agent Vault CA trust setup.

## Impact

- Affected specs: `mail-proxy`
- Affected code:
  - `internal/broker/*`
  - `cmd/mail_proxy.go`
  - `internal/oauthcredential/resolver.go`
  - `internal/mailproxy/*`
  - existing service allowlist management surfaces
  - `docs/guides/hermes-gmail-mail-proxy.mdx`
  - one minimal fork-local broker edit to reuse the OAuth resolver
- No database migration.
- No Hermes code change.
- No standalone mail-proxy web configuration page.
- No Gmail/OAuth setup flow inside the mail proxy; it uses the OAuth credential referenced by the selected service record.
- No push without explicit approval.
