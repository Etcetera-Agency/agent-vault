## MODIFIED Requirements

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
