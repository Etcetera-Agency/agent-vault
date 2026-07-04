# Proposal: Fix Quota Failover Request Replay

## Why

The spec requires Agent Vault to cool down a selected account and retry with the
next account when upstream returns HTTP 429 or a quota error. Current behavior
only retries safe `GET`/`HEAD` requests. POST/PUT/PATCH requests that are
replayable still stop at the first account, leaving account-pool failover
partially implemented.

## What Changes

- Make account-pool failover work for any replayable proxied request, including
  POST/PUT/PATCH with a materialized body.
- Materialize request body once for quota/account-pool failover paths within the
  existing request body limit, then clone fresh readers per attempt.
- Retry with the next available account within a bounded attempt count.
- Cool down failed accounts using `Retry-After` before retrying.
- Preserve existing request-size enforcement and substitution behavior.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md`

### Affected Code
- `internal/mitm/forward.go`
- `internal/brokercore` request body materialization helpers if needed.
- `internal/egressquota` reservation release/commit/cooldown flows.
- MITM tests for POST/PUT failover with body replay.

### User Impact
- Account pools fail over for real API calls that use request bodies, not only
  read-only calls.

### API Changes
- No control-plane API change.

### Migration Required
- [ ] Database migration.
- [ ] API version bump.
- [x] User communication needed.
- [x] Documentation updates.

