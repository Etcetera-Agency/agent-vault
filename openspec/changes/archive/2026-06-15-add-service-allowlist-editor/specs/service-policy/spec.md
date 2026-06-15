# Spec Delta: Service Policy

## ADDED Requirements

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

