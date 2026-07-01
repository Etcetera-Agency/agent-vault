# Spec Delta: Egress Quota

## ADDED Requirements

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
exhaustion or cooldown state for quota-configured services in the management UI.

#### Scenario: Show Usage Against Cap
GIVEN service `apify` has accounts with configured caps
WHEN an operator views `apify` in the UI
THEN the UI shows each account's current daily and monthly usage against its cap.

#### Scenario: Show Exhausted Account
GIVEN account `acct1` is at its daily cap or in cooldown
WHEN an operator views `apify` in the UI
THEN the UI marks `acct1` as exhausted or cooling
AND indicates when it becomes available again.
