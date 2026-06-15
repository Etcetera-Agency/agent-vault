# Spec Delta: Service Policy

## ADDED Requirements

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
