# Design: Service Quota and Account Config Fields

## Fork-Conflict-Surface Choice

- Only additive `,omitempty` fields at the tail of the upstream service struct in
  `internal/catalog`, each marked `// fork-local:`. No reorder/rename/reformat.
- Validation lives in a new fork-local file, not inside upstream bodies.
- API round-trip is achieved by the existing full-service update path already
  accepting the struct; if it drops unknown fields, add a minimal `// fork-local:`
  pass-through rather than changing signatures.
- Tests in new `*_test.go` files only.

## Config Shape (additive, `,omitempty`)

```yaml
services:
  - name: apify
    # ...existing host/path/method/auth policy...
    quota:                 # optional; absence = no quota
      daily_cap: 5000      # optional
      monthly_cap: 100000  # optional
      rpm: 60              # optional
      concurrency: 4       # optional
    rotation: least_used   # optional: least_used | round_robin
    accounts:              # optional; absence = single existing credential
      - id: acct1
        credential_key: APIFY_TOKEN_1
        daily_cap: 5000    # optional per-account override
        monthly_cap: 80000 # optional per-account override
      - id: acct2
        credential_key: APIFY_TOKEN_2
```

## Validation (config-time only, no enforcement)

- Numbers non-negative; `monthly_cap >= daily_cap` when both present.
- Account ids unique; each `credential_key` resolves in the credential store.
- Empty `quota` and empty `accounts` are valid (mean "no quota", "single cred").

## Non-Goals

- No enforcement, counters, rotation, exhaustion, or UI — those are separate
  slices.
