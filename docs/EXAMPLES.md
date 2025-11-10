# Examples

Comprehensive examples demonstrating go-ircevo usage.

## Table of Contents

- [Simple Bot](#simple-bot)
- [Echo Bot](#echo-bot)
- [Channel Logger](#channel-logger)
- [Multi-Channel Bot](#multi-channel-bot)
- [Command Bot](#command-bot)
- [Admin Bot with Authentication](#admin-bot-with-authentication)
- [DCC Chat Bot](#dcc-chat-bot)
- [Multi-Server Bot](#multi-server-bot)
- [Tor Connection](#tor-connection)
- [Metrics Bot](#metrics-bot)

## Simple Bot

Basic connection with join and message handling:

```go
package main

import (
    "log"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("simplebot", "bot")
    conn.RealName = "Simple Bot"
    
    conn.AddCallback("001", func(e *irc.Event) {
        log.Println("Connected!")
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

## Echo Bot

Repeats everything said in a channel:

```go
package main

import (
    "log"
    "strings"
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("echobot", "echobot")
    conn.RealName = "Echo Bot"
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#echotest")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        channel := e.Arguments[0]
        message := e.Message()
        nick := e.Nick
        
        // Don't echo ourselves
        if nick == conn.GetNick() {
            return
        }
        
        // Don't echo commands
        if strings.HasPrefix(message, "!") {
            return
        }
        
        conn.Privmsg(channel, message)
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Channel Logger

Logs all channel activity to files:

```go
package main

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
    
    irc "github.com/kofany/go-ircevo"
)

type Logger struct {
    mu    sync.Mutex
    files map[string]*os.File
}

func NewLogger() *Logger {
    return &Logger{
        files: make(map[string]*os.File),
    }
}

func (l *Logger) Log(channel, line string) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    channel = strings.TrimPrefix(channel, "#")
    
    if _, ok := l.files[channel]; !ok {
        filename := filepath.Join("logs", fmt.Sprintf("%s.log", channel))
        os.MkdirAll("logs", 0755)
        f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            return err
        }
        l.files[channel] = f
    }
    
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    _, err := fmt.Fprintf(l.files[channel], "[%s] %s\n", timestamp, line)
    return err
}

func (l *Logger) Close() {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    for _, f := range l.files {
        f.Close()
    }
}

func main() {
    logger := NewLogger()
    defer logger.Close()
    
    conn := irc.IRC("logbot", "logbot")
    conn.RealName = "Channel Logger Bot"
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#golang")
        conn.Join("#python")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        channel := e.Arguments[0]
        if !strings.HasPrefix(channel, "#") {
            return
        }
        
        line := fmt.Sprintf("<%s> %s", e.Nick, e.Message())
        logger.Log(channel, line)
    })
    
    conn.AddCallback("JOIN", func(e *irc.Event) {
        channel := e.Arguments[0]
        line := fmt.Sprintf("--> %s joined", e.Nick)
        logger.Log(channel, line)
    })
    
    conn.AddCallback("PART", func(e *irc.Event) {
        channel := e.Arguments[0]
        reason := ""
        if len(e.Arguments) > 1 {
            reason = " (" + e.Arguments[1] + ")"
        }
        line := fmt.Sprintf("<-- %s left%s", e.Nick, reason)
        logger.Log(channel, line)
    })
    
    conn.AddCallback("QUIT", func(e *irc.Event) {
        reason := e.Message()
        line := fmt.Sprintf("<-- %s quit (%s)", e.Nick, reason)
        // Log to all channels (simplified)
        for channel := range logger.files {
            logger.Log("#"+channel, line)
        }
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Multi-Channel Bot

Manages multiple channels with different behaviors:

```go
package main

import (
    "log"
    "strings"
    "time"
    irc "github.com/kofany/go-ircevo"
)

type ChannelConfig struct {
    Echo    bool
    Greet   bool
    Command bool
}

var configs = map[string]ChannelConfig{
    "#test":   {Echo: true, Greet: true, Command: true},
    "#quiet":  {Echo: false, Greet: false, Command: false},
    "#helper": {Echo: false, Greet: true, Command: true},
}

func main() {
    conn := irc.IRC("multibot", "multibot")
    
    conn.AddCallback("001", func(e *irc.Event) {
        for channel := range configs {
            conn.Join(channel)
        }
    })
    
    conn.AddCallback("JOIN", func(e *irc.Event) {
        channel := e.Arguments[0]
        cfg := configs[channel]
        
        if cfg.Greet && e.Nick != conn.GetNick() {
            conn.Privmsg(channel, "Welcome, "+e.Nick+"!")
        }
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        channel := e.Arguments[0]
        message := e.Message()
        cfg := configs[channel]
        
        if e.Nick == conn.GetNick() {
            return
        }
        
        if cfg.Echo {
            conn.Privmsg(channel, message)
        }
        
        if cfg.Command && strings.HasPrefix(message, "!time") {
            conn.Privmsg(channel, "The time is "+time.Now().Format(time.RFC3339))
        }
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Command Bot

Full-featured command handler:

```go
package main

import (
    "fmt"
    "log"
    "strings"
    "time"
    
    irc "github.com/kofany/go-ircevo"
)

type Command struct {
    Help    string
    Handler func(conn *irc.Connection, e *irc.Event, args []string)
}

var commands = map[string]Command{
    "!ping": {
        Help: "Test bot responsiveness",
        Handler: func(conn *irc.Connection, e *irc.Event, args []string) {
            conn.Privmsg(e.Arguments[0], "Pong!")
        },
    },
    "!time": {
        Help: "Show current time",
        Handler: func(conn *irc.Connection, e *irc.Event, args []string) {
            conn.Privmsg(e.Arguments[0], time.Now().Format(time.RFC3339))
        },
    },
    "!echo": {
        Help: "Echo your message",
        Handler: func(conn *irc.Connection, e *irc.Event, args []string) {
            if len(args) == 0 {
                conn.Privmsg(e.Arguments[0], "Usage: !echo <message>")
                return
            }
            conn.Privmsg(e.Arguments[0], strings.Join(args, " "))
        },
    },
    "!help": {
        Help: "Show this help",
        Handler: func(conn *irc.Connection, e *irc.Event, args []string) {
            target := e.Arguments[0]
            conn.Privmsg(target, "Available commands:")
            for name, cmd := range commands {
                conn.Privmsg(target, fmt.Sprintf("  %s - %s", name, cmd.Help))
            }
        },
    },
}

func main() {
    conn := irc.IRC("cmdbot", "cmdbot")
    conn.RealName = "Command Bot"
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#test")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        message := e.Message()
        parts := strings.Fields(message)
        
        if len(parts) == 0 {
            return
        }
        
        cmdName := parts[0]
        args := parts[1:]
        
        if cmd, ok := commands[cmdName]; ok {
            cmd.Handler(conn, e, args)
        }
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Admin Bot with Authentication

Bot with admin commands:

```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "log"
    "strings"
    "sync"
    
    irc "github.com/kofany/go-ircevo"
)

var (
    adminsMu sync.RWMutex
    admins   = make(map[string]bool)
    adminPwd = hashPassword("mysecretpassword")
)

func hashPassword(pwd string) string {
    h := sha256.Sum256([]byte(pwd))
    return hex.EncodeToString(h[:])
}

func isAdmin(nick string) bool {
    adminsMu.RLock()
    defer adminsMu.RUnlock()
    return admins[nick]
}

func setAdmin(nick string, status bool) {
    adminsMu.Lock()
    defer adminsMu.Unlock()
    admins[nick] = status
}

func main() {
    conn := irc.IRC("adminbot", "adminbot")
    conn.RealName = "Admin Bot"
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#admin")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        target := e.Arguments[0]
        message := e.Message()
        parts := strings.Fields(message)
        
        if len(parts) == 0 {
            return
        }
        
        // Authentication command (in PM only)
        if target == conn.GetNick() && parts[0] == "!auth" {
            if len(parts) < 2 {
                conn.Notice(e.Nick, "Usage: !auth <password>")
                return
            }
            
            if hashPassword(parts[1]) == adminPwd {
                setAdmin(e.Nick, true)
                conn.Notice(e.Nick, "Authentication successful!")
            } else {
                conn.Notice(e.Nick, "Authentication failed!")
            }
            return
        }
        
        // Admin commands
        if !isAdmin(e.Nick) {
            return
        }
        
        switch parts[0] {
        case "!kick":
            if len(parts) < 2 {
                conn.Notice(e.Nick, "Usage: !kick <nick>")
                return
            }
            conn.Kick(parts[1], target, "Kicked by admin")
            
        case "!say":
            if len(parts) < 2 {
                return
            }
            conn.Privmsg(target, strings.Join(parts[1:], " "))
            
        case "!join":
            if len(parts) < 2 {
                return
            }
            conn.Join(parts[1])
            
        case "!part":
            if len(parts) < 2 {
                conn.Part(target)
            } else {
                conn.Part(parts[1])
            }
        }
    })
    
    // Remove admin status on quit/nick change
    conn.AddCallback("QUIT", func(e *irc.Event) {
        setAdmin(e.Nick, false)
    })
    
    conn.AddCallback("NICK", func(e *irc.Event) {
        setAdmin(e.Nick, false)
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## DCC Chat Bot

Interactive DCC chat:

```go
package main

import (
    "log"
    "strings"
    "time"
    
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("dccbot", "dccbot")
    conn.DCCManager = irc.NewDCCManager()
    
    conn.AddCallback("001", func(e *irc.Event) {
        conn.Join("#dcc")
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        message := e.Message()
        
        if strings.HasPrefix(message, "!dcc") {
            if err := conn.InitiateDCCChat(e.Nick); err != nil {
                log.Printf("Failed to initiate DCC: %v", err)
            } else {
                conn.Privmsg(e.Nick, "DCC CHAT initiated!")
            }
        }
    })
    
    // Monitor DCC chats
    go func() {
        for {
            time.Sleep(100 * time.Millisecond)
            for _, nick := range conn.ListActiveDCCChats() {
                msg, err := conn.GetDCCMessage(nick)
                if err != nil {
                    continue
                }
                log.Printf("[DCC:%s] %s", nick, msg)
                conn.SendDCCMessage(nick, "Echo: "+msg)
            }
        }
    }()
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Multi-Server Bot

Connect to multiple servers simultaneously:

```go
package main

import (
    "crypto/tls"
    "log"
    "strings"
    "sync"
    
    irc "github.com/kofany/go-ircevo"
)

func startBot(server, channel string, wg *sync.WaitGroup) {
    defer wg.Done()
    
    conn := irc.IRC("multibot", "multibot")
    conn.RealName = "Multi-Server Bot"
    
    host := strings.Split(server, ":")[0]
    if strings.HasSuffix(server, ":6697") {
        conn.UseTLS = true
        conn.TLSConfig = &tls.Config{ServerName: host}
    }
    
    conn.AddCallback("001", func(e *irc.Event) {
        log.Printf("[%s] Connected", server)
        conn.Join(channel)
    })
    
    conn.AddCallback("PRIVMSG", func(e *irc.Event) {
        log.Printf("[%s] <%s> %s", server, e.Nick, e.Message())
    })
    
    if err := conn.Connect(server); err != nil {
        log.Printf("[%s] Failed to connect: %v", server, err)
        return
    }
    
    conn.Loop()
}

func main() {
    servers := map[string]string{
        "irc.libera.chat:6697": "#test",
        "irc.oftc.net:6697":    "#test",
    }
    
    var wg sync.WaitGroup
    
    for server, channel := range servers {
        wg.Add(1)
        go startBot(server, channel, &wg)
    }
    
    wg.Wait()
}
```

## Tor Connection

Connect through Tor:

```go
package main

import (
    "crypto/tls"
    "log"
    
    irc "github.com/kofany/go-ircevo"
)

func main() {
    conn := irc.IRC("torbot", "torbot")
    conn.RealName = "Tor Bot"
    
    // Configure Tor SOCKS5 proxy
    conn.ProxyConfig = &irc.ProxyConfig{
        Type:    "socks5",
        Address: "127.0.0.1:9050",
    }
    
    // Use TLS
    conn.UseTLS = true
    conn.TLSConfig = &tls.Config{
        ServerName: "irc.libera.chat",
    }
    
    conn.AddCallback("001", func(e *irc.Event) {
        log.Println("Connected via Tor!")
        conn.Join("#tor")
    })
    
    if err := conn.Connect("irc.libera.chat:6697"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

## Metrics Bot

Bot with Prometheus metrics:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    
    irc "github.com/kofany/go-ircevo"
)

var (
    messagesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "irc_messages_total",
            Help: "Total IRC messages by event type",
        },
        []string{"event"},
    )
    
    connectionsTotal = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "irc_connections_total",
            Help: "Total IRC connections",
        },
    )
)

func init() {
    prometheus.MustRegister(messagesTotal)
    prometheus.MustRegister(connectionsTotal)
}

func main() {
    // Start metrics server
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        log.Fatal(http.ListenAndServe(":8080", nil))
    }()
    
    conn := irc.IRC("metricsbot", "metricsbot")
    
    conn.AddCallback("*", func(e *irc.Event) {
        messagesTotal.WithLabelValues(e.Code).Inc()
    })
    
    conn.AddCallback("001", func(e *irc.Event) {
        connectionsTotal.Inc()
        conn.Join("#metrics")
    })
    
    if err := conn.Connect("irc.libera.chat:6667"); err != nil {
        log.Fatal(err)
    }
    
    conn.Loop()
}
```

---

For more examples, see the [examples/](../examples/) directory in the repository.
