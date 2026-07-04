## MODIFIED Requirements

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

