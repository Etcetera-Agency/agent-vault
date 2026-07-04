# Spec Delta: Egress Quota

## ADDED Requirements

### Requirement: Multi-Account Rotation
The system SHALL select a non-exhausted, non-cooling account from a matched
service's `accounts` pool per the service `rotation` policy at credential resolve,
before injection.

#### Scenario: Least-Used Selection
GIVEN `apify` has accounts `acct1` (4000 calls) and `acct2` (10 calls) and
`rotation: least_used`
WHEN an agent proxies an `apify` request
THEN Agent Vault selects `acct2` and injects its credential.

#### Scenario: Skip Exhausted Account
GIVEN `acct1` is at its cap and `acct2` is under its cap
WHEN an agent proxies an `apify` request
THEN Agent Vault selects `acct2` and does not select `acct1`.

### Requirement: Failover On Upstream Rejection
The system SHALL place the selected account in cooldown and retry with the next
available account within a bounded number of attempts, WHEN the upstream returns
HTTP 429 or a quota error for that account.

#### Scenario: Failover To Next Account
GIVEN `apify` has accounts `acct1` and `acct2`, both under cap
WHEN Agent Vault forwards using `acct1`
AND the upstream returns HTTP 429 with `Retry-After: 120`
THEN Agent Vault cools down `acct1` for at least 120 seconds
AND retries the request using `acct2`.

#### Scenario: Cooldown Honored On Next Request
GIVEN `acct1` was cooled down 30 seconds ago for 120 seconds
WHEN a new `apify` request arrives
THEN Agent Vault does not select `acct1`
AND selects another available account or denies if none.

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
