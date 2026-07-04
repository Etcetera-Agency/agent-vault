# Proposal: Quota and Account Management UI

## Why

Quota and account-pool config exist (`add-service-quota-config`) and are enforced
(`enforce-egress-quota`, `add-credential-rotation`), but operators can only edit
them by hand and cannot see usage. Agent Vault's management UI already edits
services (`web/src/pages/vault/ServicesTab.tsx`); it should also edit quotas and
pools and surface current usage and exhaustion.

**Current state**: No UI for quota/accounts; no usage visibility.

**Desired state**: Operators edit quota caps (all optional), manage the account
pool and rotation, and see per-account daily/monthly usage and
exhaustion/cooldown — without ever seeing credential values.

## What Changes

- Extend the service create/edit API (additive) to round-trip `quota`,
  `accounts`, `rotation`, and expose per-account window usage + exhaustion state.
- Add UI in `web/src/pages/vault/ServicesTab.tsx` plus new `QuotaEditor` and
  `AccountPoolEditor` components: edit caps, manage the pool + rotation, show
  usage/exhaustion. Never render credential values (select `credential_key` only).
- Reuse backend validation errors in UI feedback; allow an entirely empty quota.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md` - Management-surface portion.

### Affected Code
- Service create/edit API handler - Additive round-trip of quota/accounts/rotation
  plus a read path exposing per-account usage and exhaustion.
- `web/src/pages/vault/ServicesTab.tsx` + new `QuotaEditor.tsx` /
  `AccountPoolEditor.tsx` - Editor and usage display.
- New fork-local web component tests and API tests.

### User Impact
- Operators manage ceilings and read usage from the UI; one human touchpoint.

### API Changes
- Service endpoints accept/return `quota`, `accounts`, `rotation`.
- A read path exposes per-account daily/monthly usage vs cap and
  exhaustion/cooldown state.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Medium.

## Risks

- Leaking credential values in the editor. Mitigation: UI references
  `credential_key` only; values never returned by the API.
- Depends on `add-service-quota-config` (edit) and
  `enforce-egress-quota`/`add-credential-rotation` (usage/exhaustion data).
