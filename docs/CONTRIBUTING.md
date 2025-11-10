# Contributing to go-ircevo

Thank you for considering contributing to go-ircevo! This document provides guidelines for contributing to the project.

## Code of Conduct

Be respectful and constructive. We welcome contributions from everyone, regardless of experience level.

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report:

1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Search [existing issues](https://github.com/kofany/go-ircevo/issues)
3. Try to reproduce with the latest version

When filing a bug report, include:

- **Go version**: `go version`
- **Library version**: Check `go.mod` or git commit
- **Operating system and architecture**
- **Minimal reproducible example**
- **Expected behavior**
- **Actual behavior**
- **Debug logs** (sanitize credentials!)

Example:

```markdown
## Bug Report

**Go version:** 1.23.2
**OS:** Ubuntu 22.04 amd64
**go-ircevo version:** v1.2.2

### Expected behavior
Bot should reconnect after ERROR message.

### Actual behavior
Bot exits immediately after receiving ERROR.

### Minimal reproducible example
\`\`\`go
package main

import (
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("testbot", "test")
    conn.Debug = true
    conn.HandleErrorAsDisconnect = true
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#test")
    })
    
    if err := conn.Connect("irc.example.com:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
\`\`\`

### Debug logs
```
[connection logs here]
```
```

### Suggesting Enhancements

For feature requests:

1. Check if it's already implemented
2. Search existing issues for similar requests
3. Explain **why** this enhancement would be useful
4. Provide use cases and examples

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests if applicable
5. Run tests: `go test -v ./...`
6. Commit your changes with clear commit messages
7. Push to your fork
8. Open a Pull Request

## Development Setup

### Prerequisites

- Go 1.23 or higher
- Git
- Optional: IRC server for testing (or use public networks)

### Clone and Build

```bash
git clone https://github.com/kofany/go-ircevo.git
cd go-ircevo
go build
```

### Run Tests

```bash
go test -v ./...
```

### Run Examples

```bash
cd examples/simple
go run simple.go
```

## Coding Standards

### General Guidelines

- Follow standard Go conventions
- Use `gofmt` or `goimports` for formatting
- Keep functions focused and small
- Document exported functions and types
- Avoid unnecessary complexity

### Go-Specific Rules

From project AI rules:

1. **String concatenation**: Use direct concatenation instead of `fmt.Sprintf` for performance when appropriate

2. **IPv6 handling**: Enclose IPv6 addresses in square brackets for `net.Dial`:
   ```go
   if ip.To4() == nil {
       // IPv6
       addr = fmt.Sprintf("[%s]:%d", ip, port)
   } else {
       // IPv4
       addr = fmt.Sprintf("%s:%d", ip, port)
   }
   ```

3. **Random generators**: Don't use deprecated `rand.Seed()`. Use `rand.New(rand.NewSource(time.Now().UnixNano()))` for Go 1.20+

### Comments

- Comment exported functions, types, and methods
- Explain **why**, not **what** (code shows what)
- Use `//` for single-line comments
- Use godoc format for documentation

Example:

```go
// GetNickStatus returns detailed information about nickname state including
// confirmation status and any pending changes. This is useful for tracking
// whether a nick change has been acknowledged by the server.
func (irc *Connection) GetNickStatus() *NickStatus {
    // ...
}
```

### Error Handling

- Always check errors
- Return errors to callers when appropriate
- Log errors when they can't be returned

```go
if err := conn.Connect(server); err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}
```

### Testing

- Write tests for new features
- Use table-driven tests when appropriate
- Use meaningful test names

```go
func TestNickChange(t *testing.T) {
    tests := []struct {
        name     string
        oldNick  string
        newNick  string
        expected string
    }{
        {"simple change", "oldnick", "newnick", "newnick"},
        {"with underscore", "test", "test_", "test_"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

## Project Structure

```
go-ircevo/
├── irc.go              # Core connection and protocol logic
├── irc_struct.go       # Data structures and types
├── irc_callback.go     # Event handling and callbacks
├── irc_sasl.go         # SASL authentication
├── irc_dcc.go          # DCC CHAT implementation
├── irc_*_test.go       # Test files
├── examples/           # Example programs
│   ├── simple/
│   ├── multi_server_probe/
│   └── simple-tor.go/
└── docs/               # Documentation
    ├── API.md
    ├── GETTING_STARTED.md
    ├── ADVANCED.md
    └── ...
```

## Documentation

When adding features:

1. Update relevant documentation in `docs/`
2. Add godoc comments to exported symbols
3. Update examples if applicable
4. Update README.md if it's a major feature

## Commit Messages

Use clear, descriptive commit messages:

```
Add support for IRCv3 message tags

- Parse message tags from incoming messages
- Store tags in Event.Tags map
- Add test coverage for tag parsing

Fixes #123
```

Format:
- First line: Brief summary (50 chars or less)
- Blank line
- Detailed explanation if needed
- Reference issues/PRs

## Review Process

1. All PRs require review before merging
2. Address review comments promptly
3. Keep PRs focused on a single change
4. Rebase on main if needed

## License

By contributing, you agree that your contributions will be licensed under the BSD 3-Clause License.

## Questions?

- Open a GitHub issue for questions
- Check existing documentation
- Look at examples for guidance

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes for significant contributions
- Project README

Thank you for contributing to go-ircevo!
