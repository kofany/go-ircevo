# Getting Started with go-ircevo

This guide will help you create your first IRC bot using go-ircevo.

## Prerequisites

- Go 1.23 or higher
- Basic understanding of IRC protocol
- A test IRC server to connect to

## Installation

```bash
go get github.com/kofany/go-ircevo
```

## Your First Bot

### 1. Basic Connection

Create a file `main.go`:

```go
package main

import (
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    // Create a new connection
    conn := irc.IRC("mybot", "botuser")
    conn.RealName = "My First IRC Bot"
    
    // Connect to the server
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    // Start the main loop
    conn.Loop()
}
```

Run it:

```bash
go run main.go
```

Your bot will connect to the IRC server but won't do anything yet.

### 2. Adding Event Handlers

Let's make the bot join a channel and respond to messages:

```go
package main

import (
    "log"
    "strings"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("mybot", "botuser")
    conn.RealName = "My First IRC Bot"
    
    // Join channel after successful connection
    conn.AddCallback("001", func(e *irc.Event) {
        log.Println("Connected! Joining #test")
        conn.Join("#test")
    })
    
    // Respond to channel messages
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        channel := e.Arguments[0]
        message := e.Message()
        
        log.Printf("<%s> %s", e.Nick, message)
        
        if strings.HasPrefix(message, "!ping") {
            conn.Privmsg(channel, "Pong!")
        }
    })
    
    // Handle channel joins
    conn.AddCallback("JOIN", func(e *irc.Event) {
        if e.Nick == conn.GetNick() {
            log.Printf("We joined %s", e.Arguments[0])
        } else {
            log.Printf("%s joined %s", e.Nick, e.Arguments[0])
        }
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

### 3. Using TLS/SSL

For secure connections:

```go
import (
    "crypto/tls"
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("mybot", "botuser")
    
    // Enable TLS
    conn.UseTLS = true
    conn.TLSConfig = &tls.Config{
        InsecureSkipVerify: false,
    }
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#test")
    })
    
    // Connect to SSL port
    if err := conn.Connect("irc.libera.chat:6697"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Common Event Codes

Here are the most commonly used IRC event codes:

### Numeric Events

| Code | Description |
|------|-------------|
| `001` | Welcome message (connection successful) |
| `353` | Names list for a channel |
| `366` | End of names list |
| `433` | Nickname already in use |
| `464` | Bad password |

### Command Events

| Code | Description |
|------|-------------|
| `PRIVMSG` | Private message or channel message |
| `NOTICE` | Notice message |
| `JOIN` | User joins a channel |
| `PART` | User leaves a channel |
| `QUIT` | User disconnects |
| `NICK` | User changes nickname |
| `KICK` | User is kicked from channel |
| `MODE` | Channel or user mode change |
| `TOPIC` | Channel topic |
| `PING` | Server ping (handled automatically) |
| `ERROR` | Error from server |

## Event Object Structure

When your callback is invoked, it receives an `Event` object:

```go
conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    // e.Code       - Event code ("PRIVMSG")
    // e.Nick       - Sender's nickname
    // e.User       - Sender's username
    // e.Host       - Sender's hostname
    // e.Source     - Full source (nick!user@host)
    // e.Arguments  - Event arguments (channel/target, etc.)
    // e.Message()  - The actual message text
    // e.Tags       - IRCv3 message tags (if any)
    
    channel := e.Arguments[0]
    message := e.Message()
    
    log.Printf("In %s, %s said: %s", channel, e.Nick, message)
})
```

## Handling Private Messages vs Channel Messages

```go
conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    target := e.Arguments[0]
    message := e.Message()
    
    // Check if it's a private message
    if target == conn.GetNick() {
        // It's a PM, reply directly to the sender
        conn.Privmsg(e.Nick, "You sent me: " + message)
    } else {
        // It's a channel message
        conn.Privmsg(target, "Message received in channel")
    }
})
```

## Configuration Options

### Basic Configuration

```go
conn := irc.IRC("nick", "user")
conn.RealName = "Real Name"       // GECOS/real name
conn.Password = "serverpass"       // Server password (not NickServ)
conn.Debug = true                  // Enable debug logging
```

### Timing Configuration

```go
conn.Timeout = 300 * time.Second   // Connection timeout
conn.PingFreq = 15 * time.Minute   // How often to ping
conn.KeepAlive = 4 * time.Minute   // Keep-alive timeout
```

### Reconnection Behavior

```go
conn.HandleErrorAsDisconnect = true    // Treat ERROR as disconnect
conn.SmartErrorHandling = true         // Enable smart error analysis
conn.MaxRecoverableReconnects = 3      // Limit reconnection attempts
```

## Error Handling

### Handling Connection Errors

```go
if err := conn.Connect("irc.example.com:6667"); err != nil {
    log.Printf("Connection failed: %v", err)
    return
}
```

### Handling Runtime Errors

```go
conn.AddCallback("ERROR", func(e *irc.Event) {
    log.Printf("Server ERROR: %s", e.Message())
})

conn.AddCallback("433", func(e *irc.Event) {
    // Nickname in use
    conn.Nick(conn.GetNick() + "_")
})
```

## Best Practices

### 1. Always Handle the Welcome Message

```go
conn.AddCallback("001", func(e *irc.Event) {
    // Do your setup here (join channels, identify to services, etc.)
    conn.Join("#mychannel")
})
```

### 2. Handle Nickname Conflicts

```go
conn.AddCallback("433", func(e *irc.Event) {
    // Nick in use, try another
    conn.Nick(conn.GetNick() + "_")
})
```

### 3. Use Debug Mode During Development

```go
conn.Debug = true
conn.VerboseCallbackHandler = true
```

### 4. Graceful Shutdown

```go
import (
    "os"
    "os/signal"
    "syscall"
)

func main() {
    conn := irc.IRC("mybot", "botuser")
    
    // Setup signal handler
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        log.Println("Shutting down...")
        conn.QuitMessage = "Bot shutting down"
        conn.Quit()
    }()
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#test")
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Complete Example

Here's a complete, production-ready bot:

```go
package main

import (
    "crypto/tls"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"
    
    irc "github.com/kofany/go-ircevo"
)

func main() {
    // Create connection
    conn := irc.IRC("mybot", "botuser")
    conn.RealName = "My IRC Bot v1.0"
    
    // Configure TLS
    conn.UseTLS = true
    conn.TLSConfig = &tls.Config{
        InsecureSkipVerify: false,
    }
    
    // Configure behavior
    conn.SmartErrorHandling = true
    conn.HandleErrorAsDisconnect = true
    conn.MaxRecoverableReconnects = 3
    
    // Enable debug during development
    conn.Debug = true
    
    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        log.Println("Received shutdown signal")
        conn.QuitMessage = "Bot shutting down gracefully"
        conn.Quit()
    }()
    
    // Connection successful
    conn.AddCallback("001", func(e *irc.Event) {
        log.Println("Connected successfully!")
        conn.Join("#test")
    })
    
    // Joined channel
    conn.AddCallback("JOIN", func(e *irc.Event) {
        if e.Nick == conn.GetNick() {
            channel := e.Arguments[0]
            log.Printf("Successfully joined %s", channel)
            conn.Privmsg(channel, "Hello! I'm a bot powered by go-ircevo")
        }
    })
    
    // Handle messages
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        target := e.Arguments[0]
        message := e.Message()
        
        // Ignore our own messages
        if e.Nick == conn.GetNick() {
            return
        }
        
        log.Printf("[%s] <%s> %s", target, e.Nick, message)
        
        // Handle commands
        if strings.HasPrefix(message, "!ping") {
            conn.Privmsg(target, "Pong!")
        } else if strings.HasPrefix(message, "!time") {
            conn.Privmsg(target, time.Now().Format(time.RFC3339))
        } else if strings.HasPrefix(message, "!help") {
            conn.Privmsg(target, "Commands: !ping, !time, !help")
        }
    })
    
    // Handle nickname conflicts
    conn.AddCallback("433", func(e *irc.Event) {
        conn.Nick(conn.GetNick() + "_")
    })
    
    // Handle errors
    conn.AddCallback("ERROR", func(e *irc.Event) {
        log.Printf("Server ERROR: %s", e.Message())
    })
    
    // Connect
    log.Println("Connecting to IRC server...")
    if err := conn.Connect("irc.libera.chat:6697"); err != nil {
        log.Fatal(err)
    }
    
    // Main loop
    conn.Loop()
    log.Println("Bot stopped")
}
```

## Next Steps

- Read the [API Reference](API.md) for complete API documentation
- Check out [Advanced Features](ADVANCED.md) for SASL, DCC, and more
- Browse [Examples](EXAMPLES.md) for more usage patterns
- See [Troubleshooting](TROUBLESHOOTING.md) if you run into issues
