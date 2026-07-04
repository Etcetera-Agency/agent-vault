# Implementation Tasks

- [x] 1. Identify account-pool matched requests from `InjectResult` without
      exposing credential values.
- [x] 2. Materialize replayable request bodies for account-pool requests within
      the existing max request body limit.
- [x] 3. Refactor outbound request construction into an attempt builder that can
      apply a new injection result to a fresh body reader.
- [x] 4. Retry upstream 429/quota-error responses for replayable methods,
      including POST/PUT/PATCH with bodies.
- [x] 5. Bound attempts by available account candidates and a hard maximum.
- [x] 6. Cool down and release failed reservations before selecting retry
      accounts; commit only the final forwarded attempt that is returned.
- [x] 7. Add MITM tests for POST failover with body preserved, PUT failover,
      exhausted-after-retries, and request-too-large behavior.
- [x] 8. Run `go test ./internal/mitm ./internal/brokercore ./internal/egressquota`.
- [x] 9. Run `openspec validate fix-quota-failover-replay --strict`.
