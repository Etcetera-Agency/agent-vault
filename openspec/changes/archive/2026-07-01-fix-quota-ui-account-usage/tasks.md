# Implementation Tasks

- [x] 1. Update usage API inclusion predicate to `quota != nil || len(accounts) > 0`.
- [x] 2. Return account rows for account-only services with caps omitted and
      state derived from cooldown/exhaustion.
- [x] 3. Ensure usage API response contains credential keys only as references
      and never credential values.
- [x] 4. Update Services table quota column to render account-only account rows,
      not `Configured`.
- [x] 5. Keep editor behavior: blank quota plus accounts is valid.
- [x] 6. Add API tests for account-only usage, cooling state, and no credential
      value exposure.
- [x] 7. Run `npm run build` in `web`.
- [x] 8. Run `go test ./internal/server`.
- [x] 9. Run `openspec validate fix-quota-ui-account-usage --strict`.
