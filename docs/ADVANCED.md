# Advanced Features

This guide covers advanced features for production deployments.

## Table of Contents

- [Authentication](#authentication)
- [TLS and Client Certificates](#tls-and-client-certificates)
- [Proxy Support](#proxy-support)
- [WebIRC](#webirc)
- [CAP Negotiation](#cap-negotiation)
- [Smart Error Handling](#smart-error-handling)
- [Reconnection Strategy](#reconnection-strategy)
- [Nick Management](#nick-management)
- [DCC Chat](#dcc-chat)
- [Custom Logging](#custom-logging)
- [Observability](#observability)

## Authentication

### SASL PLAIN

```go
conn.UseSASL = true
conn.SASLMech = "PLAIN"
conn.SASLLogin = "username"
conn.SASLPassword = "password"
```

### SASL EXTERNAL (Client Certificates)

```go
// Load client certificate
cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
if err != nil {
    log.Fatal(err)
}

conn.UseTLS = true
conn.TLSConfig = &tls.Config{
    Certificates: []tls.Certificate{cert},
}

conn.UseSASL = true
conn.SASLMech = "EXTERNAL"
conn.SASLLogin = "your-nick"
```

### NickServ Identification

```go
conn.AddCallback("001", func(e *irc.Event) {
    conn.Privmsg("NickServ", "IDENTIFY username password")
})
```

## TLS and Client Certificates

```go
conn.UseTLS = true
conn.TLSConfig = &tls.Config{
    MinVersion:         tls.VersionTLS12,
    InsecureSkipVerify: false,
    ServerName:         "irc.libera.chat",
}
```

### Custom Root CAs

```go
rootCAs := x509.NewCertPool()
certData, _ := os.ReadFile("ca.pem")
rootCAs.AppendCertsFromPEM(certData)

conn.TLSConfig = &tls.Config{
    RootCAs:    rootCAs,
    ServerName: "irc.example.net",
}
```

### Certificate Pinning

```go
conn.TLSConfig = &tls.Config{
    InsecureSkipVerify: true, // Verify manually
    VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
        // Implement your pinning logic here
        return nil
    },
}
```

## Proxy Support

### SOCKS5 Proxy (Tor)

```go
conn.ProxyConfig = &irc.ProxyConfig{
    Type:    "socks5",
    Address: "127.0.0.1:9050",
}

conn.UseTLS = true
conn.TLSConfig = &tls.Config{ServerName: "irc.libera.chat"}
```

### HTTP Proxy

```go
conn.ProxyConfig = &irc.ProxyConfig{
    Type:     "http",
    Address:  "proxy.example.com:8080",
    Username: "proxyuser",
    Password: "proxypass",
}
```

### Legacy Helper

```go
conn.SetProxy("socks5", "127.0.0.1:1080", "", "")
```

## WebIRC

If your network supports WebIRC:

```go
conn.WebIRC = "gateway-password:client-ip:client-hostname"
```

The library will automatically send `WEBIRC` before registering.

## CAP Negotiation

```go
conn.RequestCaps = []string{
    "multi-prefix",
    "sasl",
    "away-notify",
    "chghost",
    "message-tags",
}

// Use CAP 302 (IRCv3.2)
conn.CapVersion = "302"

// Delay registration until CAP END (recommended when using SASL)
conn.RegistrationAfterCapEnd = true
```

Handle capability responses:

```go
conn.AddCallback("CAP", func(e *irc.Event) {
    subcommand := e.Arguments[1]
    payload := ""
    if len(e.Arguments) >= 3 {
        payload = e.Arguments[2]
    }

    switch subcommand {
    case "LS":
        log.Printf("Server capabilities: %s", payload)
    case "ACK":
        log.Printf("Capabilities acknowledged: %s", payload)
    case "NAK":
        log.Printf("Capabilities rejected: %s", payload)
    }
})
```

## Smart Error Handling

Enable intelligent error categorization:

```go
conn.HandleErrorAsDisconnect = true
conn.SmartErrorHandling = true
```

Customize policies:

```go
conn.AddCallback("ERROR", func(e *irc.Event) {
    errorType = irc.AnalyzeErrorMessage(e.Message())
    switch errorType {
    case irc.PermanentError:
        log.Printf("Permanent ban: %s", e.Message())
    case irc.ServerError:
        log.Printf("Server issue: %s", e.Message())
    case irc.NetworkError:
        log.Printf("Network problem: %s", e.Message())
    case irc.RecoverableError:
        log.Printf("Recoverable: %s", e.Message())
    }
})
```

### Reconnection Limits

```go
conn.MaxRecoverableReconnects = 5
```

Set `0` for unlimited attempts.

## Reconnection Strategy

Implement exponential backoff:

```go
conn.AddCallback("ERROR", func(e *irc.Event) {
    go func() {
        for attempt := 0; attempt < 5; attempt++ {
            delay := time.Duration(1<<attempt) * time.Second
            time.Sleep(delay)
            if err := conn.Reconnect(); err == nil {
                return
            }
        }
        log.Println("Giving up after 5 attempts")
    }()
})
```

## Nick Management

### Tracking Desired vs Current Nick

```go
conn.AddCallback("NICK", func(e *irc.Event) {
    status := conn.GetNickStatus()
    if status.PendingChange {
        log.Printf("Nick change pending confirmation: %s -> %s", status.Current, status.Desired)
    }
})
```

### Handling Nick Errors

```go
conn.AddCallback("433", func(e *irc.Event) {
    // Nickname in use
    conn.Nick(conn.GetNick() + "_")
})

conn.AddCallback("432", func(e *irc.Event) {
    // Erroneous nickname
    conn.Nick("fallback")
})
```

## DCC Chat

### Accepting DCC CHAT Requests

```go
conn.DCCManager = irc.NewDCCManager()

conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    message := e.Message()
    if strings.HasPrefix(message, "\x01DCC CHAT") {
        // Library automatically handles incoming DCC chats
        log.Printf("Incoming DCC CHAT from %s", e.Nick)
    }
})
```

### Sending Messages

```go
if err := conn.SendDCCMessage("friend", "Hello over DCC!"); err != nil {
    log.Printf("DCC send failed: %v", err)
}
```

### Receiving Messages

```go
msg, err := conn.GetDCCMessage("friend")
if err != nil {
    if strings.Contains(err.Error(), "no message") {
        return
    }
    log.Printf("DCC receive error: %v", err)
}
log.Printf("DCC message from friend: %s", msg)
```

### Closing DCC Chats

```go
conn.CloseDCCChat("friend")
```

## Custom Logging

Use your own logger:

```go
logger := log.New(os.Stdout, "ircbot[", log.LstdFlags|log.Lshortfile)
conn.Log = logger
conn.Debug = true
```

## Observability

### Monitoring Connection Health

```go
if !conn.ValidateConnectionState() {
    log.Println("Connection unhealthy")
}
```

### Tracking Activity

```go
conn.AddCallback("*", func(e *irc.Event) {
    metrics.ObserveEvent(e.Code)
})
```

### Prometheus Integration

```go
var (
    messagesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "irc_messages_total"},
        []string{"code"},
    )
)

func init() {
    prometheus.MustRegister(messagesTotal)
}

conn.AddCallback("*", func(e *irc.Event) {
    messagesTotal.WithLabelValues(e.Code).Inc()
})
```

## Multiple Servers

Use separate connections per server:

```go
servers := []string{
    "irc.libera.chat:6697",
    "irc.oftc.net:6697",
}

var wg sync.WaitGroup

for _, server := range servers {
    wg.Add(1)
    go func(server string) {
        defer wg.Done()
        conn := irc.IRC("multibot", "bot")
        conn.UseTLS = true
        conn.TLSConfig = &tls.Config{ServerName: strings.Split(server, ":")[0]}
        conn.AddCallback("001", func(e *irc.Event) {
            conn.Join("#sharedchannel")
        })
        if err := conn.Connect(server); err != nil {
            log.Printf("Failed to connect to %s: %v", server, err)
            return
        }
        conn.Loop()
    }(server)
}

wg.Wait()
```

## Custom Event Context

Add contextual data to events:

```go
ctx := context.WithValue(context.Background(), "botID", "primary")
conn := irc.IRC("mybot", "botuser")
conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    botID := e.Ctx.Value("botID")
    log.Printf("[%s] <%s> %s", botID, e.Nick, e.Message())
})
```

## Integration with External Systems

### Slack Bridge Example

```go
slackIncoming := make(chan slackMessage)
slackOutgoing := make(chan slackMessage)

conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    slackOutgoing <- slackMessage{
        Channel: e.Arguments[0],
        User:    e.Nick,
        Text:    e.Message(),
    }
})

go func() {
    for msg := range slackIncoming {
        conn.Privmsg(msg.Channel, fmt.Sprintf("<%s> %s", msg.User, msg.Text))
    }
}()
```

## Debugging Tips

- Set `conn.Debug = true` to log all traffic
- Use `conn.VerboseCallbackHandler = true` to trace callback execution
- Run with `GODEBUG=http2client=0` if using proxies with TLS issues
- Collect `conn.ErrorChan()` output for diagnostics

## Hardening and Security

- Always use TLS for internet-facing deployments
- Use SASL for authenticated networks
- Restrict `QuitMessage` to avoid leaking internal data
- Use dedicated proxy credentials per bot instance
- Monitor `MaxRecoverableReconnects` to avoid infinite loops
- Validate incoming CTCP and DCC requests before accepting

---

See [Examples](EXAMPLES.md) for real-world code and [Troubleshooting](TROUBLESHOOTING.md) for common issues.
