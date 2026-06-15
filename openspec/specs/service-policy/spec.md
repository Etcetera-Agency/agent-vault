# service-policy Specification

## Purpose
TBD - created by archiving change add-service-method-field. Update Purpose after archive.
## Requirements
### Requirement: Service Method Field
WHEN a service is created or updated, the system SHALL accept an optional `methods` field containing uppercase HTTP methods.

#### Scenario: Method Field Stored
GIVEN an admin submits a service with `methods: ["GET", "HEAD"]`
WHEN the service is saved
THEN the stored service includes `methods: ["GET", "HEAD"]`
AND service list output returns those methods.

#### Scenario: Lowercase Methods Normalized
GIVEN an admin submits a service with `methods: ["get", "post"]`
WHEN the service is validated
THEN the system stores `methods: ["GET", "POST"]`.

#### Scenario: Invalid Methods Rejected
GIVEN an admin submits a service with duplicate, empty, or invalid method tokens, or with `*` mixed alongside other methods
WHEN the service is validated
THEN the system rejects the service with a validation error
AND the previous service config remains unchanged.

#### Scenario: Omitted Methods Mean Any Method
GIVEN a service is submitted without a `methods` field
WHEN the service is saved
THEN the stored service has no method restriction
AND the field is treated as allowing any method.

#### Scenario: Wildcard Token Means Any Method
GIVEN an admin submits a service with `methods: ["*"]`
WHEN the service is validated
THEN the system accepts it as unrestricted
AND normalizes it to an empty (no-restriction) method set.

#### Scenario: Proposal Methods Survive Apply
GIVEN an agent raises a proposal for a service with `methods: ["GET"]`
WHEN a human applies the proposal
THEN the merged broker service includes `methods: ["GET"]`
AND a later proposal that omits `methods` clears the restriction back to any method.

### Requirement: Method Policy Discovery
WHEN an agent calls `/discover`, the system SHALL include service method allowlists without exposing credential values, representing an unrestricted service as the canonical `["*"]` token.

#### Scenario: Discover Includes Methods
GIVEN service `calendar-events-read` has `methods: ["GET"]`
WHEN an agent calls `/discover`
THEN the service entry includes `methods: ["GET"]`
AND the entry includes service name and host/path pattern.

#### Scenario: Discover Renders Unrestricted As Wildcard
GIVEN a service has no stored method restriction
WHEN an agent calls `/discover`
THEN the service entry includes `methods: ["*"]`
AND the agent reads `["*"]` as "any method allowed".

#### Scenario: Discover Omits Credential Values
GIVEN service `calendar-events-read` uses credential key `GOOGLE_ACCESS_TOKEN`
WHEN an agent calls `/discover`
THEN the response may include credential key names
AND the response does not include credential values.

### Requirement: Method Policy Management
WHEN an operator creates, edits, lists, or reviews services, the system SHALL expose method policy in CLI and UI management surfaces.

#### Scenario: CLI Round Trip
GIVEN an operator defines a service with `methods: ["GET"]` through CLI YAML or flags
WHEN the operator lists services
THEN CLI output includes `methods: ["GET"]`.

#### Scenario: UI Edit Round Trip
GIVEN an operator edits a service in the web UI
WHEN the operator selects `GET` and saves
THEN the persisted service includes `methods: ["GET"]`
AND the service table displays the method policy.

### Requirement: Method-Aware Service Match
WHEN a proxied request is evaluated against service policy, the system SHALL match request method against the service `methods` allowlist when it is present.

#### Scenario: Listed Method Allowed
GIVEN service `calendar-events-read` matches host/path
AND service `calendar-events-read` has `methods: ["GET"]`
WHEN an agent sends `GET /calendar/v3/calendars/primary/events`
THEN Agent Vault matches service `calendar-events-read`
AND proceeds with configured credential injection.
#### Scenario: Unlisted Method Denied
GIVEN service `calendar-events-read` matches host/path
AND service `calendar-events-read` has `methods: ["GET"]`
WHEN an agent sends `POST /calendar/v3/calendars/primary/events`
THEN Agent Vault returns HTTP 403
AND does not forward the request upstream.

### Requirement: Method Mismatch Fails Closed
WHEN a request matches a configured service by host/path/port but fails method policy, the system SHALL deny the request instead of treating it as unmatched passthrough.

#### Scenario: Passthrough Cannot Bypass Method Policy
GIVEN unmatched host policy is `passthrough`
AND a service matches `www.googleapis.com/calendar/v3/calendars/*/events*` with `methods: ["GET"]`
WHEN an agent sends `POST` to the same host/path
THEN Agent Vault returns HTTP 403
AND does not pass the request through.

#### Scenario: Broader Rule Does Not Override Narrow Method Deny
GIVEN a narrow service matches `gmail.googleapis.com/gmail/v1/users/me/messages*` with `methods: ["GET"]`
AND a broader, less-specific service matches `gmail.googleapis.com/*` with no method restriction
WHEN an agent sends `POST` to `/gmail/v1/users/me/messages`
THEN Agent Vault returns HTTP 403 because the most-specific match excludes `POST`
AND the broader any-method service does NOT rescue the request.

#### Scenario: Omitted Methods Allow Any Method
GIVEN a service matches host/path with no `methods` field
WHEN an agent sends any HTTP method to that host/path
THEN Agent Vault matches the service
AND proceeds with configured credential injection.

### Requirement: Service Allowlist UI Editor
WHEN an operator manages vault services in the web UI, the system SHALL provide an editor for service allowlist rules including host/path, methods, enabled state, auth config, and substitutions.

#### Scenario: Edit Method-Scoped Google Rule
GIVEN service `calendar-events-read` exists with `methods: ["GET"]`
WHEN an operator opens the service editor and changes the path pattern
THEN the service is saved with the new path pattern
AND `methods: ["GET"]` remains present
AND the auth credential key reference remains present.

#### Scenario: Edit Telegram Substitution Rule
GIVEN service `telegram-bot-api` uses path placeholder substitution
WHEN an operator edits the service in the UI and saves
THEN the saved service preserves the placeholder, credential key, substitution location, method policy, and passthrough auth type.

#### Scenario: Invalid Rule Rejected
GIVEN an operator enters an invalid method set or duplicate service name
WHEN the operator saves the form
THEN the system rejects the change
AND the previous service rule remains unchanged
AND the UI displays the validation error.

### Requirement: Credential Values Hidden In Editor
WHEN service allowlists reference stored credentials, the system SHALL show credential key names in the editor without showing credential values.

#### Scenario: Credential Selector Shows Keys Only
GIVEN credentials exist for `GOOGLE_ACCESS_TOKEN` and `DISCORD_BOT_TOKEN`
WHEN an operator opens the service editor
THEN the editor can display those credential key names
AND the editor does not display decrypted credential values.

#### Scenario: Save Does Not Require Secret Value
GIVEN a service references credential key `GOOGLE_ACCESS_TOKEN`
WHEN an operator edits only host/path or methods
THEN the service can be saved without re-entering the credential value.
