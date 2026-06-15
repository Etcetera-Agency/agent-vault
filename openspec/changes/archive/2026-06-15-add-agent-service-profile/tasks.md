# Implementation Tasks

## Phase 1: Profile Config

- [x] 1. Add agent profile service YAML with Gmail read rule using `methods: ["GET"]`.
- [x] 2. Add agent profile service YAML with Calendar events read rule using `methods: ["GET"]`.
- [x] 3. Add agent profile service YAML with Discord channel messages rule using `methods: ["POST"]`.
- [x] 4. Add agent profile service YAML with Telegram bot API placeholder substitution and `methods: ["POST"]`.

## Phase 2: Credential Proposal Examples

- [x] 5. Add proposal example for `GOOGLE_ACCESS_TOKEN` with obtain instructions.
- [x] 6. Add proposal example for `DISCORD_BOT_TOKEN` with obtain instructions.
- [x] 7. Add proposal example for `TELEGRAM_BOT_TOKEN` with obtain instructions.

## Phase 3: Verification

- [x] 8. Add allowed-route checks for each profile service.
- [x] 9. Add denied-method checks for Gmail/Calendar writes.
- [x] 10. Add denied-path checks for sibling Gmail/Calendar/Discord paths.
- [x] 11. Add Telegram substitution check proving an agent uses placeholder path and logs do not reveal token value.
- [x] 12. Add request log check for method, host, path, matched service, status, and latency.

## Phase 4: Docs

- [x] 13. Document placeholder credential values used by Agent Vault examples.
- [x] 14. Document strict deny prerequisite.
- [x] 15. Document that profile use is blocked until URL-shape verification passes.
- [x] 16. Document that agent profile rules are editable through the service allowlist UI.
