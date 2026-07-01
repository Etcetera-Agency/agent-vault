# OpenSpec TODO

## Deferred From Agent Vault Slices

1. Verify whether existing service APIs can atomically update a full single service rule. If not, create a separate spec for a focused service update endpoint before implementing the UI editor.
2. Verify whether the existing web service UI can safely round-trip every service field: host/path, methods, enabled state, auth config, custom headers, and substitutions. If not, create UI component subtasks before `add-service-allowlist-editor` implementation.
3. Verify real Google SDK URL shapes for Gmail messages and Calendar events before enabling the agent profile example.
4. Verify Discord and Telegram SDK/API URL shapes before enabling the agent profile example.
5. Verify Google OAuth handling before profile use: decide between static access token, OAuth credential stored in Agent Vault, or external refresh process.
6. Add OAuth-backed Google credential refresh support as a separate Agent Vault spec if static `GOOGLE_ACCESS_TOKEN` is insufficient.
7. Add response redaction policy as a separate Agent Vault spec if request logs or upstream responses expose sensitive payload fields beyond credential values.
8. Add remote-listen support for `agent-vault mail-proxy` only after the loopback-only MVP works. Remote support must require explicit opt-in, local TLS, clear CA trust setup for Hermes, and tests for rejecting remote plaintext.
9. Fix `agent-vault mail-proxy` controlled shutdown so active IMAP/SMTP relay connections are tracked and closed after `--shutdown-timeout`, not only listeners.
10. Make SMTP upstream STARTTLS verification derive `ServerName` from the configured `--smtp-upstream` host, or remove/custom-limit the override if upstream must stay Gmail-only.
## Resolved

11. ~~Decide how `/discover` represents unrestricted methods.~~ **Resolved: canonical `["*"]`.** Storage keeps unrestricted as empty/omitted (backward compatible); `["*"]` is accepted on input as the sole-element alias (normalized to empty) and rendered on every surface — `/discover`, service list, CLI, UI, docs. Captured in `add-service-method-field` and `surface-service-methods`.
12. ~~Decide UI copy for unrestricted method policy.~~ **Resolved: show the `["*"]` token** as the badge/value, with an `Any method` tooltip/label for readability.
