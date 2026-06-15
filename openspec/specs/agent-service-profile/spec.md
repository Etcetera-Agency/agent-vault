# agent-service-profile Specification

## Purpose
TBD - created by archiving change add-agent-service-profile. Update Purpose after archive.
## Requirements
### Requirement: Agent Method-Scoped Profile
WHEN an operator applies the agent service profile, the system SHALL create Gmail, Calendar, Discord, and Telegram services with narrow host, path, method, and credential policy.

#### Scenario: Profile Services Created
GIVEN vault `agent` has strict deny enabled
WHEN the operator applies the agent service profile
THEN services `gmail-list-read`, `calendar-events-read`, `discord-channel-messages`, and `telegram-bot-api` exist
AND each service has expected host/path and method policy.

#### Scenario: Profile Discoverable By Agent
GIVEN the agent service profile is applied
WHEN an agent calls `/discover`
THEN the response includes the four profile service names
AND each service includes its method allowlist
AND no credential values are returned.

#### Scenario: Profile Editable In UI
GIVEN the agent service profile is applied
WHEN an operator opens the service allowlist editor
THEN the operator can edit each profile service rule
AND saved changes preserve method policy, auth config, and substitutions.

### Requirement: Agent Read Method Protection
WHEN an agent uses Gmail or Calendar through the profile, the system SHALL allow read methods and deny write methods on protected read paths.

#### Scenario: Gmail Read Allowed Write Denied
GIVEN service `gmail-list-read` has `methods: ["GET"]`
WHEN an agent sends `GET` to the Gmail messages path
THEN Agent Vault forwards the request with `GOOGLE_ACCESS_TOKEN`
WHEN an agent sends `POST` to the same path
THEN Agent Vault returns HTTP 403.

#### Scenario: Calendar Read Allowed Write Denied
GIVEN service `calendar-events-read` has `methods: ["GET"]`
WHEN an agent sends `GET` to the Calendar events path
THEN Agent Vault forwards the request with `GOOGLE_ACCESS_TOKEN`
WHEN an agent sends `POST` to the same path
THEN Agent Vault returns HTTP 403.

### Requirement: Agent Send Service Protection
WHEN an agent uses Discord or Telegram through the profile, the system SHALL allow only the configured send paths and credential handling mode.

#### Scenario: Discord Channel Message Send
GIVEN service `discord-channel-messages` has `methods: ["POST"]`
WHEN an agent sends `POST` to `/api/v10/channels/{channel_id}/messages`
THEN Agent Vault forwards the request with `DISCORD_BOT_TOKEN`
AND sibling Discord API paths remain denied unless separately configured.

#### Scenario: Telegram Placeholder Send
GIVEN service `telegram-bot-api` has path placeholder `__bot_token__`
AND substitution key `TELEGRAM_BOT_TOKEN`
WHEN an agent sends `POST` to `/bot__bot_token__/sendMessage`
THEN Agent Vault substitutes the real Telegram bot token before forwarding
AND neither `/discover` nor request logs expose the token value.
