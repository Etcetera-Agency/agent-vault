# Proposal: Add Service Method Field

## Why

Agents need read/write separation for third-party APIs where host and path alone are not enough. Gmail and Calendar can share the same path shape across methods, so Agent Vault needs an explicit service method allowlist before agents can safely receive scoped integrations.

**Context**:
- `SERVICE_API_PROXY_PLAN.md` names method-level service matching as the current Agent Vault gap.
- Current `broker.Service` matches host, optional port, and path glob.
- Existing service JSON/YAML has no first-class `methods` field.

**Current state**: Service policy cannot represent `GET`-only or `POST`-only access.

**Desired state**: Service policy can store, validate, normalize, and round-trip an HTTP method allowlist without changing matching behavior yet.

## What Changes

- Add `methods` to `broker.Service` and to `proposal.Service` as a list of uppercase HTTP methods, and carry it through `toBrokerService` so proposed methods survive merge.
- Validate methods on service ingest from JSON/YAML, proposals, direct service set/add, and interactive service builder output.
- Normalize lower-case input to uppercase and reject empty strings, duplicate methods, and invalid HTTP method tokens. Accept `["*"]` as the canonical "any method" token (valid only as the sole element) and normalize it to an empty list; reject `*` mixed with other methods.
- Preserve service marshal/unmarshal behavior through existing config APIs and CLI YAML.
- Add tests proving method field round-trips without enforcing it.

## Impact

### Affected Specifications
- `openspec/specs/service-policy/spec.md` - Adds service method policy representation.

### Affected Code
- `internal/broker/broker.go` - `Service` struct, validation, normalization helpers.
- `internal/proposal/proposal.go` - `Service` struct gains `Methods`.
- `internal/proposal/merge.go` - `toBrokerService` copies `Methods` onto the broker service.
- `internal/proposal/validate.go` - Proposal service validation.
- `cmd/service.go` and `cmd/service_interactive.go` - CLI input/output support.
- `internal/server/handle_services.go` and tests - Direct service API validation and round-trip.

### User Impact
- Admins can define method-scoped services in config, but this slice does not enforce methods yet.

### API Changes
- Adds optional `methods` field to service JSON/YAML.
- No behavior change to proxy matching until enforcement slice.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Small.

## Risks

- Accepting `methods` before enforcement could create false confidence. Mitigation: docs and API response must mark it as stored but not enforced until `enforce-service-method-matching` lands.
