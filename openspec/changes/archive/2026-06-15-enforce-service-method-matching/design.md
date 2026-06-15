# Design: Method-Aware Matching

## Fork-Conflict Surface

Keep existing `broker.MatchService(host, port, path, services)` signature intact.
This repo is a fork of upstream Agent Vault, and changing that signature would
force edits across every upstream call site. Method enforcement should be a
local-fork post-match gate in new code, with only a tiny caller edit where the
request method is already available.

Preferred shape:

- add a new helper in a new file, e.g. `internal/broker/method_policy.go`
- keep upstream `MatchService` unchanged
- call existing `MatchService` first to pick the best host/path/port service
- run method policy against that matched service before credential injection
- mark the caller hook with `// fork-local:`
- put new tests in fork-local `*_test.go` files

## Matching Pseudocode

Method must NOT be a plain `continue` filter. If a method-mismatched service is
skipped, a broader any-method service can win and the request is *allowed* — the
fall-through the risk note warns against, and a direct contradiction of the
Security Invariant below. Instead, keep upstream host/path/port matching as the
first step. Then evaluate the method allowlist on the selected best match, and
deny a mismatch instead of re-running match against less-specific services.

```text
MatchServiceWithMethodPolicy(host, port, path, method, services):
  matched, score = MatchService(host, port, path, services) // upstream API unchanged
  if matched == nil:
    return nil, score, noMatch

  if methodAllowed(matched.Methods, method):
    return matched, score, ok

  return matched, score, methodDenied

methodAllowed(methods, method):
  if methods empty:
    return true
  return normalize(method) in methods
```

Disabled services are NOT filtered inside matching — the existing caller
(`brokercore/credential.go`) checks `matched.IsEnabled()` after the match and
returns `ErrServiceDisabled`. Keep that post-match shape; add only the method
policy gate between host/path match and disabled-service/credential handling.

## Deny Pseudocode

```text
ResolveCredential(req):
  service, score, status = MatchServiceWithMethodPolicy(
    req.host,
    req.port,
    req.path,
    req.method,
    services,
  )

  if status == methodDenied:
    return denied("method_not_allowed")   // before any unmatched-host policy check

  if service == nil:
    if unmatched_host_policy == "deny":
      return denied("no_match")
    return passthrough

  if not service.IsEnabled():
    return denied("service_disabled")
  inject credential for service
  return allowed
```

## Security Invariant

A request whose most-specific host/path/port match excludes its method must fail
closed (HTTP 403). It must not fall through to a broader rule and must not pass
through as unmatched traffic, regardless of `unmatched_host_policy`.
