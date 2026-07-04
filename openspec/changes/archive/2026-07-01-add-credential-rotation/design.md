# Design: Multi-Account Credential Rotation and Failover

## Fork-Conflict-Surface Choice

- Selector/cooldown logic added to `internal/egressquota` (new files). The
  brokercore hook from the enforcement slice is extended in place with an
  additive `// fork-local:` block, no signature change.
- Tests in new `*_test.go`.

## Account Selection And Rotation

- `Select(service)` returns the first account whose day and month counters are
  under cap and that is not in cooldown, ordered by:
  - `least_used`: ascending current usage.
  - `round_robin`: next index after the last used.
- Selection reserves an rpm token + concurrency slot atomically; if either is
  unavailable it advances to the next candidate.

## Failover And Cooldown

- On upstream HTTP 429 / quota error for the selected account:
  `Cooldown(account, d)` marks it unavailable for `d` (honoring upstream
  `Retry-After`, reusing the failure-counter primitive), then retry with the next
  available account within a bounded attempt count.
- When no account is available (all over cap or cooling): deny HTTP 429 +
  `X-Vault-Quota-Exhausted` + `Retry-After` (earliest reset/cooldown remaining).

## Exhaustion Alert

- Emit at most one `internal/notify` SMTP alert per `<service>:<cooldown-window>`
  (dedup) so a runaway does not spam the operator. The distinct status/header lets
  a downstream runtime raise its own operator task.

## Non-Goals

- No UI (separate slice). No cross-node distributed counter.
