# Implementation Tasks

## Phase 1: Low-Conflict Matching Gate

- [x] 1. Keep upstream `broker.MatchService(host, port, path, services)` unchanged.
- [x] 2. Add a new local-fork helper in a new file (for example `internal/broker/method_policy.go`) that calls `MatchService`, then returns `ok`, `noMatch`, or `methodDenied`.
- [x] 3. Implement fail-closed post-match precedence: if existing host/path/port matching selects a service and that service excludes the request method, deny immediately instead of falling through to a less-specific rule. Keep the disabled-service check post-match.

## Phase 2: Enforcement

- [x] 4. In credential resolution, add the smallest possible `// fork-local:` hook to call the method-policy helper and deny method mismatch before credential injection.
- [x] 5. Ensure strict deny returns HTTP 403 for method mismatch.
- [x] 6. Ensure passthrough policy does not bypass explicit method mismatch for a configured service.
- [x] 7. Return a clear error code/message for method mismatch.

## Phase 3: Tests

- [x] 8. Add broker tests in a new fork-local `*_test.go` file for GET allowed and POST denied on same host/path.
- [x] 9. Add broker tests for omitted `methods` accepting any method.
- [x] 10. Add tests where a narrow method-scoped service must not fall through to an unsafe broader rule.
- [x] 11. Add brokercore/proxy test for HTTP 403 on method mismatch.

## Phase 4: Docs

- [x] 12. Update service docs with method enforcement examples and failure behavior.
