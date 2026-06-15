# Design: Service Method Surfaces

## Fork-Conflict Surface

This repo is a fork of upstream Agent Vault. Surface changes should reuse
existing handlers/CLI/UI paths with small additive hooks:

- add method rendering through helper functions instead of rewriting response builders
- keep existing response field order except appending `methods`
- avoid broad UI layout rewrites; add only the method control/readout needed here
- mark required upstream-file touches with `// fork-local:`
- add tests in new fork-local `*_test.go` files where possible

## Discover Response

```json
{
  "services": [
    {
      "name": "calendar-events-read",
      "host": "www.googleapis.com/calendar/v3/calendars/*/events*",
      "methods": ["GET"]
    }
  ],
  "available_credentials": ["GOOGLE_ACCESS_TOKEN"]
}
```

## CLI Shape

```bash
agent-vault vault service add \
  --name calendar-events-read \
  --host 'www.googleapis.com/calendar/v3/calendars/*/events*' \
  --method GET \
  --auth bearer:GOOGLE_ACCESS_TOKEN
```

## UI Shape

Service editor:

- Host/path fields stay unchanged.
- Methods use explicit checkboxes or multi-select.
- Empty methods display as "Any method" for audit clarity.
- Saved payload uses `methods: ["GET", "POST"]`.
