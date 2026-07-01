# egress-quota Specification

## Purpose
Define optional service quota/account policy, runtime egress quota enforcement,
account rotation, and management visibility for credentialed upstream usage.
## Requirements
### Requirement: Service Quota And Account Config
The system SHALL accept and persist an optional per-service `quota`
(`daily_cap`, `monthly_cap`, `rpm`, `concurrency` — each independently optional),
an optional `accounts` pool, and a `rotation` policy, and SHALL round-trip them
through the service create/edit API without dropping existing service fields.

#### Scenario: Full Config Round-Trips
GIVEN an operator saves service `apify` with `quota.daily_cap: 5000`,
`quota.monthly_cap: 100000`, two accounts, and `rotation: least_used`
WHEN the service is read back through the API
THEN all quota, account, and rotation fields are returned unchanged
AND existing host/path/method/auth fields are preserved.

#### Scenario: Partial Quota Persists
GIVEN service `apify` is saved with only `quota.monthly_cap: 100000` set
WHEN it is read back
THEN only `monthly_cap` is present and the other quota fields remain unset.

#### Scenario: Empty Quota Persists As None
GIVEN service `internal-status` is saved with no `quota` and no `accounts`
WHEN it is read back
THEN it has no quota or account config
AND behaves exactly as a service defined before this capability existed.

### Requirement: Quota Config Never Exposes Credential Values
The system SHALL reference accounts by `credential_key` only and SHALL NOT store
or return credential values through the quota or account config.

#### Scenario: Account References Key Only
GIVEN an account `acct1` with `credential_key: APIFY_TOKEN_1`
WHEN the service config is read back through the API
THEN the response contains `credential_key: APIFY_TOKEN_1`
AND does not contain the value of `APIFY_TOKEN_1`.

### Requirement: Quota Config Validation
The system SHALL reject invalid quota or account config at save time, allowing an
empty quota and an empty account pool.

#### Scenario: Reject Monthly Below Daily
GIVEN an operator sets `quota.daily_cap: 5000` and `quota.monthly_cap: 1000`
WHEN they save
THEN the save is rejected with a validation error
AND the service is not modified.

#### Scenario: Reject Duplicate Account Id
GIVEN two accounts both with id `acct1`
WHEN the operator saves
THEN the save is rejected with a validation error.

#### Scenario: Allow Empty Quota
GIVEN an operator saves a service with no quota fields and no accounts
WHEN they save
THEN the save succeeds with no quota config.
