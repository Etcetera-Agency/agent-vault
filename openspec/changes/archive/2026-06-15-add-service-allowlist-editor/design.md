# Design: Service Allowlist Editor

## Fork-Conflict Surface

This repo is a fork of upstream Agent Vault. Editor work should stay a thin
extension of the existing service management surface:

- reuse existing service update/list endpoints if they can round-trip all fields
- add only the missing method/substitution controls needed for agent rules
- avoid replacing upstream forms, state models, routing, or API clients wholesale
- mark required upstream-file touches with `// fork-local:`
- add UI/API tests in new fork-local test files where possible

## Editor Fields

```text
name
enabled
host/path pattern
methods
auth.type
auth credential key references
custom headers when auth.type == custom
substitutions:
  key
  placeholder
  locations: header, query, path, body
```

## Save Pseudocode

```text
open service editor
load existing service object
operator edits fields
build full service payload
submit to service API
if backend validation fails:
  show field-level or form-level error
else:
  refresh service list
  refresh discover preview
```

## Credential Safety

```text
credential selector displays:
  key name
  description if available

credential selector never displays:
  credential value
  decrypted secret preview
```

## Agent Rule Examples

The editor must round-trip:

```yaml
name: calendar-events-read
host: www.googleapis.com/calendar/v3/calendars/*/events*
methods: [GET]
auth:
  type: bearer
  token: GOOGLE_ACCESS_TOKEN
```

```yaml
name: telegram-bot-api
host: api.telegram.org/bot__bot_token__/*
methods: [POST]
auth:
  type: passthrough
substitutions:
  - key: TELEGRAM_BOT_TOKEN
    placeholder: __bot_token__
    in: [path]
```
