# Implementation Tasks

## 1. OpenSpec Approval

- [x] 1.1 Review proposal, design, and requirement deltas.
- [x] 1.2 Confirm loopback-only MVP and Agent Vault CA local TLS.

## 2. Shared OAuth Resolver

- [x] 2.1 Add focused failing tests for resolving a valid OAuth access token without refresh.
- [x] 2.2 Add focused failing tests for five-minute-buffer refresh, rotated refresh token persistence, refresh error persistence, and forced refresh.
- [x] 2.3 Add `internal/oauthcredential/resolver.go` using existing `store`, `crypto`, `oauth.Refresh`, and `oauth.Refresher`.
- [x] 2.4 Update the HTTP broker with a minimal `// fork-local:` edit to reuse the resolver.
- [x] 2.5 Run resolver and broker tests.

## 3. Command, Config, And Preflight

- [x] 3.1 Add failing service-policy tests for parsing, validating, preserving, and listing an optional `mail_proxy` block on existing service allowlist records.
- [x] 3.2 Append `MailProxy` policy fields to the existing broker service shape without a database migration.
- [x] 3.3 Update existing service allowlist management surfaces to preserve and expose `mail_proxy.imap` and `mail_proxy.smtp` toggles.
- [x] 3.4 Add failing config tests for `--service`/env precedence, required service fields, both-protocols-disabled rejection, disabled service rejection, and loopback-only listener validation.
- [x] 3.5 Add `cmd/mail_proxy.go` with `init()` registration and no `cmd/root.go` edit.
- [x] 3.6 Reuse existing store open and master-key unlock helpers where possible.
- [x] 3.7 Preflight vault, selected service record, existing connected OAuth credential, refresh token, email, local password credential, and listener safety before opening listeners.
- [x] 3.8 Run service-policy and command/config tests.

## 4. Local TLS And Auth

- [ ] 4.1 Add failing tests that local IMAP requires implicit TLS and local SMTP advertises/upgrades STARTTLS.
- [ ] 4.2 Add failing tests for constant-time password verification behavior, empty password rejection, and generic auth failure messages.
- [ ] 4.3 Wire `internal/ca.New` and `MintLeaf("127.0.0.1")` into mail-proxy local TLS config.
- [ ] 4.4 Implement local password loading from an existing static Agent Vault credential.
- [ ] 4.5 Run TLS/auth tests.

## 5. XOAUTH2 Helpers

- [ ] 5.1 Add failing tests for exact Gmail XOAUTH2 payload bytes and base64 encoding.
- [ ] 5.2 Add failing tests for one forced-refresh retry on upstream auth rejection.
- [ ] 5.3 Implement `internal/mailproxy/xoauth2.go` and token-provider wrappers.
- [ ] 5.4 Run XOAUTH2 tests.

## 6. IMAP Proxy

- [ ] 6.1 Add fake upstream IMAP tests for `CAPABILITY`, `NOOP`, malformed command, bad local auth, successful `LOGIN`, XOAUTH2 auth, and raw relay after auth.
- [ ] 6.2 Implement local IMAP implicit TLS listener and pre-auth parser.
- [ ] 6.3 Implement upstream `imap.gmail.com:993` TLS connect and XOAUTH2 authenticate.
- [ ] 6.4 Implement bidirectional relay and connection cleanup.
- [ ] 6.5 Run IMAP tests.

## 7. SMTP Proxy

- [ ] 7.1 Add fake upstream SMTP tests for `EHLO`, `STARTTLS`, `AUTH PLAIN`, `AUTH LOGIN`, bad local auth, upstream XOAUTH2, and raw relay after auth.
- [ ] 7.2 Implement local SMTP listener with STARTTLS and auth commands.
- [ ] 7.3 Implement upstream `smtp.gmail.com:587` STARTTLS and XOAUTH2 authenticate.
- [ ] 7.4 Implement bidirectional relay and connection cleanup.
- [ ] 7.5 Run SMTP tests.

## 8. Lifecycle And Docs

- [ ] 8.1 Add tests for SIGINT/SIGTERM shutdown, listener close, setup cancellation, relay grace period, and both-protocol-disabled failure.
- [ ] 8.2 Implement graceful shutdown with `--shutdown-timeout`.
- [ ] 8.3 Add `docs/guides/hermes-gmail-mail-proxy.mdx` with Gmail scopes, Agent Vault credentials, CA trust, and Hermes env.
- [ ] 8.4 Update `openspec/TODO.md` for any deferred work discovered during implementation.
- [ ] 8.5 Run full relevant test suite and `openspec validate add-hermes-gmail-mail-proxy --strict`.
- [ ] 8.6 Use Code Simplifier before finishing implementation.

## 9. Archive

- [ ] 9.1 After implementation and approval, archive the OpenSpec change.
- [ ] 9.2 Validate archived specs.
