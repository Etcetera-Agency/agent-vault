# Design: Enforce Egress Quota (Single Credential)

## Fork-Conflict-Surface Choice

- All logic in new package `internal/egressquota`. `internal/ratelimit` reused
  unchanged; if an unexported primitive is needed, add a new exported wrapper in a
  fork-local file.
- One `// fork-local:` enforcement hook in `internal/brokercore/credential.go`
  after service match / before injection. No signature changes.
- Tests in new `*_test.go`.

## Enforcement Point

```
match service
  └─ fork-local: if service has quota:
        account = the service credential as one implicit account
        if counter(day|month) >= configured cap: deny 429 (+ Retry-After)
        reserve rpm token + concurrency slot; if unavailable: deny/defer
inject credential
proxy upstream
  └─ on ProxyEvent success: increment day + month counters
release concurrency slot
```

## Counter Model (persisted, restart-safe)

- Windows: day (`YYYY-MM-DD`) and month (`YYYY-MM`), each enforced only when its
  cap is configured.
- Authoritative source is `requestlog` (`MatchedService` + `CredentialKeys` +
  timestamp + status). In-memory counters seeded at startup from current-day and
  current-month rows, incremented on `ProxyEvent`, periodic reconcile.
- `rpm`/`concurrency` are in-memory only.
- Exhausted when any configured cap reached; `Retry-After` = earliest window
  reset.

## Non-Goals

- No multi-account pool/rotation/failover (next slice).
- No UI (later slice). No cost accounting.
