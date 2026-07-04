# Proposal: Multi-Account Credential Rotation and Failover

## Why

`enforce-egress-quota` caps a single credential. Operators may hold several
accounts for the same service (e.g. multiple Apify tokens) and want Agent Vault
to spread load across them and fail over when one is throttled, instead of
stopping at the first account's ceiling.

**Current state**: Quota is enforced against one implicit account per service.

**Desired state**: When a service defines an `accounts` pool, Agent Vault selects
a non-exhausted account per the `rotation` policy at credential resolve, cools
down an account on upstream 429, fails over to the next, and — only when the whole
pool is exhausted — denies with a distinct signal plus one operator alert.

## What Changes

- Add account selection to `internal/egressquota`: pick a non-exhausted,
  non-cooling account by `rotation` (`least_used` | `round_robin`), reserving rpm
  + concurrency atomically.
- Extend the `internal/brokercore/credential.go` `// fork-local:` hook to inject
  the selected account's credential.
- On upstream HTTP 429 / quota error, cool down that account (honoring
  `Retry-After`) and retry with the next available account within bounded
  attempts.
- When all accounts are exhausted, deny with HTTP 429 + `X-Vault-Quota-Exhausted`
  + `Retry-After`, and emit at most one `internal/notify` SMTP alert per service
  per cooldown window.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md` - Rotation, failover, exhaustion.

### Affected Code
- `internal/egressquota/*.go` - Account selector, cooldown, per-account counters.
- `internal/brokercore/credential.go` - Extend the existing `// fork-local:` hook
  for account selection and mid-request failover.
- `internal/notify` - Reused for the exhaustion alert (new caller only).
- New fork-local `*_test.go`.

### User Impact
- Work continues across accounts; a single throttled account does not stop it.

### API Changes
- Exhaustion still surfaces as HTTP 429 + `X-Vault-Quota-Exhausted`; now it means
  the whole pool is exhausted.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Medium.

## Risks

- Rotation selects an exhausted account under races. Mitigation: atomic
  reserve-then-record; failover covers a stale pick.
- ToS exposure from multiple accounts. Mitigation: opt-in per service; documented
  as an operator decision.
- Depends on `enforce-egress-quota`.
