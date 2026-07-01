# Implementation Tasks

- [x] 1. Implement `Select(service)` in `internal/egressquota` with `least_used`
      and `round_robin` policies and per-account day/month counters.
- [x] 2. Reserve rpm token + concurrency slot atomically during selection; skip to
      the next candidate when unavailable.
- [x] 3. Implement `Cooldown(account, d)` honoring upstream `Retry-After` (reuse
      the failure-counter primitive).
- [x] 4. Extend the `// fork-local:` hook in `internal/brokercore/credential.go`
      to inject the selected account and fail over to the next on upstream 429
      within bounded attempts.
- [x] 5. Deny with HTTP 429 + `X-Vault-Quota-Exhausted` + `Retry-After` when the
      whole pool is exhausted.
- [x] 6. Emit one `internal/notify` SMTP alert per `<service>:<window>` (dedup).
- [x] 7. Fork-local tests: least-used, round-robin, skip-exhausted, failover on
      upstream 429, cooldown honored, full-pool exhaustion, single alert per
      window.
- [x] 8. Docs: rotation policies, failover, exhaustion alert; ToS note on multiple
      accounts.
