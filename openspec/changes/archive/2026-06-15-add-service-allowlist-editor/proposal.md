# Proposal: Add Service Allowlist Editor

## Why

Method-scoped agent allowlists must be editable in Agent Vault UI. Operators need to review and adjust service rules without hand-editing YAML for every host/path/method/auth/substitution change.

**Context**:
- Agent profile rules use host, optional path glob, HTTP methods, auth config, and substitutions.
- `surface-service-methods` exposes method policy, but allowlist editing needs complete rule editing.
- Agent Vault already has service management APIs and a vault services UI surface.

**Current state**: Operators can manage services through APIs/CLI, but agent allowlist editing is not specified as a first-class UI workflow.

**Desired state**: The web UI provides a service allowlist editor that can create, edit, disable, and delete method-scoped service rules without exposing credential values.

## What Changes

- Add an allowlist editor to the vault services UI for service name, host/path, methods, enabled state, auth type, credential key references, and substitutions.
- Use existing service create/update/delete APIs where possible.
- Validate UI input with the same rules as backend service validation.
- Preserve credential secrecy: UI can show credential key names, never values.
- Add tests for editing method-scoped agent-style rules.

## Impact

### Affected Specifications
- `openspec/specs/service-policy/spec.md` - Adds UI editing workflow for service allowlists.

### Affected Code
- `web/src/pages/vault/ServicesTab.tsx` - Service list and edit entry points.
- `web/src/components/*` - Form controls for methods, auth config, substitutions if factored out.
- `web/src/lib/api.ts` - Service update payload types if needed.
- `internal/server/handle_services.go` - Reuse or verify API support for edit/delete/disable.
- Tests for service UI/API editing.

### User Impact
- Operators can safely edit agent allowlists from UI.
- Agents can rely on changed allowlists after `/discover` refresh.

### API Changes
- No new API required if existing service APIs can update all fields.
- If existing APIs cannot update a single service atomically, add a focused update endpoint or extend existing patch behavior.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Medium.

## Risks

- Partial UI edits could drop auth or substitution fields. Mitigation: edit form must round-trip full service object and tests must cover substitutions.
- UI may imply access to credential values. Mitigation: display key names only and use credential selectors from available keys.
