# Proposal: Enforce Service Method Matching

## Why

After services can store `methods`, Agent Vault must enforce them at proxy time. Agents need `GET` Gmail/Calendar reads without accidentally allowing writes to the same host/path shape.

**Context**:
- `SERVICE_API_PROXY_PLAN.md` originally sketches `MatchService(host, port, path, method, services)`.
- Current `internal/brokercore/credential.go` calls `broker.MatchService(matchHost, targetPort, targetPath, services)`.
- MITM code already knows request method for HTTPS CONNECT and absolute-form HTTP paths.
- This repo is a fork of upstream `github.com/Infisical/agent-vault`; method enforcement must minimize rebase conflict surface.

**Current state**: A service that matches host/path permits every HTTP method.

**Desired state**: If a service defines methods, Agent Vault only matches that service for listed methods and denies host/path matches with method mismatch under strict policy.

## What Changes

- Keep upstream `broker.MatchService` signature unchanged.
- Add a small local-fork helper that calls existing host/path/port matching, then checks request method on the matched service.
- Treat omitted `methods` as all methods only for services that intentionally omit method policy.
- Detect matched services that fail only because of method and return a clear method-denied error before credential injection.
- Ensure strict deny returns HTTP 403 for method mismatch without falling through to a broader or unmatched passthrough path.
- Add unit and proxy tests for `GET` allowed and `POST` denied on same host/path.

## Impact

### Affected Specifications
- `openspec/specs/service-policy/spec.md` - Adds method-aware proxy enforcement.

### Affected Code
- `internal/broker/method_policy.go` - New local-fork method policy helper wrapping existing `MatchService`.
- `internal/broker/*_test.go` - Fork-local match score and method tests without editing upstream test files.
- `internal/brokercore/credential.go` - Small `// fork-local:` hook to pass request method into post-match policy.
- `internal/brokercore/credential_test.go` - Credential injection and deny behavior.
- `internal/mitm/proxy.go` and tests if method plumbing is missing at boundary.

### User Impact
- Services with `methods` become enforceable least-privilege rules.
- Agents can use API SDKs through normal proxy env while Agent Vault blocks disallowed methods.

### API Changes
- Existing proxy responses may return 403 for method mismatches.
- Error body should identify method policy mismatch without exposing credentials.

### Migration Required
- [ ] Database migration
- [ ] API version bump
- [x] User communication needed
- [x] Documentation updates

## Timeline Estimate

Medium.

## Risks

- Matching fallback could accidentally select a broader service after a narrow method mismatch. Mitigation: tests must cover narrow host/path GET-only service plus broader same-host rule.
