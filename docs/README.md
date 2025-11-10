# go-ircevo Documentation

Welcome to the go-ircevo documentation!

## Quick Links

- **[Getting Started](GETTING_STARTED.md)** - Your first IRC bot in Go
- **[API Reference](API.md)** - Complete API documentation
- **[Examples](EXAMPLES.md)** - Real-world code examples
- **[Advanced Features](ADVANCED.md)** - SASL, TLS, Proxy, DCC
- **[Architecture](ARCHITECTURE.md)** - Design and internals
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common problems and solutions
- **[Migration Guide](MIGRATION.md)** - Migrating from other libraries
- **[Contributing](CONTRIBUTING.md)** - How to contribute

## Documentation Overview

### For Beginners

Start here if you're new to go-ircevo or IRC bot development:

1. Read [Getting Started](GETTING_STARTED.md)
2. Run the [simple example](../examples/simple/)
3. Browse [Examples](EXAMPLES.md) for common patterns

### For Experienced Users

If you're familiar with IRC protocols:

- Jump to [API Reference](API.md) for complete method documentation
- Check [Advanced Features](ADVANCED.md) for SASL, proxies, and DCC
- Review [Architecture](ARCHITECTURE.md) to understand internals

### For Contributors

Contributing to go-ircevo:

- Read [Contributing](CONTRIBUTING.md) for guidelines
- Check [Architecture](ARCHITECTURE.md) to understand the codebase
- Follow the coding standards outlined in CONTRIBUTING.md

## Quick Start

```go
package main

import (
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("mybot", "botuser")
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#test")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        log.Printf("<%s> %s", e.Nick, e.Message())
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Key Features

- **Smart Error Handling** - Automatically categorizes errors and adjusts reconnection strategy
- **Nick Management** - RFC 2812 compliant nickname tracking with atomic operations
- **SASL Authentication** - PLAIN and EXTERNAL mechanisms
- **TLS/SSL** - Full TLS support with client certificates
- **Proxy Support** - SOCKS5 and HTTP proxies (Tor-compatible)
- **DCC Chat** - Complete DCC CHAT implementation
- **IRCv3** - CAP negotiation and message tags
- **Production Ready** - Tested with 500+ concurrent connections

## Common Tasks

### Connecting with TLS

```go
conn.UseTLS = true
conn.TLSConfig = &tls.Config{ServerName: "irc.libera.chat"}
err := conn.Connect("irc.libera.chat:6697")
```

### SASL Authentication

```go
conn.UseSASL = true
conn.SASLLogin = "account"
conn.SASLPassword = "password"
conn.SASLMech = "PLAIN"
```

### Using a Proxy (Tor)

```go
conn.ProxyConfig = &irc.ProxyConfig{
    Type:    "socks5",
    Address: "127.0.0.1:9050",
}
```

### Handling Commands

```go
conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    if e.Message() == "!ping" {
        conn.Privmsg(e.Arguments[0], "Pong!")
    }
})
```

## Need Help?

- Check [Troubleshooting](TROUBLESHOOTING.md)
- Browse [Examples](EXAMPLES.md)
- Read the [API Reference](API.md)
- Search [GitHub Issues](https://github.com/kofany/go-ircevo/issues)
- File a new issue with details

## Additional Resources

- [RFC 2812 - IRC Protocol](https://tools.ietf.org/html/rfc2812)
- [IRCv3 Specifications](https://ircv3.net/)
- [IRC Numerics Reference](https://www.alien.net.au/irc/irc2numerics.html)

## License

BSD 3-Clause License - see [LICENSE](../LICENSE)
