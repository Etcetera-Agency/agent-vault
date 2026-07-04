# Implementation Tasks

- [x] 1. Add `,omitempty` service-config fields in `internal/catalog`
      (tail-appended, `// fork-local:`): `Quota{DailyCap,MonthlyCap,RPM,
      Concurrency}`, `Accounts[]{ID,CredentialKey,DailyCap,MonthlyCap,RPM}`,
      `Rotation`.
- [x] 2. Add a fork-local validation helper (non-negative; `monthly_cap >=
      daily_cap`; unique account ids; `credential_key` exists). Empty
      quota/accounts are valid.
- [x] 3. Ensure the service create/edit API round-trips the new fields without
      dropping existing fields; add a minimal `// fork-local:` pass-through only
      if needed.
- [x] 4. Round-trip + validation tests in new `*_test.go`: full config,
      monthly-only, rpm-only, empty quota, multi-account, invalid (monthly<daily,
      dup id, unknown key).
- [x] 5. Docs: service config reference for quota/accounts/rotation; note fields
      are optional and enforcement arrives in a later slice.
