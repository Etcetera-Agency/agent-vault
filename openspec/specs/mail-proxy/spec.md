# mail-proxy Specification

## Purpose
Provide a local IMAP/SMTP proxy for Hermes Gmail usage while Agent Vault keeps Gmail OAuth and local proxy credentials encrypted.
## Requirements
### Requirement: Hermes Gmail Mail Proxy Command

WHEN an operator starts `agent-vault mail-proxy`, the system SHALL run a standalone local mail proxy using an existing Gmail service allowlist record so unmodified Hermes can use IMAP and SMTP while Agent Vault keeps Google OAuth tokens encrypted.

#### Scenario: Command Starts With Existing Gmail Service Record

- **GIVEN** the configured vault exists
- **AND** the configured service name resolves to an enabled existing Gmail service allowlist record in that vault
- **AND** the service record has `mail_proxy.imap` or `mail_proxy.smtp` enabled
- **AND** the service record references an existing Gmail OAuth credential through `auth.token`
- **AND** the OAuth credential is already connected and has a refresh token
- **AND** the service record has a non-empty `mail_proxy.email`
- **AND** the service record has a `mail_proxy.local_password_credential` that exists and is non-empty
- **WHEN** the operator starts `agent-vault mail-proxy`
- **THEN** the system opens only the local listeners enabled by the service record
- **AND** the system does not require `agent-vault server` to be running.

#### Scenario: Command Fails Before Listening

- **GIVEN** the configured vault, service record, OAuth credential, refresh token, local password credential, email address, protocol policy, or listener safety check is invalid
- **WHEN** the operator starts `agent-vault mail-proxy`
- **THEN** the system fails before opening any listener
- **AND** the system returns a diagnostic that identifies the missing configuration class without exposing secret values.

#### Scenario: Mail Proxy Does Not Create OAuth Credentials

- **GIVEN** the OAuth credential referenced by the selected service record does not exist or is not connected
- **WHEN** the operator starts `agent-vault mail-proxy`
- **THEN** the system fails before opening any listener
- **AND** the system does not create an OAuth credential
- **AND** the system does not start an OAuth browser consent flow.

### Requirement: Mail Proxy Protocol Policy On Service Allowlist

WHEN an operator manages an existing Gmail service allowlist record, the system SHALL allow that record to carry explicit mail proxy policy for IMAP and SMTP enablement.

#### Scenario: Service Record Enables IMAP And SMTP

- **GIVEN** a service allowlist record has `mail_proxy.imap=true`
- **AND** the same record has `mail_proxy.smtp=true`
- **WHEN** `agent-vault mail-proxy --service <name>` starts
- **THEN** the system opens both the local IMAP listener and the local SMTP listener.

#### Scenario: Service Record Enables IMAP Only

- **GIVEN** a service allowlist record has `mail_proxy.imap=true`
- **AND** the same record has `mail_proxy.smtp=false`
- **WHEN** `agent-vault mail-proxy --service <name>` starts
- **THEN** the system opens the local IMAP listener
- **AND** does not open the local SMTP listener.

#### Scenario: Service Record Enables SMTP Only

- **GIVEN** a service allowlist record has `mail_proxy.imap=false`
- **AND** the same record has `mail_proxy.smtp=true`
- **WHEN** `agent-vault mail-proxy --service <name>` starts
- **THEN** the system opens the local SMTP listener
- **AND** does not open the local IMAP listener.

#### Scenario: Service Record Disables Both Protocols

- **GIVEN** a service allowlist record has no enabled mail proxy protocol
- **WHEN** `agent-vault mail-proxy --service <name>` starts
- **THEN** the system fails before opening any listener.

#### Scenario: Disabled Service Record Cannot Start Mail Proxy

- **GIVEN** a service allowlist record has `enabled=false`
- **WHEN** `agent-vault mail-proxy --service <name>` starts
- **THEN** the system fails before opening any listener.

### Requirement: Existing Agent Vault Credential And Master Key Reuse

WHEN the mail proxy needs secrets, the system SHALL reuse existing Agent Vault store, master-key unlock, credential decryption, OAuth refresh, and CA primitives instead of introducing another credential database or key format.

#### Scenario: Existing Store And Master Key Used

- **GIVEN** Agent Vault has an initialized store and master-key record
- **WHEN** `agent-vault mail-proxy` starts
- **THEN** the system opens the same store implementation used by other Agent Vault commands
- **AND** unlocks the same master key
- **AND** wipes the master key on exit.

#### Scenario: No Schema Migration Required

- **GIVEN** the mail proxy configuration is supplied by flags or environment variables
- **WHEN** the operator enables the mail proxy
- **THEN** the system does not require a database migration
- **AND** stores only OAuth state and local password credentials in existing Agent Vault credential tables.

### Requirement: Shared OAuth Access Token Resolver

WHEN the mail proxy or HTTP broker needs an OAuth access token, the system SHALL resolve and refresh it through a shared Agent Vault OAuth credential resolver.

#### Scenario: Valid Token Returned Without Refresh

- **GIVEN** an OAuth credential has a decrypted access token that expires more than five minutes in the future
- **WHEN** the resolver is asked for an access token
- **THEN** the resolver returns the current access token
- **AND** does not call the token endpoint.

#### Scenario: Expiring Token Refreshed

- **GIVEN** an OAuth credential has a refresh token and an access token that expires within five minutes
- **WHEN** the resolver is asked for an access token
- **THEN** the resolver calls the existing OAuth refresh primitive
- **AND** encrypts and persists the new access token
- **AND** persists a rotated refresh token when the provider returns one
- **AND** returns the new access token.

#### Scenario: Forced Refresh Retry

- **GIVEN** Gmail rejects XOAUTH2 authentication for an access token
- **WHEN** the mail proxy requests a forced refresh
- **THEN** the resolver refreshes once even if the stored expiry has not passed
- **AND** the mail proxy retries upstream authentication at most one time.

### Requirement: Agent Vault CA Local TLS

WHEN Hermes connects to local IMAP or SMTP listeners, the system SHALL secure local protocol negotiation with a certificate minted by the existing Agent Vault CA.

#### Scenario: IMAP Uses Implicit TLS

- **GIVEN** Hermes is configured with `EMAIL_IMAP_HOST=127.0.0.1` and `EMAIL_IMAP_PORT=1993`
- **WHEN** Hermes connects using `imaplib.IMAP4_SSL`
- **THEN** the mail proxy completes a TLS handshake using a leaf certificate minted by the Agent Vault CA
- **AND** the certificate covers `127.0.0.1`.

#### Scenario: SMTP Uses STARTTLS

- **GIVEN** Hermes is configured with `EMAIL_SMTP_HOST=127.0.0.1` and `EMAIL_SMTP_PORT=1587`
- **WHEN** Hermes sends `STARTTLS`
- **THEN** the mail proxy upgrades the local SMTP connection to TLS using a leaf certificate minted by the Agent Vault CA
- **AND** continues SMTP authentication only after the TLS upgrade.

#### Scenario: Hermes Trusts Agent Vault CA

- **GIVEN** the Hermes process trusts the Agent Vault root CA PEM through `SSL_CERT_FILE` or system trust
- **WHEN** Hermes connects to the local IMAP and SMTP listeners
- **THEN** TLS verification succeeds without disabling certificate verification.

### Requirement: Loopback-Only Local Listeners

WHEN the mail proxy opens local listeners, the system SHALL bind to loopback addresses only in the MVP.

#### Scenario: Default Listeners Are Loopback

- **WHEN** the operator omits listener flags
- **THEN** the system listens on `127.0.0.1:1993` for IMAP
- **AND** listens on `127.0.0.1:1587` for SMTP.

#### Scenario: Non-Loopback Listener Rejected

- **GIVEN** the operator configures a listener address that is not loopback
- **WHEN** the operator starts `agent-vault mail-proxy`
- **THEN** the system rejects the configuration
- **AND** does not open any listener.

### Requirement: Local Password Authentication

WHEN Hermes authenticates to the local mail proxy, the system SHALL verify a separate local password stored as an Agent Vault credential.

#### Scenario: Valid Local Password Accepted

- **GIVEN** the service record's local password credential decrypts to a non-empty value
- **WHEN** Hermes authenticates with the configured email address and that local password
- **THEN** the system accepts local authentication
- **AND** proceeds to upstream Gmail OAuth authentication.

#### Scenario: Invalid Local Password Rejected

- **GIVEN** Hermes authenticates with an incorrect username or password
- **WHEN** the local proxy validates the credentials
- **THEN** the system compares the password in constant time
- **AND** rejects authentication with a generic failure
- **AND** does not reveal whether the username or password was wrong.

### Requirement: Gmail IMAP XOAUTH2 Relay

WHEN Hermes authenticates to the local IMAP proxy, the system SHALL authenticate upstream to Gmail IMAP with XOAUTH2 and then relay IMAP bytes transparently.

#### Scenario: IMAP Login Establishes Gmail Session

- **GIVEN** Hermes sends a tagged IMAP `LOGIN` command with valid local credentials
- **WHEN** the proxy obtains an OAuth access token
- **THEN** the proxy connects to Gmail IMAP over TLS
- **AND** sends an XOAUTH2 authentication payload for the configured email address
- **AND** returns a tagged `OK` response to Hermes after upstream authentication succeeds.

#### Scenario: IMAP Relay After Authentication

- **GIVEN** local IMAP authentication and upstream Gmail XOAUTH2 authentication have succeeded
- **WHEN** Hermes sends post-authentication IMAP commands
- **THEN** the system relays raw bytes between Hermes and Gmail
- **AND** does not implement mailbox semantics itself.

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
