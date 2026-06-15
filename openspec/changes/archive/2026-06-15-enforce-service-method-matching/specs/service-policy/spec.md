# Spec Delta: Service Policy

## ADDED Requirements

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
