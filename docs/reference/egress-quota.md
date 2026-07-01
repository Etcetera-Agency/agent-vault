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

## Credential Pool Eligibility

Credentials are not eligible for account pools by default. To use a credential
inside `service.accounts`, set its non-secret `pool_provider` metadata to a
provider slug such as `apify`.

Services with account pools must set `account_pool_provider` to the same slug.
Every `accounts[].credential_key` must reference a credential whose
`pool_provider` exactly matches the service `account_pool_provider`.

This keeps identity-bound credentials, such as Gmail, Google Calendar, and mail
proxy credentials, isolated unless an operator explicitly opts them into a
provider pool.
