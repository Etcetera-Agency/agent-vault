## MODIFIED Requirements

### Requirement: Multi-Account Rotation
The system SHALL select a non-exhausted, non-cooling account from a matched
service's `accounts` pool per the service `rotation` policy at credential
resolve, before injection. The system SHALL use `accounts[].id` as the stable
account identity for usage counters, cooldowns, round-robin state, exhaustion
state, and usage display, and SHALL use `accounts[].credential_key` only to
resolve and inject the credential value.

#### Scenario: Least-Used Selection
GIVEN `apify` has accounts `acct1` (4000 calls) and `acct2` (10 calls) and
`rotation: least_used`
WHEN an agent proxies an `apify` request
THEN Agent Vault selects `acct2` by account id
AND injects the credential referenced by `acct2.credential_key`.

#### Scenario: Skip Exhausted Account
GIVEN `acct1` is at its cap and `acct2` is under its cap
WHEN an agent proxies an `apify` request
THEN Agent Vault selects `acct2` by account id
AND does not select `acct1`.

#### Scenario: Credential Key Differs From Account Id
GIVEN account `acct1` uses `credential_key: APIFY_TOKEN_PRIMARY`
AND account `acct2` uses `credential_key: APIFY_TOKEN_SECONDARY`
WHEN Agent Vault records usage, cooldown, or exhaustion for the selected account
THEN the quota state is keyed by `acct1` or `acct2`
AND the credential key is used only for credential lookup and safe metadata.

### Requirement: Usage And Exhaustion Visible In UI
The system SHALL surface current per-account daily and monthly usage and
exhaustion or cooldown state for services with quota or account-pool config in
the management UI. Usage and state SHALL be keyed by service account id and SHALL
never display credential values.

#### Scenario: Show Usage Against Cap
GIVEN service `apify` has account `acct1` with configured caps
WHEN an operator views `apify` in the UI
THEN the UI shows `acct1` current daily and monthly usage against its cap.

#### Scenario: Show Exhausted Account
GIVEN account `acct1` is at its daily cap or in cooldown
WHEN an operator views `apify` in the UI
THEN the UI marks `acct1` as exhausted or cooling
AND indicates when it becomes available again.

#### Scenario: Credential Key Not Used As Account Label
GIVEN account `acct1` uses `credential_key: APIFY_TOKEN_PRIMARY`
WHEN an operator views quota usage
THEN the row identity is `acct1`
AND the UI does not treat `APIFY_TOKEN_PRIMARY` as the quota account identity.

