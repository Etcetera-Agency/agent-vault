# Spec Delta: mail-proxy

This file contains specification changes for `openspec/specs/mail-proxy/spec.md`.

## ADDED Requirements

### Requirement: Existing Service Mail Proxy Toggle Command

WHEN an operator updates mail proxy policy for an existing service allowlist record, the system SHALL mutate only that record's `mail_proxy` fields through an explicit CLI command while preserving all unrelated service fields.

#### Scenario: Toggle Existing Mail Proxy Protocol

GIVEN a vault has an existing enabled service named `gmail-mail`
AND that service has bearer auth token `GOOGLE_MAIL_OAUTH`
AND that service has `mail_proxy.email=agent@gmail.com`
AND that service has `mail_proxy.local_password_credential=HERMES_MAIL_LOCAL_PASSWORD`
AND that service has `mail_proxy.imap=true`
AND that service has `mail_proxy.smtp=true`
WHEN the operator runs `agent-vault vault service mail-proxy set gmail-mail --imap=false`
THEN the system stores the same service with `mail_proxy.imap=false`
AND keeps `mail_proxy.smtp=true`
AND keeps `mail_proxy.email=agent@gmail.com`
AND keeps `mail_proxy.local_password_credential=HERMES_MAIL_LOCAL_PASSWORD`
AND preserves the service host, auth, methods, enabled state, and substitutions.

#### Scenario: Create Mail Proxy Policy On Existing Service

GIVEN a vault has an existing enabled service named `gmail-mail`
AND that service has no `mail_proxy` block
WHEN the operator runs `agent-vault vault service mail-proxy set gmail-mail --imap=true --smtp=false --email agent@gmail.com --local-password-credential HERMES_MAIL_LOCAL_PASSWORD`
THEN the system stores a `mail_proxy` block on that service
AND sets `mail_proxy.imap=true`
AND sets `mail_proxy.smtp=false`
AND sets `mail_proxy.email=agent@gmail.com`
AND sets `mail_proxy.local_password_credential=HERMES_MAIL_LOCAL_PASSWORD`
AND preserves all unrelated service fields.

#### Scenario: Missing Toggle Target Rejected

GIVEN a vault has no service named `gmail-mail`
WHEN the operator runs `agent-vault vault service mail-proxy set gmail-mail --imap=true`
THEN the system fails without changing the service allowlist.

#### Scenario: No Mail Proxy Field Change Rejected

GIVEN a vault has an existing service named `gmail-mail`
WHEN the operator runs `agent-vault vault service mail-proxy set gmail-mail`
THEN the system rejects the command as a no-op
AND does not update the service allowlist.

## MODIFIED Requirements

### Requirement: Gmail SMTP XOAUTH2 Relay

WHEN Hermes authenticates to the local SMTP proxy, the system SHALL authenticate upstream to Gmail SMTP with XOAUTH2, verify upstream STARTTLS against the configured SMTP upstream host, and then relay SMTP bytes transparently.

#### Scenario: SMTP Auth Establishes Gmail Session

GIVEN Hermes has upgraded the local SMTP connection with STARTTLS
AND Hermes sends valid `AUTH PLAIN` or `AUTH LOGIN` credentials
WHEN the proxy obtains an OAuth access token
THEN the proxy connects to the configured SMTP upstream address
AND performs upstream STARTTLS
AND verifies the upstream certificate using the host part of the configured SMTP upstream address as the TLS server name
AND sends an XOAUTH2 authentication payload for the configured email address
AND returns local authentication success to Hermes after upstream authentication succeeds.

#### Scenario: Default SMTP Upstream Keeps Gmail Server Name

GIVEN the SMTP upstream address is omitted
WHEN the proxy authenticates upstream SMTP
THEN the system connects to `smtp.gmail.com:587`
AND verifies upstream STARTTLS with TLS server name `smtp.gmail.com`.

#### Scenario: SMTP Relay After Authentication

GIVEN local SMTP authentication and upstream Gmail XOAUTH2 authentication have succeeded
WHEN Hermes sends `MAIL FROM`, `RCPT TO`, `DATA`, or related SMTP commands
THEN the system relays raw bytes between Hermes and Gmail
AND Gmail remains responsible for sender validation, quotas, and delivery behavior.

### Requirement: Controlled Shutdown

WHEN the mail proxy receives a termination signal, the system SHALL stop accepting new sessions, allow active relays a bounded grace period, close active relay connections after the grace period, and release sensitive state.

#### Scenario: Graceful Shutdown Without Active Relay Timeout

GIVEN the mail proxy is running
AND active sessions exit before `--shutdown-timeout`
WHEN the process receives `SIGINT` or `SIGTERM`
THEN the system closes listeners
AND cancels in-progress connection setup
AND exits without forcing active connection close
AND wipes the in-memory master key before exit.

#### Scenario: Active Relay Closed After Shutdown Timeout

GIVEN the mail proxy is running
AND an IMAP or SMTP relay remains active after listeners are closed
WHEN the process receives `SIGINT` or `SIGTERM`
AND `--shutdown-timeout` elapses
THEN the system closes the active local relay connection
AND the relay goroutine exits
AND the process returns a shutdown timeout diagnostic.
