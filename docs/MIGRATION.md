# Migration Guide

This guide helps you migrate from other IRC libraries or older versions of go-ircevo.

## Table of Contents

- [Migrating from go-ircevent](#migrating-from-go-ircevent)
- [Migrating from goirc](#migrating-from-goirc)
- [Migrating from older go-ircevo](#migrating-from-older-go-ircevo)
- [Breaking Changes](#breaking-changes)
- [Recommended Defaults](#recommended-defaults)

## Migrating from go-ircevent

### Module Path

Update imports:

```go
- import irc "github.com/thoj/go-ircevent"
+ import irc "github.com/kofany/go-ircevo"
```

### Connection Creation

The API matches go-ircevent:

```go
conn := irc.IRC("nick", "user")
```

### Key Improvements

- Smart error handling (`SmartErrorHandling`, `HandleErrorAsDisconnect`)
- Nickname tracking (`GetNickStatus`, atomic changes)
- DCC chat support (`DCCManager`)
- SASL improvements (PLAIN/EXTERNAL, CAP registration control)
- Proxy configuration (`ProxyConfig`)
- Timeout management (`EnableTimeoutFallback` default false)

### Behavior Changes

| Feature | go-ircevent | go-ircevo |
|---------|-------------|-----------|
| Timeout fallback | Enabled by default | Disabled by default |
| Error handling | Manual | Smart categorization |
| QUIT delay | Immediate | 1-second delay to prevent ghost clients |
| Nick tracking | Basic | RFC-compliant with validation |
| Reconnect limit | Unlimited | Configurable (`MaxRecoverableReconnects`) |

### Migration Checklist

1. Replace import path
2. Set desired defaults:
   ```go
   conn.SmartErrorHandling = true
   conn.HandleErrorAsDisconnect = true
   conn.EnableTimeoutFallback = false
   conn.MaxRecoverableReconnects = 3
   ```
3. Initialize DCC manager if using DCC:
   ```go
   conn.DCCManager = irc.NewDCCManager()
   ```
4. Review callbacks for numeric handling (same as go-ircevent)
5. Update tests if they inspect internal state (use `GetNickStatus`)

## Migrating from goirc

If you're using github.com/fluffle/goirc or similar event-based libraries:

### Connection Setup

```go
conn := irc.IRC("nick", "user")
conn.AddCallback("001", func(e *irc.Event) {
    conn.Join("#channel")
})

if err := conn.Connect("irc.example.com:6667"); err != nil {
    log.Fatal(err)
}
conn.Loop()
```

### Event Handling

- goirc uses `client.HandleFunc("PRIVMSG", handler)`
- go-ircevo uses `conn.AddCallback("PRIVMSG", handler)`

### Differences

| Feature | goirc | go-ircevo |
|---------|-------|-----------|
| Callback signature | `func(*irc.Event)` | `func(*irc.Event)` (same) |
| TLS config | Inline | `UseTLS`, `TLSConfig` |
| SASL | Manual | Built-in support |
| Proxy | Limited | SOCKS5/HTTP support |
| Smart errors | Manual | Built-in |

## Migrating from older go-ircevo

### v1.1.x to v1.2.x

- `EnableTimeoutFallback` default changed to `false`
- Added `SmartErrorHandling` (default `false` for backward compatibility)
- Added `MaxRecoverableReconnects` (default `0` for unlimited)
- Added `RecoverableError` category

**Recommended update:**

```go
conn.SmartErrorHandling = true
conn.HandleErrorAsDisconnect = true
conn.MaxRecoverableReconnects = 3
```

### Nickname Handling

- `GetNick()` now always returns confirmed nickname
- Use `GetNickStatus()` for desired vs current
- `Nick` method is asynchronous; wait for confirmation via `GetNickStatus().Confirmed`

### DCC Support

- Initialize DCC manager for DCC features:
  ```go
  conn.DCCManager = irc.NewDCCManager()
  ```

### CAP Negotiation

- `CapVersion` introduced for CAP 302
- `RegistrationAfterCapEnd` controls NICK/USER timing (default `false`)

## Breaking Changes

| Version | Change |
|---------|--------|
| 1.2.0   | Added 1-second QUIT delay |
| 1.2.0   | `EnableTimeoutFallback` default false |
| 1.2.0   | DCC support requires initializing `DCCManager` |
| 1.2.0   | Smart error handling optional but recommended |

## Recommended Defaults

For production bots:

```go
conn := irc.IRC("botnick", "botuser")
conn.UseTLS = true
conn.SmartErrorHandling = true
conn.HandleErrorAsDisconnect = true
conn.MaxRecoverableReconnects = 3
conn.EnableTimeoutFallback = false
conn.DCCManager = irc.NewDCCManager()
```

## FAQ

### Q: Why did `EnableTimeoutFallback` change?

A: Timeout fallback caused "ghost bots" on some networks. Disabling it by default prevents duplicate connections.

### Q: Why is QUIT delayed by 1 second?

A: Some networks need a delay to process QUIT and avoid race conditions that leave phantom connections.

### Q: How do I disable smart error handling?

A: Simply leave `conn.SmartErrorHandling` as `false` (default) or set it explicitly.

### Q: How do I migrate custom reconnect logic?

A: Set `conn.HandleErrorAsDisconnect = false` to keep manual handling.

---

If you encounter migration issues, consult [TROUBLESHOOTING](TROUBLESHOOTING.md) or open a GitHub issue with details.
