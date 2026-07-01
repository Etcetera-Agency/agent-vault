# Implementation Tasks

- [x] 1. Change quota registry entry condition from `quota != nil` to
      `quota != nil || len(accounts) > 0`.
- [x] 2. Ensure empty quota value enforces no daily/monthly/rpm/concurrency
      limits while still checking cooldown state.
- [x] 3. Ensure account-only reservations still record selected account usage
      after forwarding.
- [x] 4. Update usage API to include account-only services.
- [x] 5. Update UI quota column to render account-only service states instead
      of generic `Configured`.
- [x] 6. Add account-only runtime tests for round-robin, least-used, cooldown
      skip, all-cooling exhaustion, and no-account/no-quota passthrough parity.
- [x] 7. Add validation tests proving account-only pools still reject
      unpoolable or wrong-provider credentials.
- [x] 8. Add server/UI tests or snapshots for account-only usage output.
- [x] 9. Update docs for account-only rotation.
- [x] 10. Run `go test ./internal/egressquota ./internal/brokercore ./internal/mitm ./internal/server`.
- [x] 11. Run `npm run build` in `web`.
- [x] 12. Run `openspec validate add-account-only-rotation --strict`.
