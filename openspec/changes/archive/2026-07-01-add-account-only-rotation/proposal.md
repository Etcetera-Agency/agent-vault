# Proposal: Add Account-Only Rotation

## Why

Operators can save `accounts` and `rotation` without quota caps. The UI already
supports this shape, but runtime currently skips account selection whenever
`quota` is absent. That makes an account pool look configured while traffic still
uses the base service credential.

Chosen behavior: account-only config is valid and SHALL rotate/fail over without
caps.

## What Changes

- Runtime enters account selection when a service has `accounts`, even if
  `quota` is absent.
- Account-only services apply rotation and cooldown/failover but no daily,
  monthly, rpm, or concurrency limits unless those fields are configured.
- Account-only rotation still honors credential pool eligibility:
  `accounts[].credential_key` must be enabled for the service
  `account_pool_provider`.
- Services with no `quota` and no `accounts` keep current unlimited/pass-through
  behavior.
- Usage API returns account states for account-only services.
- Tests cover account-only least-used/round-robin and UI/API visibility.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md`

### Affected Code
- `internal/egressquota/registry.go`
- `internal/brokercore/credential.go`
- `internal/mitm/forward.go`
- `internal/server/handle_services.go`
- `web/src/pages/vault/ServicesTab.tsx`

### User Impact
- Operators can use Agent Vault as account rotator without setting numeric caps.

### API Changes
- Existing `accounts` and `rotation` fields become active runtime policy even
  when `quota` is omitted.
- Account-only service config must satisfy credential pool eligibility rules
  from `add-credential-pool-eligibility`.
- Quota usage endpoint includes services with account pools even when quota is
  omitted.

### Migration Required
- [ ] Database migration.
- [ ] API version bump.
- [x] User communication needed.
- [x] Documentation updates.
