# Proposal: Enforce Egress Quota (Single Credential)

## Why

`add-service-quota-config` lets a service carry a `quota`, but nothing enforces
it. A downstream consumer (Hermes portfolio runtime) will run auto-generated
scripts; a bug or aggressive cron can burn a paid upstream (Apify, Amadeus,
Google Places). The only unbypassable chokepoint is Agent Vault: every
credentialed egress already flows through `internal/brokercore` and is recorded
by `internal/requestlog`.

This slice enforces a service's quota against a **single credential** (the
service credential, treated as one implicit account). Multi-account rotation is a
later slice (`add-credential-rotation`).

**Current state**: Quota config is stored but ignored at proxy time.

**Desired state**: Agent Vault denies an egress request when the configured
`daily_cap`/`monthly_cap` is reached, and throttles by `rpm`/`concurrency`, using
request-log-backed counters that survive restart. Services without quota are
unaffected.

## What Changes

- Add fork-local package `internal/egressquota` reusing `internal/ratelimit`
  primitives (token bucket = rpm, semaphore = concurrency, failure counter =
  cooldown) keyed by `(service, account)`.
- Track two windows — day (`YYYY-MM-DD`) and month (`YYYY-MM`) — seeded at startup
  from `internal/requestlog` and incremented on `ProxyEvent`.
- Enforce at credential resolve in `internal/brokercore/credential.go` via a
  `// fork-local:` hook: deny with HTTP 429 + `X-Vault-Quota-Exhausted` +
  `Retry-After` when any configured cap is reached; apply rpm/concurrency.
- Enforce only configured fields; omitted caps/limits impose nothing; services
  with no quota take the unchanged path.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md` - Enforcement portion.

### Affected Code
- `internal/egressquota/*.go` - New fork-local package.
- `internal/ratelimit/*.go` - Reused unchanged (new exported wrapper if needed).
- `internal/brokercore/credential.go` - Minimal `// fork-local:` enforcement hook.
- `internal/requestlog` - Read-only aggregation for counter seeding.
- New fork-local `*_test.go`.

### User Impact
- Configured single-credential services get an unbypassable outbound ceiling.

### API Changes
- Proxy may return HTTP 429 with `X-Vault-Quota-Exhausted` and `Retry-After`.
- Error body names quota exhaustion without exposing credentials.

### Migration Required
- [ ] Database migration (counter derives from existing request log)
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Medium.

## Risks

- Hot-path cost on every egress. Mitigation: in-memory counters seeded from the
  request log; log query only on cold start / reconcile.
- Depends on `add-service-quota-config` for the config fields.
