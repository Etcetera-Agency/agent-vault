# Implementation Tasks

- [x] 1. Create `internal/egressquota` with a `Registry` keyed by `(service,
      account)` composing `internal/ratelimit` primitives (token bucket = rpm,
      semaphore = concurrency, failure counter = cooldown).
- [x] 2. Implement day + month counters seeded from `internal/requestlog` at
      startup; increment on `ProxyEvent`; periodic reconcile. Enforce a window
      only when its cap is set; exhausted = any configured cap reached.
- [x] 3. Treat a quota'd service with no `accounts` as one implicit account from
      its existing credential.
- [x] 4. Add the `// fork-local:` enforcement hook in
      `internal/brokercore/credential.go`: deny before injection when over cap or
      rate/concurrency unavailable.
- [x] 5. Emit deny response: HTTP 429 + `X-Vault-Quota-Exhausted` + `Retry-After`;
      body names exhaustion without leaking credentials.
- [x] 6. Ensure services without quota take the unchanged path (passthrough parity
      regression test).
- [x] 7. Fork-local tests: daily deny, monthly deny under daily, earliest window
      governs, rpm throttle, concurrency cap, monthly-only, rpm-only, restart-seed
      reconstructs both windows, unconfigured passthrough.
- [x] 8. Docs: enforcement behavior + "set a ceiling when adding a credential"
      (provider-side cap first, then Vault quota).
