# go-ircevo

[![Go Reference](https://pkg.go.dev/badge/github.com/kofany/go-ircevo.svg)](https://pkg.go.dev/github.com/kofany/go-ircevo)
[![Go Report Card](https://goreportcard.com/badge/github.com/kofany/go-ircevo)](https://goreportcard.com/report/github.com/kofany/go-ircevo)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kofany/go-ircevo)](go.mod)

> **Fork of [go-ircevent](https://github.com/thoj/go-ircevent) by Thomas Jager**

## Credits

- **Original Author**: Thomas Jager ([go-ircevent](https://github.com/thoj/go-ircevent), 2009)
- **Current Maintainer**: Jerzy Dąbrowski
- **Contributors**: See [GitHub Contributors](https://github.com/kofany/go-ircevo/graphs/contributors)

---

A robust, production-ready IRC client library for Go 1.23+ with advanced features optimized for mass deployments.

## ✨ Features

- **🔄 Smart Error Handling** - Intelligent ERROR categorization and reconnection strategies
- **👤 Advanced Nick Management** - Atomic operations with RFC 2812 compliance
- **🔐 Authentication** - SASL (PLAIN/EXTERNAL), server passwords, client certificates
- **🌐 Connectivity** - SOCKS5/HTTP proxy support, TLS/SSL, WebIRC
- **💬 DCC Chat** - Full DCC CHAT protocol support
- **🏭 Mass Deployment Ready** - Tested with 500+ concurrent connections
- **📊 Health Monitoring** - Real socket health checks with activity monitoring
- **🎯 Event System** - Flexible callback-based event handling
- **🔧 IRCv3** - CAP negotiation, message tags, SASL authentication

## 🆕 What's New in v1.2.7

**Disconnected Event Fix** - registered `DISCONNECTED` callbacks now run when a connection is closed:

- 🔔 **DISCONNECTED**: `Connection.AddCallback("DISCONNECTED", ...)` is emitted on manual disconnects and terminal `Loop()` exits
- 🧩 **Compatibility**: existing handlers start working after upgrading the module, with no application code changes
- 📉 **State Accuracy**: terminal disconnects clear the fully-connected state before callbacks run

See [CHANGELOG.md](CHANGELOG.md) for detailed release notes.

## 📦 Installation

```bash
# Latest version
go get github.com/kofany/go-ircevo

# Specific version (recommended for production)
go get github.com/kofany/go-ircevo@v1.2.7
```

**Requirements:** Go 1.23 or higher

## 🚀 Quick Start

```go
package main

import (
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("mynick", "myuser")
    conn.RealName = "My Real Name"
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#mychannel")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        log.Printf("<%s> %s", e.Nick, e.Message())
    })
    
    if err := conn.Connect("irc.example.com:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## 📚 Documentation

- **[Getting Started Guide](docs/GETTING_STARTED.md)** - Your first IRC bot
- **[API Reference](docs/API.md)** - Complete API documentation
- **[Architecture Overview](docs/ARCHITECTURE.md)** - Design and internals
- **[Examples](docs/EXAMPLES.md)** - Comprehensive usage examples
- **[Advanced Features](docs/ADVANCED.md)** - SASL, DCC, Proxy, TLS
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Migration Guide](docs/MIGRATION.md)** - Migrating from other libraries

## 🎯 Key Features

### Smart Error Handling

Automatically categorizes IRC ERROR messages and handles them intelligently:

```go
conn.SmartErrorHandling = true
conn.HandleErrorAsDisconnect = true
conn.MaxRecoverableReconnects = 3  // Limit reconnection attempts
```

**Error Categories:**
- **PermanentError** - Bans, permanent blocks (no reconnect)
- **ServerError** - Server overload, host limits (reconnect with delay)
- **NetworkError** - Network issues, timeouts (immediate reconnect)
- **RecoverableError** - Temporary issues (normal reconnect with limit)

### Advanced Nick Management

RFC 2812 compliant with atomic operations:

```go
conn.Nick("newnick")

status := conn.GetNickStatus()
if status.Confirmed {
    log.Printf("Nick confirmed: %s", status.Current)
}
```

### SASL Authentication

```go
conn.UseSASL = true
conn.SASLLogin = "username"
conn.SASLPassword = "password"
conn.SASLMech = "PLAIN"  // or "EXTERNAL"
```

### TLS/SSL Support

```go
conn.UseTLS = true
conn.TLSConfig = &tls.Config{
    InsecureSkipVerify: false,
    Certificates: []tls.Certificate{cert},
}
```

### Proxy Support

```go
conn.ProxyConfig = &irc.ProxyConfig{
    Type:     "socks5",
    Address:  "127.0.0.1:9050",  // Tor
    Username: "",
    Password: "",
}
```

### DCC Chat

```go
conn.DCCManager = irc.NewDCCManager()

conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    if strings.Contains(e.Message(), "!dcc") {
        conn.InitiateDCCChat(e.Nick)
    }
})
```

## 🏭 Production Features

### Mass Deployment Optimization

Designed and tested for running hundreds of concurrent connections:

```go
conn.EnableTimeoutFallback = false  // Prevent ghost bots (default)
conn.SmartErrorHandling = true
conn.MaxRecoverableReconnects = 3
```

### Connection Health Validation

```go
if conn.ValidateConnectionState() {
    log.Println("Connection healthy")
}

if conn.Connected() {
    log.Println("Fully connected and registered")
}
```

### Custom QUIT Messages

```go
conn.QuitMessage = "Bot shutting down for maintenance"
conn.Quit()  // Sends: QUIT :Bot shutting down for maintenance
```

## 📊 Event System

Register callbacks for any IRC command or numeric:

```go
conn.AddCallback("JOIN", func(e *irc.Event) {
    log.Printf("%s joined %s", e.Nick, e.Arguments[0])
})

conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    if e.Message() == "!ping" {
        conn.Privmsg(e.Arguments[0], "pong!")
    }
})

conn.AddCallback("*", func(e *irc.Event) {
    log.Printf("Event: %s", e.Code)
})
```

## 🔧 Configuration Options

```go
conn := irc.IRC("nick", "user")

// Connection settings
conn.Server = "irc.example.com:6667"
conn.Password = "serverpass"
conn.RealName = "My Real Name"

// Timing configuration
conn.Timeout = 300 * time.Second
conn.PingFreq = 15 * time.Minute
conn.KeepAlive = 4 * time.Minute

// Behavior configuration
conn.SmartErrorHandling = true
conn.HandleErrorAsDisconnect = true
conn.EnableTimeoutFallback = false
conn.MaxRecoverableReconnects = 3

// Debug mode
conn.Debug = true
conn.VerboseCallbackHandler = true
```

## 📖 Examples

Check out the [examples/](examples/) directory for complete working examples:

- **[Simple Bot](examples/simple/)** - Basic IRC bot with SSL/TLS
- **[Multi-Server Probe](examples/multi_server_probe/)** - Connect to multiple servers
- **[Tor Connection](examples/simple-tor.go/)** - IRC over Tor with SOCKS5

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](docs/CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## 📝 License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/kofany/go-ircevo/issues)
- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)

## 🗺️ Roadmap

- [ ] IRCv3.3 compliance
- [ ] File transfer support (DCC SEND)
- [ ] Connection pooling
- [ ] Metrics and observability
- [ ] More SASL mechanisms (SCRAM-SHA-256)

---

**go-ircevo** - Production-ready IRC for Go
