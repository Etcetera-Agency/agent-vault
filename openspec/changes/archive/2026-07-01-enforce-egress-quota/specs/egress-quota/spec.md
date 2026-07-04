# Spec Delta: Egress Quota

## ADDED Requirements

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
apply no egress enforcement to a service that declares no `quota`.

#### Scenario: Only Monthly Cap Configured
GIVEN `apify` declares only `quota.monthly_cap: 100000`
WHEN an agent proxies `apify` requests
THEN Agent Vault enforces only the monthly cap
AND applies no daily, rate, or concurrency limit.

#### Scenario: Unconfigured Service Unaffected
GIVEN service `internal-status` has no `quota`
WHEN an agent proxies any number of requests to it
THEN Agent Vault forwards every request with no egress enforcement.
