# Proposal: Fix Quota Account Identity

## Why

The egress quota runtime currently uses `credential_key` as the account identity
for counters, cooldowns, round-robin state, request-log seeding, and usage
snapshots. The service model and spec define account identity as `accounts[].id`
(`acct1`, `acct2`) while `credential_key` is only the secret reference to inject.

This mismatch makes the UI display one identity while enforcement tracks another,
and it breaks stable usage/cooldown semantics when an operator rotates the
credential key behind the same account id.

## What Changes

- Treat `accounts[].id` as the canonical quota account identity everywhere in
  runtime state and API usage output.
- Keep `credential_key` only as the credential reference used to resolve and
  inject a secret.
- Extend quota reservations and proxy request logging with `account_id` so
  restart seeding reconstructs account counters from account ids.
- Update least-used, round-robin, cooldown, exhaustion, and usage tests so they
  assert account ids (`acct1`, `acct2`) instead of credential keys.
- Preserve the guarantee that no credential value is stored or returned.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md`

### Affected Code
- `internal/egressquota/registry.go` - Candidate identity, reservation metadata,
  counters, cooldowns, snapshots, request-log seeding.
- `internal/brokercore/credential.go` - Selected account id plus selected
  credential key propagation.
- `internal/mitm/forward.go` - Proxy event/request-log account id emission.
- `internal/requestlog` and `internal/store` - Account id persistence/read model.
- `internal/server/handle_services.go` - Usage read path keyed by account id.
- Tests under `internal/egressquota`, `internal/brokercore`, `internal/mitm`,
  `internal/server`.

### User Impact
- UI usage rows and runtime quota decisions refer to the same account names.
- Renaming a credential key no longer changes quota/cooldown identity when
  account id stays the same.

### API Changes
- Quota usage read path continues returning `id` and `credential_key`; `id` is
  now the authoritative usage/cooldown identity.
- Request logs gain non-secret `account_id` metadata for matched account-pool
  traffic.

### Migration Required
- [x] Database/storage change for request-log account id metadata.
- [ ] API version bump.
- [x] User communication needed.
- [x] Documentation updates.

