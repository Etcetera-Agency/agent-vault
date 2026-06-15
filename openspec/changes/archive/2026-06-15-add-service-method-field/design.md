# Design: Service Method Field

## Fork-Conflict Surface

This repo is a fork of upstream Agent Vault. Keep this schema slice additive:

- append `methods,omitempty` fields at the end of existing service structs
- put normalization/validation in a new helper file, e.g. `internal/broker/methods.go`
- avoid reformatting existing struct bodies or proposal merge logic
- mark required upstream-file touches with `// fork-local:`
- add tests in new fork-local `*_test.go` files

## Service Shape

```yaml
services:
  - name: calendar-events-read
    host: www.googleapis.com/calendar/v3/calendars/*/events*
    methods: [GET]
    auth:
      type: bearer
      token: GOOGLE_ACCESS_TOKEN
```

## Validation Pseudocode

```text
normalizeMethods(methods):
  # "*" is the canonical "any method" token. It is valid ONLY as the sole
  # element and normalizes to an empty list — the single internal
  # representation of "unrestricted". Surfaces render empty back as ["*"].
  if methods (trimmed) == ["*"]:
    return []   # unrestricted
  seen = empty set
  normalized = empty list
  for method in methods:
    trimmed = strings.TrimSpace(method)
    upper = strings.ToUpper(trimmed)
    if upper == "":
      error "method must not be empty"
    if upper == "*":
      error "wildcard '*' is only valid as the sole method"
    if upper contains non-token char:
      error "invalid HTTP method"
    if upper in seen:
      error "duplicate HTTP method"
    add upper to seen and normalized
  return normalized
```

## Proposal Round-Trip

Agents never edit services directly; they raise proposals that a human applies.
`proposal.Service` (`internal/proposal/proposal.go`) is a separate struct from
`broker.Service`, and `toBrokerService` (`internal/proposal/merge.go`) rebuilds a
`broker.Service` on every upsert. So `methods` must be added to **both** structs and
copied in `toBrokerService`, or proposed methods are silently dropped at merge time.

Merge-preserve semantics differ from `Substitutions`. An empty `Substitutions`
list on an upsert means "leave existing alone" (callers clear by delete+recreate).
`methods` must NOT inherit that rule: an empty/omitted `methods` legitimately means
"any method", so an upsert SHALL set `methods` exactly as proposed. This keeps a
service widenable back to any-method through a normal proposal.

## Non-Goal

No proxy deny/allow behavior changes in this slice. Enforcement belongs to `enforce-service-method-matching`.
