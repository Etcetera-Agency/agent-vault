# Egress Quota Account Identity

Agent Vault quota account pools use two non-secret identifiers:

- `accounts[].id` is the stable account identity. Runtime counters, cooldowns,
  round-robin cursors, request-log seeding, and quota usage responses are keyed
  by this value.
- `accounts[].credential_key` is only the credential reference used to resolve
  and inject the secret value for the selected account.

Keep `id` stable when rotating a credential key behind the same upstream
account. Changing the credential key alone does not reset usage or cooldown
state for that account.

Request logs may include `account_id` for matched account-pool traffic. Logs and
quota usage responses return account ids and credential key names only; they
never include credential values.
