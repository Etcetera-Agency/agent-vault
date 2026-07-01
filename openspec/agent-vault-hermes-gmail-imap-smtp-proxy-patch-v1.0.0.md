# Agent Vault Mail Proxy Patch for Hermes

**Version:** 1.0.0
**Target:** `Infisical/agent-vault`
**Purpose:** allow the unmodified Hermes Email Gateway to use Gmail IMAP/SMTP through Google OAuth 2.0 without exposing Google OAuth tokens or an app password to Hermes.

## 1. Goal

Hermes currently expects a normal IMAP/SMTP mailbox:

```env
EMAIL_ADDRESS=agent@gmail.com
EMAIL_PASSWORD=<password>
EMAIL_IMAP_HOST=<host>
EMAIL_IMAP_PORT=<port>
EMAIL_SMTP_HOST=<host>
EMAIL_SMTP_PORT=<port>
```

The patch adds a local mail proxy to Agent Vault:

```text
Hermes
  ├─ IMAP LOGIN → 127.0.0.1:1993
  └─ SMTP AUTH  → 127.0.0.1:1587
                       │
                       ▼
             Agent Vault mail-proxy
                       │
               Gmail OAuth XOAUTH2
                       │
          imap.gmail.com / smtp.gmail.com
```

Hermes continues using its existing IMAP/SMTP adapter and does not need to be patched.

## 2. Main design decision

Implement the proxy as a **separate Agent Vault command**:

```bash
agent-vault mail-proxy
```

Do **not** attach it to `agent-vault server`.

This keeps the patch isolated from the most frequently changed upstream areas:

- `internal/server/server.go`
- `cmd/server.go`
- HTTP router and handlers
- MITM proxy lifecycle
- web UI
- database migrations
- existing broker configuration

The new command opens the same Agent Vault store, decrypts the configured OAuth credential using the existing master-key flow, refreshes it using existing OAuth code, and exposes local IMAP/SMTP listeners.

## 3. Upstream compatibility objective

The implementation should require only one small edit to an existing upstream file:

```text
cmd/root.go or the current command-registration file
```

The edit only registers:

```go
rootCmd.AddCommand(mailProxyCmd)
```

Everything else must be added as new files.

If command registration is already split into `init()` functions, even this edit is unnecessary: `cmd/mail_proxy.go` can register itself from `init()`.

## 4. Files to add

```text
cmd/
  mail_proxy.go

internal/mailproxy/
  config.go
  proxy.go
  token_provider.go
  imap.go
  smtp.go
  relay.go
  xoauth2.go
  tls.go
  errors.go

internal/mailproxy/
  config_test.go
  token_provider_test.go
  imap_test.go
  smtp_test.go
  xoauth2_test.go
  integration_test.go

docs/guides/
  hermes-gmail-mail-proxy.mdx
```

No existing database schema should be changed.

## 5. Reuse existing Agent Vault components

The patch should reuse existing code rather than duplicate OAuth behavior.

Relevant existing components:

```text
internal/oauth/oauth.go
internal/oauth/refresher.go
internal/store/*
internal/crypto/*
```

Agent Vault already implements:

- OAuth authorization-code exchange;
- encrypted storage of access and refresh tokens;
- token expiry tracking;
- refresh-token grant;
- refresh-token rotation;
- refresh deduplication through `singleflight`;
- a five-minute refresh buffer.

The mail proxy must call the same OAuth refresh primitives.

Do not copy OAuth refresh code into `internal/mailproxy`.

## 6. Command interface

```bash
agent-vault mail-proxy [flags]
```

Recommended flags:

```text
--vault string
--oauth-credential string

--email string

--imap-listen string       default 127.0.0.1:1993
--imap-upstream string     default imap.gmail.com:993

--smtp-listen string       default 127.0.0.1:1587
--smtp-upstream string     default smtp.gmail.com:587
--smtp-upstream-tls string default starttls

--local-password-credential string
--password-stdin

--database-url string
--log-level string         default info

--disable-imap
--disable-smtp
```

Environment equivalents:

```env
AGENT_VAULT_MAIL_VAULT=hermes
AGENT_VAULT_MAIL_OAUTH_CREDENTIAL=GOOGLE_MAIL_OAUTH
AGENT_VAULT_MAIL_EMAIL=agent@gmail.com

AGENT_VAULT_MAIL_IMAP_LISTEN=127.0.0.1:1993
AGENT_VAULT_MAIL_IMAP_UPSTREAM=imap.gmail.com:993

AGENT_VAULT_MAIL_SMTP_LISTEN=127.0.0.1:1587
AGENT_VAULT_MAIL_SMTP_UPSTREAM=smtp.gmail.com:587
AGENT_VAULT_MAIL_SMTP_UPSTREAM_TLS=starttls

AGENT_VAULT_MAIL_LOCAL_PASSWORD_CREDENTIAL=HERMES_MAIL_LOCAL_PASSWORD
```

Flag values override environment variables.

## 7. Local authentication

Hermes must authenticate to the local proxy with a separate random password stored in Agent Vault:

```text
HERMES_MAIL_LOCAL_PASSWORD
```

Hermes configuration:

```env
EMAIL_ADDRESS=agent@gmail.com
EMAIL_PASSWORD=<local proxy password>

EMAIL_IMAP_HOST=127.0.0.1
EMAIL_IMAP_PORT=1993

EMAIL_SMTP_HOST=127.0.0.1
EMAIL_SMTP_PORT=1587
```

This password is **not** a Gmail password and has no value outside the local proxy.

Requirements:

- compare passwords in constant time;
- never log the username or password together;
- never accept an empty password;
- bind to loopback by default;
- reject non-loopback listeners unless `--allow-remote-listen` is explicitly supplied;
- when remote listening is enabled, require TLS for the local side.

## 8. OAuth scopes

For Gmail IMAP and SMTP through XOAUTH2, the OAuth connection must include:

```text
https://mail.google.com/
```

For Calendar API through the existing HTTP broker, the same OAuth connection may additionally include:

```text
https://www.googleapis.com/auth/calendar
```

Recommended combined scope set:

```text
openid
email
https://mail.google.com/
https://www.googleapis.com/auth/calendar
```

The proxy must validate that the stored OAuth credential is connected and has a refresh token.

It should not attempt to infer whether Google granted every requested scope. Authentication failure from Gmail must produce a clear diagnostic suggesting reconnection with `https://mail.google.com/`.

## 9. Token provider abstraction

Create a narrow interface:

```go
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}
```

Production implementation:

```go
type VaultOAuthTokenProvider struct {
    Store         OAuthStore
    VaultID       string
    CredentialKey string
    EncryptionKey []byte
    Refresher     *oauth.Refresher
}
```

Responsibilities:

1. load the OAuth credential;
2. decrypt the current access token;
3. inspect expiry;
4. return it when valid for more than five minutes;
5. otherwise decrypt the refresh token;
6. call `oauth.Refresh`;
7. encrypt and persist rotated tokens;
8. return the new access token.

Do not expose a generic HTTP endpoint that returns OAuth tokens.

The token remains inside the `agent-vault mail-proxy` process.

## 10. Avoid coupling to `brokercore.StoreCredentialProvider`

Do not call:

```go
StoreCredentialProvider.Inject(...)
```

That API is designed around HTTP host/path matching and header injection.

Instead, extract or add a small reusable OAuth-token resolver in a new file:

```text
internal/oauthcredential/resolver.go
```

Suggested API:

```go
type Store interface {
    GetCredential(ctx context.Context, vaultID, key string) (*store.Credential, error)
    GetCredentialOAuth(ctx context.Context, vaultID, key string) (*store.CredentialOAuth, error)
    UpdateCredentialOAuthTokens(
        ctx context.Context,
        vaultID, key string,
        accessCT, accessNonce,
        refreshCT, refreshNonce []byte,
        expiresAt *time.Time,
    ) error
    UpdateCredentialOAuthError(
        ctx context.Context,
        vaultID, key, message string,
    ) error
}

type Resolver struct {
    Store     Store
    EncKey    []byte
    Refresher *oauth.Refresher
}

func (r *Resolver) Resolve(
    ctx context.Context,
    vaultID string,
    credentialKey string,
) (string, error)
```

Then make the existing HTTP broker use this resolver too.

However, to minimize the first patch, the MVP may keep a private resolver inside `internal/mailproxy/token_provider.go`. A later upstream-friendly refactor can share it with `brokercore`.

## 11. IMAP proxy behavior

### 11.1 Local side

Listen on:

```text
127.0.0.1:1993
```

The local listener may be plaintext because it is loopback-only.

Support the minimum pre-authentication commands required by Python `imaplib`:

```text
CAPABILITY
NOOP
LOGIN
LOGOUT
```

Advertise:

```text
IMAP4rev1
AUTH=PLAIN
```

Do not advertise XOAUTH2 to Hermes.

### 11.2 Upstream side

Connect using implicit TLS:

```text
imap.gmail.com:993
```

Perform:

```text
AUTHENTICATE XOAUTH2
```

XOAUTH2 initial client response:

```text
user=<email>\x01auth=Bearer <access_token>\x01\x01
```

Base64-encode the complete byte sequence.

### 11.3 Relay transition

After successful local `LOGIN` and successful upstream XOAUTH2:

1. return a tagged `OK` response to Hermes;
2. switch to transparent bidirectional relay;
3. preserve raw bytes;
4. do not parse post-authentication IMAP commands;
5. close both sides when either side closes.

This avoids implementing IMAP mailbox semantics.

### 11.4 IMAP command tags

Before authentication, preserve the client's IMAP tag:

```text
A001 LOGIN user password
```

Responses must use the same tag:

```text
A001 OK LOGIN completed
```

Reject malformed commands with:

```text
<tag> BAD malformed command
```

Reject invalid local credentials with:

```text
<tag> NO authentication failed
```

Do not reveal whether the username or password was incorrect.

## 12. SMTP proxy behavior

### 12.1 Local side

Listen on:

```text
127.0.0.1:1587
```

Support enough SMTP for Python `smtplib`:

```text
EHLO
HELO
NOOP
RSET
QUIT
STARTTLS
AUTH LOGIN
AUTH PLAIN
```

For loopback-only MVP, local STARTTLS may be optional. Hermes can connect without local TLS.

Advertise:

```text
AUTH LOGIN PLAIN
8BITMIME
SMTPUTF8
SIZE
```

Do not advertise XOAUTH2 locally.

### 12.2 Upstream side

Connect to:

```text
smtp.gmail.com:587
```

Flow:

```text
EHLO
STARTTLS
EHLO
AUTH XOAUTH2 <base64 payload>
```

Then return successful local authentication to Hermes.

### 12.3 Relay transition

After authentication, relay raw SMTP traffic bidirectionally.

The proxy should not parse:

```text
MAIL FROM
RCPT TO
DATA
```

Gmail remains responsible for sender validation, message limits and delivery.

## 13. Token refresh and retry behavior

For each new upstream connection:

1. request a valid access token;
2. authenticate with Gmail;
3. if Gmail rejects XOAUTH2 with an authentication-specific error:
   - invalidate the cached access token in memory;
   - force exactly one refresh;
   - reconnect once;
4. if the second attempt fails, return an authentication error locally;
5. never retry indefinitely.

Do not refresh per IMAP command or per SMTP message. Refresh only while establishing an authenticated upstream connection.

Existing authenticated IMAP sessions may remain connected after the token expires; OAuth tokens are only needed during authentication.

## 14. Process lifecycle

The `mail-proxy` command runs independently from `agent-vault server`.

It must:

- listen for `SIGINT` and `SIGTERM`;
- stop accepting new sessions;
- cancel in-progress connection setup;
- allow active relays a configurable grace period;
- then close remaining connections;
- wipe the in-memory encryption key before exit.

Suggested flag:

```text
--shutdown-timeout 10s
```

The command must fail before opening listeners when:

- the vault does not exist;
- OAuth credential does not exist;
- OAuth credential is not connected;
- local-password credential does not exist;
- email is empty;
- listener address is unsafe;
- both IMAP and SMTP are disabled.

## 15. Master key and database access

Reuse the existing Agent Vault unlock and store-opening helpers wherever possible.

Preferred implementation:

```go
func runMailProxy(cmd *cobra.Command, args []string) error {
    db := openStoreUsingExistingHelpers(...)
    masterKey := unlockUsingExistingHelpers(...)
    defer masterKey.Wipe()

    cfg := loadMailProxyConfig(...)
    proxy := mailproxy.New(...)

    return proxy.Run(cmd.Context())
}
```

Do not introduce another master-key format.

Do not create a second token database.

The proxy may run concurrently with `agent-vault server` against the same store.

SQLite requirements:

- use the existing store opener;
- rely on the existing WAL/busy-timeout configuration;
- keep OAuth refresh writes short;
- do not hold a DB transaction while connecting to Google.

PostgreSQL uses the same existing store implementation.

## 16. Configuration storage

For the first version, keep mail-proxy configuration outside the database and supply it through flags/environment variables.

Reasons:

- no migration;
- no web UI changes;
- no new CRUD handlers;
- smaller upstream conflict surface;
- easier rollback.

Only secrets and OAuth state remain in Agent Vault.

## 17. No changes required in Hermes

Hermes retains its normal configuration:

```env
EMAIL_ADDRESS=agent@gmail.com
EMAIL_PASSWORD=<value of HERMES_MAIL_LOCAL_PASSWORD>

EMAIL_IMAP_HOST=127.0.0.1
EMAIL_IMAP_PORT=1993

EMAIL_SMTP_HOST=127.0.0.1
EMAIL_SMTP_PORT=1587

EMAIL_POLL_INTERVAL=15
EMAIL_ALLOWED_USERS=denys@example.com
```

Hermes still believes it is using password-based IMAP/SMTP.

## 18. Calendar access

The mail proxy only handles Gmail IMAP/SMTP.

Google Calendar remains an HTTPS API and should use the existing Agent Vault HTTP credential broker:

```text
Hermes calendar tool
    → Agent Vault HTTP/MITM broker
    → Google Calendar API
```

The same Google OAuth credential may be reused when its scope set includes both Gmail and Calendar permissions.

## 19. Security boundaries

The design must ensure:

- Google refresh token never leaves Agent Vault storage/processes;
- Google access token never reaches Hermes;
- Hermes receives only the local proxy password;
- proxy binds to loopback by default;
- secrets are redacted from logs;
- raw email content is not logged;
- XOAUTH2 payload is never logged;
- local authentication failures are rate-limited per source address;
- upstream TLS verification is always enabled;
- Gmail hostnames are fixed by default;
- arbitrary upstream hosts require an explicit unsafe/development flag.

## 20. Logging

Allowed fields:

```text
component=mailproxy
protocol=imap|smtp
listener=127.0.0.1:1993
upstream=imap.gmail.com:993
event=accepted|authenticated|relay_started|relay_closed|error
duration_ms=...
bytes_up=...
bytes_down=...
```

Forbidden fields:

```text
password
access_token
refresh_token
Authorization header
XOAUTH2 payload
email body
SMTP DATA
IMAP FETCH response
```

The configured mailbox address should be masked or omitted at info level.

## 21. Health checks

Add local health output through either:

```bash
agent-vault mail-proxy health
```

or a loopback-only HTTP listener:

```text
127.0.0.1:19090/healthz
```

Recommended response:

```json
{
  "status": "ok",
  "imap_listener": true,
  "smtp_listener": true,
  "oauth_connected": true,
  "last_refresh_error": null
}
```

The health check must not force token revelation.

A startup token refresh may be performed to fail early when the OAuth connection is broken.

## 22. Testing requirements

### Unit tests

Test:

- config validation;
- loopback enforcement;
- constant-time password validation;
- XOAUTH2 payload construction;
- IMAP tagged responses;
- SMTP multiline EHLO responses;
- AUTH LOGIN state machine;
- AUTH PLAIN parsing;
- refresh before expiry;
- forced single retry after Gmail auth rejection;
- no secret values in errors/logs.

### Protocol tests

Use fake upstream servers:

```text
fake Gmail IMAP server
fake Gmail SMTP server
fake OAuth token endpoint
```

Verify:

- Hermes-style IMAP login;
- Python `imaplib.IMAP4` compatibility;
- Python `smtplib.SMTP` compatibility;
- transparent post-authentication relay;
- token rotation;
- connection shutdown;
- malformed client input;
- upstream timeout;
- upstream TLS failure.

### Integration test

Run the actual Hermes email adapter against the local proxy using fake upstream services.

The test should prove that Hermes code requires no changes.

## 23. Update-resistance rules

The patch must follow these rules:

1. Add new files instead of editing existing files.
2. Do not modify `internal/server.Server`.
3. Do not modify HTTP routes.
4. Do not modify existing migrations.
5. Do not modify the web UI.
6. Do not alter `brokercore.CredentialProvider`.
7. Reuse public or stable store methods already required by OAuth.
8. Keep all protocol code under `internal/mailproxy`.
9. Register the command through a self-contained `init()` when possible.
10. Keep the patch as a small commit series that can be rebased independently.

Recommended commit split:

```text
1. add mailproxy config and token provider
2. add IMAP proxy
3. add SMTP proxy
4. add mail-proxy CLI command
5. add tests and documentation
```

## 24. Rebase strategy

Maintain the patch branch:

```text
upstream/main
custom/mail-proxy
```

Update procedure:

```bash
git fetch upstream
git checkout custom/mail-proxy
git rebase upstream/main
go test ./internal/mailproxy/... ./cmd/...
go test ./...
```

Expected conflict area:

```text
cmd command registration only
```

All protocol implementation should remain in new files and normally rebase without conflicts.

## 25. systemd example

Agent Vault server:

```ini
[Unit]
Description=Agent Vault
After=network-online.target

[Service]
User=agentvault
ExecStart=/usr/local/bin/agent-vault server --host 127.0.0.1
Restart=on-failure
EnvironmentFile=/etc/agent-vault/server.env

[Install]
WantedBy=multi-user.target
```

Mail proxy:

```ini
[Unit]
Description=Agent Vault Gmail Mail Proxy
After=network-online.target agent-vault.service
Requires=agent-vault.service

[Service]
User=agentvault
ExecStart=/usr/local/bin/agent-vault mail-proxy \
  --vault hermes \
  --oauth-credential GOOGLE_MAIL_OAUTH \
  --email agent@gmail.com \
  --local-password-credential HERMES_MAIL_LOCAL_PASSWORD
Restart=on-failure
RestartSec=3
EnvironmentFile=/etc/agent-vault/mail-proxy.env

[Install]
WantedBy=multi-user.target
```

If both commands require an interactive master password, use the existing supported Agent Vault password-delivery mechanism rather than placing the master password directly in the unit file.

## 26. Acceptance criteria

The patch is complete when:

- stock Hermes connects without code changes;
- Hermes reads Gmail through IMAP;
- Hermes can send mail through SMTP;
- Google authentication uses OAuth XOAUTH2;
- Agent Vault refreshes expired access tokens;
- Hermes never receives Google OAuth tokens;
- Agent Vault server continues working when mail proxy is disabled or stopped;
- no database migration is introduced;
- rebasing onto a newer upstream version normally touches only command registration;
- `go test ./...` passes;
- integration tests prove compatibility with `imaplib` and `smtplib`.

## 27. Explicit non-goals

Version 1 does not include:

- generic IMAP/SMTP proxying for arbitrary providers;
- mailbox filtering;
- email parsing;
- Calendar protocol proxying;
- Gmail API replacement for IMAP;
- web UI configuration;
- multi-mailbox routing on one listener;
- remote/public listeners by default;
- message caching or persistence.

## 28. Recommended implementation order

1. Implement `VaultOAuthTokenProvider`.
2. Test refresh and token rotation against a fake OAuth endpoint.
3. Implement IMAP local authentication and upstream XOAUTH2.
4. Add transparent relay and IMAP tests.
5. Implement SMTP AUTH LOGIN/PLAIN and upstream XOAUTH2.
6. Add the standalone CLI command.
7. Test with Python `imaplib` and `smtplib`.
8. Point stock Hermes at the local listeners.
9. Add Calendar scopes to the same Google OAuth connection if needed.
