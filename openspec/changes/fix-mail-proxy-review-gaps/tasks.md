# Implementation Tasks

## 1. Approval

- [x] 1.1 Review this proposal, design, and spec delta.
- [x] 1.2 Confirm no Web UI mail proxy configuration is added.

## 2. Slice 1: Controlled Shutdown

- [x] 2.1 Add failing test where an active IMAP relay stays open after listener shutdown.
- [x] 2.2 Add failing test where an active SMTP relay stays open after listener shutdown.
- [x] 2.3 Add active connection tracking to `mailproxy.Proxy`.
- [x] 2.4 Close tracked active connections after `--shutdown-timeout`.
- [x] 2.5 Keep protocol handlers and `Relay` simple; lifecycle ownership stays in `Proxy`.
- [x] 2.6 Run `/usr/local/bin/go test ./internal/mailproxy`.
- [x] 2.7 Use Code Simplifier for this slice.

## 3. Slice 2: SMTP Upstream TLS ServerName

- [x] 3.1 Add failing test for `serverNameFromAddress("smtp.test.local:587") == "smtp.test.local"`.
- [x] 3.2 Add failing SMTP upstream STARTTLS test proving the TLS client uses the configured upstream host as `ServerName`.
- [x] 3.3 Pass derived upstream server name from `Proxy` into SMTP upstream auth.
- [x] 3.4 Keep default Gmail behavior unchanged for `smtp.gmail.com:587`.
- [x] 3.5 Run `/usr/local/bin/go test ./internal/mailproxy`.
- [x] 3.6 Use Code Simplifier for this slice.

## 4. Slice 3: Existing Service Mail Proxy Toggle CLI

- [x] 4.1 Add failing CLI test for updating `mail_proxy.imap=false` on an existing service while preserving host, auth, methods, enabled, substitutions, email, and local password credential.
- [x] 4.2 Add failing CLI test for updating `mail_proxy.smtp=true` on an existing service that has no prior `mail_proxy` block.
- [x] 4.3 Add failing CLI test that rejects no-op calls with none of `--imap`, `--smtp`, `--email`, or `--local-password-credential` changed.
- [x] 4.4 Add `agent-vault vault service mail-proxy set <service>` under the existing service command tree.
- [x] 4.5 Implement GET full services, exact name lookup, focused `MailProxy` mutation, and PUT full services using existing admin request helpers.
- [x] 4.6 Do not add Web UI edit controls for mail proxy policy.
- [x] 4.7 Run `/usr/local/bin/go test ./cmd ./internal/server ./internal/broker`.
- [x] 4.8 Use Code Simplifier for this slice.

## 5. Slice 4: Hygiene And Docs

- [ ] 5.1 Remove trailing whitespace from `openspec/agent-vault-hermes-gmail-imap-smtp-proxy-patch-v1.0.0.md`.
- [ ] 5.2 Update `docs/guides/hermes-gmail-mail-proxy.mdx` with the new CLI toggle command.
- [ ] 5.3 Update `openspec/TODO.md` if implementation discovers more deferred work.
- [ ] 5.4 Run `git diff --check`.
- [ ] 5.5 Use Code Simplifier for this slice.

## 6. Final Verification

- [ ] 6.1 Run `/usr/local/bin/go test ./...`.
- [ ] 6.2 Run `npm run build --prefix web`.
- [ ] 6.3 Run `openspec validate fix-mail-proxy-review-gaps --strict`.
- [ ] 6.4 Run `openspec validate --specs --strict`.
- [ ] 6.5 Update `completion.review` after fixes.
- [ ] 6.6 Archive after user approval and successful verification.
