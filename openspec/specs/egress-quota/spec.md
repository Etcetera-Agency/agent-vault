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

### Requirement: Per-Credential Quota Caps
The system SHALL deny a proxied egress request when the matched service declares
a `quota` and the credential has reached any configured window cap — `daily_cap`
for the current day or `monthly_cap` for the current month.

#### Scenario: Under Cap Allowed
GIVEN service `apify` has `quota.daily_cap: 5000` and 4999 recorded calls today
WHEN an agent proxies a request that matches `apify`
THEN Agent Vault injects the credential and forwards the request
AND records the call, reaching 5000.

#### Scenario: Daily Cap Denied
GIVEN 5000 recorded calls today and `daily_cap` is 5000
WHEN an agent proxies another request matching `apify`
THEN Agent Vault returns HTTP 429 with `X-Vault-Quota-Exhausted: apify`
AND does not forward the request or inject any credential.

#### Scenario: Monthly Cap Denied Even When Under Daily
GIVEN `quota.daily_cap: 5000` and `quota.monthly_cap: 100000`
AND 200 calls today but 100000 calls this month
WHEN an agent proxies another request matching `apify`
THEN Agent Vault denies with HTTP 429 despite the daily count being under cap.

#### Scenario: Counts Survive Restart
GIVEN 4000 calls today and 40000 this month recorded in the request log
WHEN Agent Vault restarts
THEN the day and month counters are seeded from the request log to 4000 and 40000
AND neither window resets to its cap.

### Requirement: Egress Rate And Concurrency Limits
The system SHALL apply the configured per-credential `rpm` and `concurrency`
limits to egress calls before forwarding upstream.

#### Scenario: Rate Limited
GIVEN `quota.rpm: 60` and the rate tokens for `apify` are consumed
WHEN an agent proxies another `apify` request in the same window
THEN Agent Vault returns HTTP 429 with `Retry-After`
AND does not forward the request.

#### Scenario: Concurrency Capped
GIVEN `quota.concurrency: 4` and 4 `apify` requests are in flight
WHEN a 5th arrives
THEN Agent Vault never exceeds 4 concurrent upstream calls for `apify`.

### Requirement: Enforcement Only For Configured Limits
The system SHALL enforce only the limit fields a service configures and SHALL
apply no egress quota enforcement to a service that declares no `quota` and no
`accounts`. When a service declares `accounts` without `quota`, the system SHALL
apply account selection, cooldown, failover, and usage tracking while applying no
daily, monthly, rate, or concurrency limit, provided every account credential is
eligible for the service account-pool provider.

#### Scenario: Only Monthly Cap Configured
GIVEN `apify` declares only `quota.monthly_cap: 100000`
WHEN an agent proxies `apify` requests
THEN Agent Vault enforces only the monthly cap
AND applies no daily, rate, or concurrency limit.

#### Scenario: Account-Only Rotation Has No Caps
GIVEN `apify` declares two accounts and `rotation: round_robin`
AND declares no `quota`
AND every account credential has matching `pool_provider: apify`
WHEN agents proxy `apify` requests
THEN Agent Vault rotates between the configured accounts
AND applies no daily, monthly, rpm, or concurrency limit.

#### Scenario: Unconfigured Service Unaffected
GIVEN service `internal-status` has no `quota` and no `accounts`
WHEN an agent proxies any number of requests to it
THEN Agent Vault forwards every request with no egress quota or account-pool
enforcement.

### Requirement: Multi-Account Rotation
The system SHALL select a non-exhausted, non-cooling account from a matched
service's `accounts` pool per the service `rotation` policy at credential
resolve, before injection. This selection SHALL run for account-pool services
even when the service has no `quota`.

#### Scenario: Least-Used Selection
GIVEN `apify` has accounts `acct1` (4000 calls) and `acct2` (10 calls) and
`rotation: least_used`
WHEN an agent proxies an `apify` request
THEN Agent Vault selects `acct2` and injects its credential.

#### Scenario: Round-Robin Without Quota
GIVEN `apify` has accounts `acct1` and `acct2`
AND `rotation: round_robin`
AND no `quota`
WHEN two agents proxy two sequential `apify` requests
THEN Agent Vault injects `acct1` for the first request
AND injects `acct2` for the second request.

#### Scenario: Skip Cooling Account Without Quota
GIVEN `acct1` is cooling after an upstream 429
AND `acct2` is available
AND service `apify` has no `quota`
WHEN an agent proxies an `apify` request
THEN Agent Vault selects `acct2`
AND does not select `acct1`.

### Requirement: Failover On Upstream Rejection
The system SHALL place the selected account in cooldown and retry with the next
available account within a bounded number of attempts, WHEN the upstream returns
HTTP 429 or a quota error for that account. The system SHALL perform this
failover for any replayable proxied request, including requests with materialized
bodies, while preserving request method, URL, headers, and body across attempts
except for account-specific credential injection.

#### Scenario: Failover To Next Account
GIVEN `apify` has accounts `acct1` and `acct2`, both under cap
WHEN Agent Vault forwards using `acct1`
AND the upstream returns HTTP 429 with `Retry-After: 120`
THEN Agent Vault cools down `acct1` for at least 120 seconds
AND retries the request using `acct2`.

#### Scenario: Failover Replays POST Body
GIVEN `apify` has accounts `acct1` and `acct2`, both under cap
AND an agent sends a POST request with JSON body `{"task":"run"}`
WHEN Agent Vault forwards using `acct1`
AND the upstream returns HTTP 429
THEN Agent Vault retries using `acct2`
AND the retry preserves method `POST`
AND the retry preserves body `{"task":"run"}`.

#### Scenario: Cooldown Honored On Next Request
GIVEN `acct1` was cooled down 30 seconds ago for 120 seconds
WHEN a new `apify` request arrives
THEN Agent Vault does not select `acct1`
AND selects another available account or denies if none.

#### Scenario: Retry Attempts Are Bounded
GIVEN every account in `apify` returns upstream HTTP 429
WHEN Agent Vault attempts failover
THEN Agent Vault stops after the bounded retry count
AND returns HTTP 429 without an unbounded retry loop.

### Requirement: Pool Exhaustion Signal
The system SHALL deny the request with HTTP 429, an `X-Vault-Quota-Exhausted`
header, and a `Retry-After`, and SHALL emit at most one operator alert per service
per cooldown window, WHEN every account for a matched service is over cap or in
cooldown.

#### Scenario: All Accounts Exhausted
GIVEN both `acct1` and `acct2` for `apify` are at cap
WHEN an agent proxies an `apify` request
THEN Agent Vault returns HTTP 429 with `X-Vault-Quota-Exhausted: apify`
AND `Retry-After` set to the earliest reset
AND the body names exhaustion without exposing any credential.

#### Scenario: Single Alert Per Window
GIVEN `apify` just became fully exhausted and an alert was emitted
WHEN further `apify` requests arrive in the same cooldown window
THEN Agent Vault denies each with HTTP 429
AND emits no additional operator alerts for `apify` in that window.

### Requirement: Quota And Accounts Editable In Service UI
The system SHALL let an operator view and edit a service's `quota` caps and
`accounts` pool through the management UI, persisting them via the service edit
API, without ever displaying credential values.

#### Scenario: Edit Quota Caps
GIVEN an operator opens service `apify` in the management UI
WHEN the operator sets `daily_cap` to 5000 and `monthly_cap` to 100000 and saves
THEN the service edit API persists the quota
AND subsequent egress enforcement uses the saved caps.

#### Scenario: Manage Account Pool
GIVEN service `apify` in the UI
WHEN the operator adds an account `acct2` with credential key `APIFY_TOKEN_2`,
sets `rotation` to `least_used`, and saves
THEN the service has a two-account pool with least-used rotation
AND the value of `APIFY_TOKEN_2` is never shown in the editor.

#### Scenario: Empty Quota Leaves Service Unlimited
GIVEN an operator edits `internal-status` and leaves all quota fields blank
WHEN the operator saves
THEN the service persists with no quota
AND egress enforcement applies no limits to it.

### Requirement: Usage And Exhaustion Visible In UI
The system SHALL surface current per-account daily and monthly usage and
exhaustion or cooldown state for services with quota or account-pool config in
the management UI. For account-only services, the system SHALL show account
state and usage counters without caps. The system SHALL never display credential
values.

#### Scenario: Show Usage Against Cap
GIVEN service `apify` has accounts with configured caps
WHEN an operator views `apify` in the UI
THEN the UI shows each account's current daily and monthly usage against its cap.

#### Scenario: Show Exhausted Account
GIVEN account `acct1` is at its daily cap or in cooldown
WHEN an operator views `apify` in the UI
THEN the UI marks `acct1` as exhausted or cooling
AND indicates when it becomes available again.

#### Scenario: Show Account-Only Pool State
GIVEN service `apify` has accounts `acct1` and `acct2`
AND has no `quota`
WHEN an operator views the Services table
THEN the quota column shows account rows and their current state
AND does not show only a generic configured label.

#### Scenario: Blank Quota With Accounts Remains Valid
GIVEN an operator adds two accounts in the UI
AND leaves all quota fields blank
WHEN the operator saves the service
THEN the service persists with account-only rotation
AND the UI shows account state for the service after reload.

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

