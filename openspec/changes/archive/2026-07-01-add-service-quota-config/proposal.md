# Proposal: Service Quota and Account Config Fields

## Why

Agent Vault services carry host/path/method policy and credential config, but no
notion of an outbound usage ceiling or a pool of interchangeable credentials.
Before Agent Vault can enforce egress quotas or rotate accounts, the service
schema and its management API must be able to **carry and round-trip** that
config.

This slice adds only the config surface — no enforcement, no rotation, no UI
behavior change. It is the schema step that later slices build on
(`enforce-egress-quota`, `add-credential-rotation`, `add-quota-ui`).

**Current state**: A service has no quota or account-pool fields.

**Desired state**: A service MAY declare an optional `quota` (each of
`daily_cap`, `monthly_cap`, `rpm`, `concurrency` independently optional), an
optional `accounts` pool, and a `rotation` policy, and these round-trip through
the service create/edit API without loss.

## What Changes

- Add additive, `,omitempty` service-config fields: `quota`
  (`daily_cap`/`monthly_cap`/`rpm`/`concurrency`, each optional), `accounts[]`
  (`id`, `credential_key`, optional per-account `daily_cap`/`monthly_cap`/`rpm`),
  and `rotation` (`least_used` | `round_robin`).
- Validate the fields (non-negative numbers; `monthly_cap >= daily_cap` when both
  set; unique account ids; each `credential_key` exists) without enforcing usage.
- Ensure the service create/edit API accepts and returns the new fields without
  dropping existing service fields.
- Reference `credential_key` only; never store or return credential values.

## Impact

### Affected Specifications
- `openspec/specs/egress-quota/spec.md` - New capability (config portion).

### Affected Code
- Service config struct in `internal/catalog` - Tail-appended `,omitempty`
  fields, marked `// fork-local:`; no reordering of upstream fields.
- Service create/edit API handler - Additive accept/return of the new fields.
- New fork-local `*_test.go` - Round-trip and validation tests incl. partial and
  empty quota configs.

### User Impact
- Operators can declare quotas/accounts; nothing is enforced yet.

### API Changes
- Service create/edit endpoints accept and return `quota`, `accounts`,
  `rotation` (additive; services without them behave exactly as before).

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Small.

## Risks

- Fork conflict on the service struct. Mitigation: tail-appended `,omitempty`
  fields only, `// fork-local:` marked, no reformatting of surrounding upstream.
