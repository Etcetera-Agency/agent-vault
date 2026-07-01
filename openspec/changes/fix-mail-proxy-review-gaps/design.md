# Design: Mail proxy review gap fixes

## Goals

- Keep implementation simple.
- Reuse current Agent Vault service API and mail proxy structures.
- Do not add Web UI mail proxy configuration.
- Do not add database schema or new persistence shape.
- Preserve existing service record fields when toggling mail proxy policy.

## Slice 1: Controlled Shutdown Closes Active Sessions

`Proxy.Run` should continue to own lifecycle. Protocol handlers should stay protocol-focused.

Add a tiny active connection registry to `Proxy`:

```go
type Proxy struct {
    // existing fields
    mu     sync.Mutex
    active map[net.Conn]struct{}
}

func (p *Proxy) track(conn net.Conn) func() {
    p.mu.Lock()
    if p.active == nil {
        p.active = map[net.Conn]struct{}{}
    }
    p.active[conn] = struct{}{}
    p.mu.Unlock()

    return func() {
        p.mu.Lock()
        delete(p.active, conn)
        p.mu.Unlock()
    }
}

func (p *Proxy) closeActive() {
    p.mu.Lock()
    conns := make([]net.Conn, 0, len(p.active))
    for conn := range p.active {
        conns = append(conns, conn)
    }
    p.mu.Unlock()

    for _, conn := range conns {
        _ = conn.Close()
    }
}
```

Accept loops call `done := p.track(conn)` before starting a session and `defer done()` inside the session goroutine.

Shutdown flow:

1. Context is canceled or listener error occurs.
2. Close listeners.
3. Wait for session goroutines until `ShutdownTimeout`.
4. If timeout fires, close active connections.
5. Wait once more for goroutines to exit or return shutdown timeout error.

This keeps `Relay(left, right)` simple and unchanged.

## Slice 2: SMTP Upstream TLS Server Name

SMTP upstream address is already configurable. TLS verification must match the configured address host.

Add a helper:

```go
func serverNameFromAddress(addr string) (string, error) {
    host, _, err := net.SplitHostPort(addr)
    if err != nil {
        return "", err
    }
    return host, nil
}
```

Extend `SMTPOptions`:

```go
type SMTPOptions struct {
    // existing fields
    UpstreamServerName string
}
```

`Proxy.acceptSMTP` sets `UpstreamServerName` from `Config.SMTPUpstream`.

`AuthenticateSMTPXOAUTH2` receives the server name through options or a small wrapper. It builds:

```go
&tls.Config{
    MinVersion: tls.VersionTLS12,
    ServerName: upstreamServerName,
}
```

Fallback remains `smtp.gmail.com` only when no configured name is present, so current tests and default behavior stay stable.

## Slice 3: CLI Toggle Existing Mail Proxy Policy

No Web UI config. The CLI updates existing service records through the existing service API.

Command:

```text
agent-vault vault service mail-proxy set <service> \
  --imap=<bool> \
  --smtp=<bool> \
  [--email <addr>] \
  [--local-password-credential <key>]
```

Behavior:

1. Resolve vault using existing `resolveVault`.
2. Use existing admin session helpers.
3. GET `/v1/vaults/<vault>/services`.
4. Find one service by exact `name`.
5. Copy the service.
6. Create `MailProxy` if missing.
7. Update only changed mail proxy fields.
8. PUT the full service list back to `/v1/vaults/<vault>/services`.

Pseudocode:

```go
func runServiceMailProxySet(cmd, serviceName) error {
    services := getServices(vault)
    idx := findServiceByName(services, serviceName)
    if idx < 0 {
        return fmt.Errorf("service %q not found", serviceName)
    }

    svc := services[idx]
    if svc.MailProxy == nil {
        svc.MailProxy = &broker.MailProxyPolicy{}
    }
    if flagChanged("imap") {
        svc.MailProxy.IMAP = imap
    }
    if flagChanged("smtp") {
        svc.MailProxy.SMTP = smtp
    }
    if flagChanged("email") {
        svc.MailProxy.Email = strings.TrimSpace(email)
    }
    if flagChanged("local-password-credential") {
        svc.MailProxy.LocalPasswordCredential = strings.TrimSpace(localPasswordCredential)
    }

    services[idx] = svc
    return putServices(vault, services)
}
```

Validation stays in the existing server-side service validation and `mail-proxy` preflight. The toggle command should reject a no-op call where none of the four mail proxy flags changed.

## Slice 4: Hygiene

Remove trailing whitespace in the concept patch. No behavior change.

## Test Strategy

- Unit test active relay shutdown with a stuck session.
- Unit test SMTP upstream server name extraction and STARTTLS config use.
- CLI tests for partial `mail_proxy` update preserving unrelated service fields.
- Regression tests for default Gmail SMTP behavior.
- Run:
  - `/usr/local/bin/go test ./...`
  - `npm run build --prefix web`
  - `openspec validate fix-mail-proxy-review-gaps --strict`
  - `git diff --check`
