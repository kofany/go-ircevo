# Architecture Overview

This document explains the internal architecture of go-ircevo.

## High-Level Design

go-ircevo is an event-driven IRC client built around the `Connection` type. Each connection manages:

- Socket lifecycle (connect, reconnect, disconnect)
- Read, write, and ping goroutines
- Event parsing and dispatch
- Intelligent error analysis and reconnection
- Nickname state management and validation
- DCC chat orchestration

![Architecture Diagram](https://mermaid.ink/img/pako:eNqNVNtugzAQ_BXLN0MZB60qBRCoAbVWqhQdFWxULNhMidpMU-K_MywlyXW4uaN-53pnd5t0Y2vxO2Y5so0SoDdrNaiACbIuBsndCCz9msDhgs0BLzMxPbiQhG1oMoKrwY_uqM0S-Sg5mIRVpmABXpNWmmvHCWRLwEcVwhrfvRWjPnEjKPbIMBoJay9MoPt4rJ3dGOM1YGklmlswqBsg1j9u2C30qtq1CF6KexEiaLZ0kC0yK4Y35hUdDaEVX9gU8oi0JcbgNxiMXuPaUhKpQVHHXvvWEDVEVKPjNKXBF9XNchtWIlIO4yHJenC0OG37E8x9acE2BZnx8TI5KfmTRFlcvmS8iMCFJfEC0LCsqLB7eaDG_zb5kgHyH-tHwZxfoeStpcaiiFwQEG5-OJ983IU8-E6C)

*Diagram: Connection lifecycle and goroutines*

## Core Components

### Connection

The `Connection` struct in `irc_struct.go` holds all mutable state:

- Authentication and identity (nick, user, real name)
- Transport configuration (TLS, proxies, WebIRC)
- Timers (timeout, ping frequency, keep-alive)
- Error handling behavior (smart error handling, reconnect limits)
- DCC manager and callback registry

### Goroutines

Each active connection runs three primary goroutines:

1. **readLoop** - Reads from socket, parses messages into `Event`, dispatches callbacks
2. **writeLoop** - Consumes outbound message channel and writes to socket
3. **pingLoop** - Sends periodic PING and manages nickname resynchronization

These goroutines communicate via:

- `pwrite` channel (outgoing messages)
- `end` channel (shutdown control)
- `Error` channel (reported errors)

### Event System

- `parseToEvent` converts raw IRC lines to `Event` objects
- `AddCallback` registers handlers per event code (including `*` wildcard)
- `RunCallbacks` invokes handlers synchronously in registration order
- `VerboseCallbackHandler` logs callback dispatch for debugging

### Error Handling

- Incoming `ERROR` messages are analyzed by `AnalyzeErrorMessage`
- Four categories: Recoverable, Permanent, Server, Network
- `HandleErrorAsDisconnect` controls whether ERROR triggers reconnection logic
- `MaxRecoverableReconnects` limits reconnection attempts
- Loop monitors `Error` channel for reconnection logic

### Nickname Tracking

- `nick` - desired nickname
- `nickcurrent` - confirmed nickname
- `nickPending` - pending change awaiting confirmation
- `nickChangeInProgress` flag prevents race conditions
- `GetNickStatus` exposes current nickname state to callers
- `ValidateOwnNick` auto-corrects desynchronization

A detailed analysis is available in [NICK_TRACKING_AUDIT.md](../NICK_TRACKING_AUDIT.md).

### CAP Negotiation

- `RequestCaps` lists capabilities to request
- `negotiateCaps` manages CAP LS/REQ/ACK flow
- `CapVersion` allows CAP 302 vs legacy LS
- `RegistrationAfterCapEnd` delays NICK/USER registration until CAP END

### SASL Authentication

- Configured via `UseSASL`, `SASLLogin`, `SASLPassword`, `SASLMech`
- `setupSASLCallbacks` handles CAP ACK, AUTHENTICATE, and numeric responses
- Supports `PLAIN` and `EXTERNAL` mechanisms

### DCC Chat

- `DCCManager` handles active DCC sessions (map of nick → chat)
- `InitiateDCCChat` and `handleIncomingDCCChat` manage connection setup
- `readDCCChat` and `writeDCCChat` run per-chat goroutines for message flow
- IPv4/IPv6 aware (`ip2int` utility, bracketed IPv6 addresses)

### Proxy Support

- `ProxyConfig` supports SOCKS5 and HTTP proxies
- `SetProxy` helper for legacy configuration
- Integrates with `golang.org/x/net/proxy` and `h12.io/socks`
- Allows Tor routing (see `examples/simple-tor.go`)

### Health Monitoring

- `lastMessage` timestamp tracked per message
- `pingLoop` sends PING when `KeepAlive` threshold reached
- `ValidateConnectionState` examines socket state and activity

## Lifecycle

1. **Initialization** - `IRC()` factory sets defaults (TLS off, timeouts)
2. **Configuration** - Caller sets fields (SASL, TLS, proxies, features)
3. **Connect** - Dial socket, start goroutines, begin CAP negotiation
4. **Registration** - Send NICK/USER (respect CAP ordering), wait for 001
5. **Event Processing** - readLoop → parse → RunCallbacks → user handlers
6. **Error Detection** - read/write/ping loops report errors to `Error` channel
7. **Reconnection** - `Loop()` orchestrates reconnect attempts based on error type
8. **Shutdown** - `Quit()` or `Disconnect()` close channels and goroutines

## Goroutine Safety

- `Connection` embeds `sync.Mutex` and `sync.WaitGroup`
- Critical sections guard access to nickname state, callbacks, DCC map
- `pwrite` channel ensures serialized writes
- `eventsMutex` protects callback registry

## Dependency Graph

- `irc.go` - Core transport and protocol logic
- `irc_struct.go` - Data structures and exported types
- `irc_callback.go` - Callback registry and event dispatch
- `irc_sasl.go` - SASL capability handling
- `irc_dcc.go` - DCC CHAT implementation
- `irc_*_test.go` - Unit tests

External dependencies:

- `golang.org/x/net` - Proxy support
- `golang.org/x/text/encoding` - Character encoding support
- `h12.io/socks` - SOCKS proxy implementation

## Key Design Decisions

- **Event-Driven**: Callbacks keep business logic separate from protocol handling
- **Smart Reconnect**: Error analysis prevents infinite loops and respects bans
- **Nick Synchronization**: Internal state machine ensures correctness across servers
- **Separation of Concerns**: SASL, DCC, and callbacks modularized into separate files
- **Mass Deployment Ready**: Timeouts, health checks, and recoverable limits tuned for large fleets

## Extending go-ircevo

- **Custom CAPs**: Append to `RequestCaps` before connecting
- **Additional SASL Mechanisms**: Extend `setupSASLCallbacks`
- **New Event Types**: Register callbacks for custom numerics/commands
- **Custom Error Policies**: Wrap or observe `ErrorChan`
- **Metrics/Logging**: Tap into `VerboseCallbackHandler` and internal logging

## Testing Strategy

- Unit tests cover parsing, nick state, SASL flows, and connection lifecycle
- Fuzz tests (`irc_test_fuzz.go`) harden parser against malformed input
- Integration examples connect to real networks (requires network access)

## Performance Characteristics

- Optimized for long-lived persistent connections
- Minimal allocations during steady-state event processing
- Controlled goroutine count (3 core loops + per-DCC sessions)
- Tunable timeouts and PING intervals for deployment needs

---

For further details, inspect the source files (`irc.go`, `irc_struct.go`, `irc_callback.go`) and refer to the [API Reference](API.md).
