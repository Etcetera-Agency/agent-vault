## ADDED Requirements

### Requirement: Credential Pool Eligibility
The system SHALL treat credential account pooling as an explicit credential-level
opt-in. A credential SHALL be poolable only when it has a non-empty
`pool_provider` metadata value. A service with `accounts` SHALL declare
`account_pool_provider`, and every account credential in that service SHALL have
a matching `pool_provider`. Credentials with no `pool_provider` SHALL NOT be
eligible for account pools.

#### Scenario: Poolable Credential Accepted
GIVEN credential `APIFY_TOKEN_1` has `pool_provider: apify`
AND credential `APIFY_TOKEN_2` has `pool_provider: apify`
WHEN an operator saves service `apify` with `account_pool_provider: apify`
AND accounts referencing both credentials
THEN the service save succeeds.

#### Scenario: Unpoolable Credential Rejected
GIVEN credential `GOOGLE_CALENDAR_TOKEN` has no `pool_provider`
WHEN an operator saves service `google-calendar` with an account pool referencing
`GOOGLE_CALENDAR_TOKEN`
THEN the service save is rejected with a validation error
AND the service is not modified.

#### Scenario: Wrong Provider Rejected
GIVEN credential `APIFY_TOKEN_1` has `pool_provider: apify`
WHEN an operator saves service `amadeus` with `account_pool_provider: amadeus`
AND an account referencing `APIFY_TOKEN_1`
THEN the service save is rejected with a validation error
AND the service is not modified.

#### Scenario: Credential Metadata Exposes No Secret
GIVEN credential `APIFY_TOKEN_1` has `pool_provider: apify`
WHEN credentials are listed through the API
THEN the response may include `pool_provider: apify`
AND does not include the credential value unless the caller explicitly uses the
existing reveal flow.

## MODIFIED Requirements

### Requirement: Quota Config Validation
The system SHALL reject invalid quota or account config at save time, allowing an
empty quota and an empty account pool. The system SHALL reject any non-empty
account pool unless the service declares `account_pool_provider` and every
account credential is explicitly enabled for that provider.

#### Scenario: Reject Monthly Below Daily
GIVEN an operator sets `quota.daily_cap: 5000` and `quota.monthly_cap: 1000`
WHEN they save
THEN the save is rejected with a validation error
AND the service is not modified.

#### Scenario: Reject Duplicate Account Id
GIVEN two accounts both with id `acct1`
WHEN the operator saves
THEN the save is rejected with a validation error.

#### Scenario: Reject Account Without Matching Pool Provider
GIVEN credential `GOOGLE_CALENDAR_TOKEN` has no `pool_provider`
WHEN an operator saves a service account pool referencing
`GOOGLE_CALENDAR_TOKEN`
THEN the save is rejected with a validation error
AND the service is not modified.

#### Scenario: Allow Empty Quota
GIVEN an operator saves a service with no quota fields and no accounts
WHEN they save
THEN the save succeeds with no quota config.

