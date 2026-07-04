## MODIFIED Requirements

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

