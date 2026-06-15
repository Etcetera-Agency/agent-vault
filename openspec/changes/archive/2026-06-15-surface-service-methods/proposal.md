# Proposal: Surface Service Methods

## Why

Once Agent Vault stores and enforces service methods, operators and agents must see method policy in every management surface. Hidden method policy makes agent integrations hard to audit and can cause agents to propose or call wrong operations.

**Context**:
- `/discover` currently returns service `name` and joined `host` pattern.
- CLI service list/set/add and interactive builder are primary operator paths.
- Web UI service forms and service tables are primary visual audit paths.

**Current state**: No management surface displays method allowlists.

**Desired state**: CLI, API, UI, docs, and `/discover` expose method policy consistently without exposing credential values.

## What Changes

- Include `methods` in `/discover` service entries, rendering unrestricted (empty/omitted) services as the canonical `["*"]` token consistently across API, CLI, UI, and docs.
- Display and edit `methods` in CLI service commands.
- Display and edit `methods` in web service forms/tables.
- Update docs and examples with method-scoped services.
- Add tests for API and UI/CLI serialization where available.

## Impact

### Affected Specifications
- `openspec/specs/service-policy/spec.md` - Adds management-surface visibility.

### Affected Code
- `internal/server/handle_discovery.go` - Add methods to discovery response.
- `cmd/service.go` and `cmd/service_interactive.go` - CLI method flags/prompts/output.
- `web/src/pages/vault/ServicesTab.tsx` and service components - UI method controls.
- `docs/learn/services.mdx`, `docs/agents/protocol.mdx`, `docs/reference/cli.mdx` - Docs.
- SDK types if service/discover response types are exported.

### User Impact
- Operators can audit and edit method-scoped rules.
- Agents can inspect `/discover` and avoid attempting disallowed methods.

### API Changes
- `/discover` service entries gain optional `methods`.
- Service list responses already expose service objects and should include `methods`.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Medium.

## Risks

- UI can accept invalid method state. Mitigation: reuse backend validation and add client-side method token constraints.
