# Proposal: Add Credential Pool Eligibility

## Why

Some credentials are identity-bound and must never be pooled with other accounts
for the same provider. Examples: mail proxy credentials, Gmail, Google Calendar,
and other user/workspace-scoped OAuth tokens. Other provider credentials, such as
interchangeable data-provider API tokens, may be intentionally pooled.

Account pooling therefore needs an explicit credential-level opt-in, not only a
service-level `accounts` list.

## What Changes

- Add non-secret credential metadata `pool_provider`.
- Empty `pool_provider` means the credential is not eligible for account pools.
- A service with `accounts` MUST declare `account_pool_provider`.
- Every `accounts[].credential_key` MUST reference a credential whose
  `pool_provider` equals the service `account_pool_provider`.
- UI lets operators set/clear a credential's pool provider.
- UI disables or rejects service account-pool rows that use credentials not
  eligible for that provider.
- Mail proxy/Gmail/Google Calendar credentials default to no `pool_provider`.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md`

### Affected Code
- `internal/store` - Add credential metadata column(s) and read/write methods.
- `internal/server/handle_credentials.go` - Read/update credential pool provider.
- `internal/server/handle_services.go` - Validate service account credentials
  against credential pool metadata.
- `internal/broker/broker.go` - Add service `account_pool_provider` config.
- `web/src/pages/vault/CredentialsTab.tsx` - Credential-level pool provider UI.
- `web/src/pages/vault/ServicesTab.tsx` - Account-pool provider selector and
  credential filtering/error messages.
- SDK/types/docs for service and credential surfaces.

### User Impact
- Operators explicitly mark which credentials may be combined into provider
  pools.
- Identity-bound credentials remain isolated by default.

### API Changes
- Credential list/update surfaces expose non-secret `pool_provider`.
- Service config accepts `account_pool_provider` when `accounts` is non-empty.
- Service save rejects account pools whose credentials are not eligible for the
  selected provider.

### Migration Required
- [x] Database migration for credential pool metadata.
- [ ] API version bump.
- [x] User communication needed.
- [x] Documentation updates.

