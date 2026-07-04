# Implementation Tasks

- [x] 1. Add non-secret `account_id` metadata to proxy events/request logs and
      store read/write paths.
- [x] 2. Change `internal/egressquota` candidates and quota keys to use
      account id for counters, cooldowns, rate state, concurrency state, and
      round-robin cursor decisions.
- [x] 3. Extend reservations with separate `AccountID` and `CredentialKey`
      accessors.
- [x] 4. Update `internal/brokercore` injection so selected credential key drives
      auth injection while selected account id is propagated to logs and usage.
- [x] 5. Update quota usage API to read registry usage by account id and return
      account id as authoritative state identity.
- [x] 6. Add tests where `id != credential_key`: least-used, round-robin,
      cooldown, exhaustion, request-log seeding, usage response.
- [x] 7. Update docs for account id vs credential key semantics.
- [x] 8. Run `go test ./internal/egressquota ./internal/brokercore ./internal/mitm ./internal/server`.
- [x] 9. Run `openspec validate fix-quota-account-identity --strict`.
