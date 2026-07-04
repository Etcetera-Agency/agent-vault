# Proposal: Fix Quota UI Account Usage

## Why

The management UI shows quota/account status in the service table. For
account-only services, the current API omits usage data and the table falls back
to a generic `Configured` label. That hides whether accounts are available,
cooling, or exhausted.

This slice makes the UI and usage API mirror runtime account-pool state for both
quota-backed and account-only services.

## What Changes

- Usage API returns services with `quota` or `accounts`.
- Account-only service rows include account id, credential key reference,
  daily/monthly usage counters, state, and available-at when cooling.
- UI table renders account states for account-only services.
- UI editor keeps allowing blank quota with accounts.
- UI never displays credential values.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md`

### Affected Code
- `internal/server/handle_services.go`
- `web/src/pages/vault/ServicesTab.tsx`
- Server tests for quota usage API.
- Web build and component coverage where available.

### User Impact
- Operators can trust table state for account-only pools and quota pools.

### API Changes
- `GET /v1/vaults/{name}/services/quota-usage` includes account-only services.

### Migration Required
- [ ] Database migration.
- [ ] API version bump.
- [x] User communication needed.
- [x] Documentation updates.

