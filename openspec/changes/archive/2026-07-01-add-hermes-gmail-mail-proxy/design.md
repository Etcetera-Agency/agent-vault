# Design: Hermes Gmail mail proxy

## Context

Hermes Email Gateway uses Python stdlib `imaplib.IMAP4_SSL` for inbound polling and `smtplib.SMTP` with `STARTTLS` for outbound replies. It polls `UNSEEN` messages at `EMAIL_POLL_INTERVAL`; it does not listen for inbound mail. Therefore Agent Vault must present TLS-capable local IMAP/SMTP endpoints to Hermes.

Agent Vault already has the needed primitives:

- encrypted credential rows in `internal/store`;
- master-key unlock in `cmd/server.go`;
- OAuth refresh protocol in `internal/oauth`;
- broker OAuth refresh logic in `internal/brokercore`;
- encrypted CA root and leaf minting in `internal/ca`.

## Goals

- Keep Hermes unmodified.
- Keep Google OAuth tokens inside Agent Vault.
- Use an existing Gmail service allowlist record from Agent Vault.
- Use one local random password for Hermes-to-proxy auth.
- Use Agent Vault CA for local TLS.
- Prefer new files and tiny additive fork-local edits.
- Keep first implementation loopback-only.

## Non-Goals

- No remote listener support in first slice.
- No local plaintext mode.
- No new database schema.
- No standalone web UI configuration page outside the existing service allowlist editor.
- No OAuth credential creation or reconnection flow in the mail proxy.
- No Gmail API rewrite.
- No custom mailbox semantics in Agent Vault.

## Decisions

### Existing Gmail service allowlist record only

The operator passes `--service <name>` or `AGENT_VAULT_MAIL_SERVICE=<name>`. The proxy resolves that service in the configured vault's existing allowlist.

The selected service record carries the mail proxy policy:

```yaml
name: gmail-mail
host: gmail.googleapis.com/gmail/v1/users/me/messages*
methods: ["GET"]
auth:
  type: bearer
  token: GOOGLE_MAIL_OAUTH
mail_proxy:
  email: agent@gmail.com
  local_password_credential: HERMES_MAIL_LOCAL_PASSWORD
  imap: true
  smtp: true
```

The proxy requires:

- `mail_proxy.imap` or `mail_proxy.smtp` is true;
- `mail_proxy.email` is non-empty;
- `mail_proxy.local_password_credential` names an existing non-empty static credential;
- `auth.type` is `bearer`;
- `auth.token` names an existing connected OAuth credential with a refresh token.

The proxy does not create OAuth credentials, edit scopes, or launch browser consent. If Gmail rejects XOAUTH2, the diagnostic tells the operator to reconnect the OAuth credential referenced by `auth.token` with `https://mail.google.com/`.

### Separate command

Add `agent-vault mail-proxy` as an independent command. Register it from `cmd/mail_proxy.go` using `init()` so `cmd/root.go` does not need edits.

### Existing CA for local TLS

Use `internal/ca.New(masterKey.Key(), ca.Options{...})` and `MintLeaf("127.0.0.1")` for local IMAP implicit TLS and SMTP STARTTLS.

Hermes must trust the Agent Vault root PEM through `SSL_CERT_FILE`, system trust, or the same CA file used for Agent Vault MITM. The proxy should print or document the root PEM path but must not expose private CA key material.

### Shared OAuth resolver

Add `internal/oauthcredential/resolver.go` with the narrow token-resolution contract needed by both HTTP broker and mail proxy. The resolver reads encrypted OAuth credentials, decrypts tokens with the Agent Vault master key, refreshes when expiry is within the existing five-minute buffer, persists rotated tokens, and can force one refresh after an upstream XOAUTH2 rejection.

Keep the broker edit minimal: replace private refresh logic with resolver use while preserving existing behavior.

### Loopback-only listeners

Default listeners:

```text
127.0.0.1:1993  local IMAP implicit TLS
127.0.0.1:1587  local SMTP with STARTTLS
```

Reject non-loopback listener addresses in the MVP. Remote listening needs explicit TLS/cert/trust UX and is deferred.

The service `enabled` field remains the kill switch for the mail proxy record. If the selected service is disabled, `agent-vault mail-proxy` fails before listening.

### Protocol handling

IMAP local side supports only pre-auth commands needed by Hermes:

```text
CAPABILITY
NOOP
LOGIN
LOGOUT
```

After successful local `LOGIN`, proxy connects to Gmail IMAP over TLS, authenticates with XOAUTH2, returns tagged `OK`, then relays raw bytes.

SMTP local side supports only auth/setup commands needed by Hermes:

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

After successful local auth, proxy connects to Gmail SMTP, performs `EHLO`, `STARTTLS`, `EHLO`, `AUTH XOAUTH2`, returns local auth success, then relays raw bytes.

## Pseudocode

```go
func runMailProxy(cmd *cobra.Command, args []string) error {
    cfg := loadMailProxyConfig(cmd.Flags(), os.Environ())
    if err := cfg.ValidateLoopbackOnly(); err != nil { return err }

    db := openStoreLikeServer(cfg.DatabaseURL)
    defer db.Close()

    masterKey := unlockOrSetup(cmd, db, cfg.PasswordStdin)
    defer masterKey.Wipe()

    vault := db.GetVault(ctx, cfg.VaultName)
    service := loadEnabledMailProxyService(db, vault.ID, cfg.ServiceName)
    assertExistingConnectedOAuthCredential(db, vault.ID, service.Auth.Token)
    localPassword := decryptStaticCredential(db, vault.ID, service.MailProxy.LocalPasswordCredential, masterKey.Key())

    caProvider := ca.New(masterKey.Key(), ca.Options{Store: optionalPostgresCAStore(db)})
    localCert := caProvider.MintLeaf("127.0.0.1")

    tokenResolver := oauthcredential.Resolver{
        Store: db,
        EncKey: masterKey.Key(),
        Refresher: oauth.NewRefresher(),
    }

    proxy := mailproxy.New(mailproxy.Options{
        Config: cfg,
        LocalPassword: localPassword,
        LocalTLSCert: localCert,
        TokenProvider: mailproxy.OAuthTokenProvider{
            Resolver: tokenResolver,
            VaultID: vault.ID,
            CredentialKey: service.Auth.Token,
        },
    })

    return proxy.Run(signalContext)
}
```

```go
func handleIMAP(conn net.Conn) {
    tlsConn := tls.Server(conn, localTLSConfig)
    read pre-auth commands
    on LOGIN:
        verify local password constant-time
        token := tokenProvider.AccessToken(ctx)
        upstream := tls.Dial("tcp", imap.gmail.com:993)
        authenticateXOAUTH2(upstream, email, token)
        if auth rejected:
            token = tokenProvider.ForceRefresh(ctx)
            reconnect once and retry XOAUTH2
        send "<tag> OK LOGIN completed"
        relayBothDirections(tlsConn, upstream)
}
```

```go
func handleSMTP(conn net.Conn) {
    read EHLO/HELO
    advertise STARTTLS and AUTH LOGIN PLAIN
    on STARTTLS:
        wrap local connection with tls.Server(localTLSConfig)
    on AUTH:
        verify local password constant-time
        token := tokenProvider.AccessToken(ctx)
        upstream := dial smtp.gmail.com:587
        upstream STARTTLS
        authenticateXOAUTH2(upstream, email, token)
        if auth rejected:
            token = tokenProvider.ForceRefresh(ctx)
            reconnect once and retry XOAUTH2
        return local auth success
        relayBothDirections(localConn, upstream)
}
```

## Fork Conflict Surface

- New files for command, OAuth resolver, mail proxy package, tests, and docs.
- A small optional `mail_proxy` field appended to the existing service JSON shape.
- Minimal service allowlist management updates to preserve and expose the `mail_proxy` IMAP/SMTP toggles.
- One minimal fork-local edit in broker credential code to call the shared resolver.
- No command root edit if `cmd/mail_proxy.go` registers itself in `init()`.
- No store schema edit.
- Tests live in new `*_test.go` files.

## Security Notes

- Constant-time local password comparison.
- Empty local password rejected.
- No username/password/token logged.
- Non-loopback listeners rejected.
- Local TLS required because current Hermes uses IMAP SSL and SMTP STARTTLS.
- OAuth token never returned over HTTP or printed.
- One forced refresh retry only after upstream auth failure.
