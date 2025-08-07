
# go-ircevo

A robust, production-ready IRC client library for Go with advanced features for mass deployments.

## 🚀 Key Features

### **Smart Error Handling & Reconnection**

- **Intelligent ERROR categorization** - Automatically categorizes server errors (Permanent, Server, Network, Recoverable)
- **Smart reconnection logic** - Reconnects on recoverable errors, blocks on permanent bans
- **Configurable timeout fallback** - Prevents ghost bot issues in mass deployments
- **Enhanced connection validation** - Real socket health checks with activity monitoring

### **Advanced Nick Management**

- **Atomic nick operations** - Race condition prevention with `nickChangeInProgress` flag
- **RFC 2812 compliant** - Proper handling of all NICK error codes (431-437, 484)
- **Self-validation mechanism** - Automatic nick desynchronization detection and correction
- **Post-connection nick error handling** - Handles nick conflicts after initial connection

### **Production-Ready Features**

- **Mass deployment optimized** - Tested with 500+ concurrent connections
- **Activity-based false positive elimination** - Removed problematic connection detection
- **QUIT message support** - Custom quit messages with proper RFC compliance
- **1-second QUIT delay** - Prevents server-side race conditions
- **Connection state validation** - Comprehensive connection health monitoring

## 📦 Installation

```bash
go get github.com/kofany/go-ircevo
```

## 🔧 Quick Start

### Basic Usage

```go
package main

import (
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("mynick", "myuser")

    // Enable smart features (recommended for production)
    conn.SmartErrorHandling = true        // Intelligent error categorization
    conn.HandleErrorAsDisconnect = true   // Smart reconnection logic
    conn.EnableTimeoutFallback = false    // Prevent ghost bots (default: false)

    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#mychannel")
    })

    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        log.Printf("<%s> %s", e.Nick, e.Message())
    })

    err := conn.Connect("irc.example.com:6667")
    if err != nil {
        log.Fatal(err)
    }

    conn.Loop() // Blocks until disconnected
}
```

### Advanced Configuration

```go
conn := irc.IRC("mynick", "myuser")

// Smart Error Handling (NEW)
conn.SmartErrorHandling = true           // Enable intelligent error analysis
conn.HandleErrorAsDisconnect = true      // Handle ERRORs as disconnects

// Connection Management
conn.EnableTimeoutFallback = false       // Disable timeout fallback (prevents ghost bots)
conn.Timeout = 300 * time.Second         // Connection timeout
conn.PingFreq = 15 * time.Minute         // PING frequency
conn.KeepAlive = 4 * time.Minute         // Keep-alive timeout

// Custom QUIT message
conn.QuitMessage = "Goodbye from go-ircevo!"

// Debug mode
conn.Debug = true
```

## 🧠 Smart Error Handling

The library automatically categorizes IRC ERROR messages and handles them appropriately:

### Error Categories

- **PermanentError** - Bans, permanent blocks (no reconnect)
- **ServerError** - Server overload, host limits (reconnect with delay)
- **NetworkError** - Network issues, timeouts (immediate reconnect)
- **RecoverableError** - Temporary issues (normal reconnect)

### Example Error Handling

```go
conn.AddCallback("ERROR", func(e *irc.Event) {
    errorType := irc.AnalyzeErrorMessage(e.Message())
    log.Printf("Received %s: %s", errorType.String(), e.Message())

    switch errorType {
    case irc.PermanentError:
        log.Printf("Permanent error - bot will not reconnect")
        // Custom cleanup logic
    case irc.ServerError:
        log.Printf("Server error - will retry with delay")
        // Maybe implement exponential backoff
    case irc.RecoverableError:
        log.Printf("Recoverable error - normal reconnection")
    }
})
```

## 🔄 Nick Management

### Atomic Nick Changes

```go
// Safe nick changes with race condition prevention
conn.Nick("newnick")

// Check nick status
status := conn.GetNickStatus()
if status.Confirmed {
    log.Printf("Nick confirmed: %s", status.Current)
} else {
    log.Printf("Nick change pending: %s -> %s", status.Current, status.Desired)
}
```

### Self-Validation

The library automatically validates and corrects nick desynchronization:

```go
// Automatic validation in JOIN/PART/PRIVMSG events
// If event nick != internal nick, auto-corrects silently
conn.ValidateOwnNick(eventNick) // Called automatically
```

## 🔌 Connection Management

### Connection State Validation

```go
// Comprehensive connection health check
if conn.ValidateConnectionState() {
    log.Println("Connection is healthy")
} else {
    log.Println("Connection issues detected")
}

// Check if fully connected (not just socket connected)
if conn.Connected() {
    log.Println("Fully connected and registered")
}
```

### Custom QUIT Messages

```go
// Set custom quit message
conn.QuitMessage = "Bot shutting down for maintenance"
conn.Quit() // Sends: QUIT :Bot shutting down for maintenance

// Or use default
conn.Quit() // Sends: QUIT
```

## 🏭 Mass Deployment Features

### Preventing Ghost Bots

```go
// Disable timeout fallback to prevent ghost bots (default: false)
conn.EnableTimeoutFallback = false

// This prevents false positive connection detection that can
// create "ghost bots" in mass deployments (500+ concurrent connections)
```

### Smart Error Handling for Production

```go
conn.SmartErrorHandling = true // Enable intelligent error categorization

// The library will automatically:
// - Reconnect on ServerError (host limits, server overload)
// - Reconnect on NetworkError (network issues)
// - NOT reconnect on PermanentError (bans, permanent blocks)
// - Handle RecoverableError normally
```

## 📚 Migration Guide

### From Previous Versions

If you're upgrading from an older version:

```go
// OLD (may cause issues in mass deployments)
conn.EnableTimeoutFallback = true  // Can create ghost bots

// NEW (recommended for production)
conn.EnableTimeoutFallback = false // Prevents ghost bots
conn.SmartErrorHandling = true     // Intelligent error handling
```

### Error Handling Changes

```go
// OLD - Manual error handling
conn.AddCallback("ERROR", func(e *irc.Event) {
    // Manual reconnect logic
    go reconnectAfterDelay()
})

// NEW - Automatic smart handling
conn.SmartErrorHandling = true
conn.HandleErrorAsDisconnect = true
// Library handles reconnection automatically based on error type
```

## 🧪 Testing

Run the test suite:

```bash
go test -v
```

For connection tests (requires network):

```bash
go test -v -run TestConnection
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/kofany/go-ircevo/issues)
- **Documentation**: This README and inline code documentation
- **Examples**: See the `examples/` directory

---

**go-ircevo** - Production-ready IRC client library for Go with advanced features for mass deployments.
