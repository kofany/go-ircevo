# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[1.2.3]: https://github.com/kofany/go-ircevo/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/kofany/go-ircevo/releases/tag/v1.2.2
