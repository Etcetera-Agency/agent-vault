# Proposal: Add Agent Service Profile

## Why

After method-scoped service policy exists, Agent Vault needs a concrete agent profile that encodes the first approved service rules from `SERVICE_API_PROXY_PLAN.md`: Gmail read, Calendar read, Discord channel messages, and Telegram bot API substitution.

**Context**:
- Agents should call real upstream APIs/SDKs through Agent Vault's existing proxy flow.
- Real credentials stay in Agent Vault.
- Agent-facing examples use placeholders only.
- Strict deny must be enabled before profile use.

**Current state**: Operators must hand-build agent service rules and may over-broaden host/path/method scope.

**Desired state**: Agent Vault ships a documented/tested agent service profile that produces narrow method-scoped rules and verification cases.

## What Changes

- Add an agent service profile example with method-scoped Gmail, Calendar, Discord, and Telegram rules.
- Add verification fixtures or docs proving allowed and denied routes.
- Add proposal payload examples for missing credentials.
- Add request-log acceptance checks for method/host/path/matched service/status/latency.
- Require agent profile rules to be editable through the service allowlist UI from `add-service-allowlist-editor`.

## Impact

### Affected Specifications
- `openspec/specs/agent-service-profile/spec.md` - Adds agent profile behavior.

### Affected Code
- `examples/` - Agent service YAML/profile.
- `docs/guides/agent-service-profile.mdx` - Profile setup and verification.
- `docs/agents/protocol.mdx` - Method-aware discover usage.
- Tests or fixtures under `internal/broker`, `internal/brokercore`, or docs test harness.

### User Impact
- Operators can apply a known-good agent starting policy.
- Operators can edit agent allowlists without hand-editing YAML.
- Agents can discover method policy and avoid unsupported operations.

### API Changes
- No new API required.
- Uses existing services/proposals/discover endpoints plus method fields from prior slices.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Small after prior method slices.

## Risks

- Real SDK URL shapes may differ from planned globs. Mitigation: profile verification must capture and compare actual SDK/API URLs before profile use.
