# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-05-03

### Added

- Added `Connection.AutoNickRecoveryPostRegistration` to let applications disable automatic alternative `NICK` retries for 432/433/436/437 after registration.

### Changed

- Pre-registration nickname recovery remains enabled regardless of `AutoNickRecoveryPostRegistration`.

## [1.2.7] - 2026-05-02

### Added

- Added `EventDisconnected` for the `DISCONNECTED` callback event code.

### Fixed

- Emit `DISCONNECTED` callbacks from `Disconnect()` and terminal `Loop()` exits.
- Prevent duplicate `DISCONNECTED` callbacks within one connection lifecycle.
- Reset disconnected-event idempotency after a successful `Connect()` or `Reconnect()`.
- Clear `fullyConnected` before `DISCONNECTED` callbacks run.

## [1.2.6] - 2026-05-01

### Added

- Added `Connection.StopReconnect()` to synchronously prevent future automatic reconnect attempts.
- Added a bounded default callback timeout for new connections.

### Fixed

- Set the quit/reconnect-stop state before `Quit()` waits to send `QUIT`, removing a reconnect race window.
- Made connection shutdown idempotent around the internal `end` channel.
- Closed the socket before waiting in `Disconnect()` and avoided holding `irc.Lock` across `Wait()`.
- Applied `MaxRecoverableReconnects` to `ServerError` reconnects as well as `RecoverableError`.
- Prevented `SendRaw()` from blocking or panicking after the write channel has been closed.

## [1.2.5] - 2026-04-26

### Fixed

- Reset per-session registration state before each new connection so automatic reconnects resend `NICK` and `USER`
- Prevent stale CAP negotiation fallback goroutines from sending registration commands for an older connection session
- Reset `MaxRecoverableReconnects` accounting only after IRC registration succeeds instead of after a TCP reconnect

## [1.2.4] - 2026-04-17

### Fixed

- Corrected nickname tracking to use IRC RFC casemapping rules instead of plain string equality
- Fixed `ERR_ERRONEUSNICKNAME` recovery to generate RFC-valid fallback nicknames
- Stopped infinite retries of invalid or restricted nicknames after permanent nickname errors
- Prevented false pending nick changes and redundant `NICK` commands for RFC-equivalent nicknames

### Improved

- Hardened nickname fallback generation to keep alternatives within RFC nickname syntax and length limits
- Expanded automated tests for nickname tracking, RFC casemapping, and permanent nickname error handling
- Added `memory.db` to `.gitignore`

## [1.2.3] - 2025-11-13

### Security

- Fixed DCC CHAT IPv4/IPv6 conversion bug in `ip2int` function
- Added comprehensive DCC CHAT argument validation (IP addresses and port ranges)
- Fixed race condition in `CloseDCCChat` function that could cause panic

### Fixed

- Fixed IRC message parser minimum length check (changed from 5 to 1 character)
- Fixed `AnalyzeErrorMessage` default return value (now returns `RecoverableError` instead of `PermanentError`)

### Improved

- Better input validation for DCC protocol
- Enhanced concurrency safety in DCC chat operations
- Improved RFC 1459/2812 compliance for message parsing
- Better fault tolerance for unknown error messages

### Documentation

- Added comprehensive security audit report (`SECURITY_AUDIT_2025-11-13.md`)
- Updated README with release information
- Updated API documentation with new version

## [1.2.2] - 2025-11-03

### Fixed

- Make nickname tracking fully RFC-compliant and robust
- Fix post-registration nick tracking bug
- Fix error handler state corruption
- Fix race condition in nick change tracking
- Fix incomplete state management in connection lifecycle

### Added

- Three-state nick tracking system (desired, current, pending)
- Automatic retry mechanism for desired nicknames
- Comprehensive nick tracking audit report (`NICK_TRACKING_AUDIT.md`)

### Improved

- Enhanced nick change coordination
- Better RFC 2812 section 3.1.2 compliance
- Improved connection state management

## [1.2.1] - Earlier

### Added

- Smart ERROR handling with error categorization
- CAP negotiation improvements
- DCC CHAT support
- SASL authentication (PLAIN/EXTERNAL)
- Proxy support (SOCKS4/5, HTTP)
- TLS/SSL support
- IRCv3 message tags
- Mass deployment optimizations

### Improved

- Connection health monitoring
- Error recovery strategies
- Reconnection handling

---

[1.3.0]: https://github.com/kofany/go-ircevo/compare/v1.2.7...v1.3.0
[1.2.7]: https://github.com/kofany/go-ircevo/compare/v1.2.6...v1.2.7
[1.2.6]: https://github.com/kofany/go-ircevo/compare/v1.2.5...v1.2.6
[1.2.5]: https://github.com/kofany/go-ircevo/compare/v1.2.4...v1.2.5
[1.2.4]: https://github.com/kofany/go-ircevo/compare/v1.2.3...v1.2.4
[1.2.3]: https://github.com/kofany/go-ircevo/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/kofany/go-ircevo/releases/tag/v1.2.2
