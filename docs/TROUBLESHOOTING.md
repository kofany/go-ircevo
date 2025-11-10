# Troubleshooting

Common issues and their solutions.

## Table of Contents

- [Connection Issues](#connection-issues)
- [Authentication Problems](#authentication-problems)
- [Nickname Issues](#nickname-issues)
- [TLS/SSL Problems](#tlsssl-problems)
- [Proxy Issues](#proxy-issues)
- [Reconnection Problems](#reconnection-problems)
- [Performance Issues](#performance-issues)
- [DCC Issues](#dcc-issues)

## Connection Issues

### Cannot Connect to Server

**Problem:** Connection fails immediately.

**Solutions:**

1. Check server address format:
   ```go
   conn.Connect("irc.example.com:6667")  // Correct
   conn.Connect("irc.example.com")       // Wrong - missing port
   ```

2. Verify the port is correct:
   - Plain text: Usually 6667
   - TLS/SSL: Usually 6697

3. Test connectivity:
   ```bash
   telnet irc.libera.chat 6667
   # or for TLS:
   openssl s_client -connect irc.libera.chat:6697
   ```

4. Check firewall rules

### Connection Times Out

**Problem:** `Connect()` hangs or times out.

**Solutions:**

1. Increase timeout:
   ```go
   conn.Timeout = 5 * time.Minute
   ```

2. Check network connectivity

3. Try a different server

4. Disable timeout fallback:
   ```go
   conn.EnableTimeoutFallback = false
   ```

### Disconnects Immediately After Connecting

**Problem:** Bot connects but disconnects within seconds.

**Solutions:**

1. Check server ERROR message:
   ```go
   conn.AddCallback("ERROR", func(e *irc.Event) {
       log.Printf("Server ERROR: %s", e.Message())
   })
   ```

2. Common causes:
   - Banned (K-lined): Wait or use different IP/nick
   - Too many connections: Reduce connection rate
   - Bad credentials: Check password/SASL settings
   - Proxy detected: Use clean IP or VPN

### Ghost Bot / Can't Reconnect

**Problem:** Old connection still active, new connection rejected with "Nick already in use".

**Solutions:**

1. Send GHOST command to NickServ:
   ```go
   conn.AddCallback("433", func(e *irc.Event) {
       conn.Privmsg("NickServ", "GHOST "+conn.GetNick()+" password")
       time.Sleep(2 * time.Second)
       conn.Nick(conn.GetNick())
   })
   ```

2. Use different nick initially, then change:
   ```go
   conn.AddCallback("001", func(e *irc.Event) {
       conn.Privmsg("NickServ", "GHOST originalnick password")
       time.Sleep(2 * time.Second)
       conn.Nick("originalnick")
   })
   ```

## Authentication Problems

### SASL Authentication Fails

**Problem:** `901`, `902`, or `904` numeric received.

**Solutions:**

1. Verify credentials:
   ```go
   conn.UseSASL = true
   conn.SASLLogin = "account"      // Account name, not nick
   conn.SASLPassword = "password"
   conn.SASLMech = "PLAIN"
   ```

2. Check mechanism support:
   ```go
   conn.AddCallback("CAP", func(e *irc.Event) {
       if e.Arguments[1] == "LS" {
           log.Printf("Server capabilities: %s", e.Arguments[2])
       }
   })
   ```

3. Ensure proper CAP ordering:
   ```go
   conn.RegistrationAfterCapEnd = true
   ```

### Server Password Rejected

**Problem:** `464` numeric (bad password).

**Solutions:**

1. Verify password is set correctly:
   ```go
   conn.Password = "serverpass"
   ```

2. Note: `conn.Password` is for server password, not NickServ

3. For NickServ identification, use callback:
   ```go
   conn.AddCallback("001", func(e *irc.Event) {
       conn.Privmsg("NickServ", "IDENTIFY password")
   })
   ```

## Nickname Issues

### Nick Already in Use

**Problem:** `433` numeric on connection.

**Solutions:**

1. Handle automatically:
   ```go
   conn.AddCallback("433", func(e *irc.Event) {
       conn.Nick(conn.GetNick() + "_")
   })
   ```

2. Use GHOST (see above)

3. Use unique nick per instance:
   ```go
   nick := fmt.Sprintf("bot%d", rand.Intn(9999))
   conn := irc.IRC(nick, "bot")
   ```

### Nick Desynchronization

**Problem:** `GetNick()` returns wrong nickname.

**Solutions:**

1. The library auto-corrects this. If issues persist, file a bug.

2. Use `GetNickStatus()` to inspect state:
   ```go
   status := conn.GetNickStatus()
   log.Printf("Current: %s, Desired: %s, Confirmed: %v",
       status.Current, status.Desired, status.Confirmed)
   ```

### Erroneous Nickname

**Problem:** `432` numeric (invalid characters in nick).

**Solutions:**

1. Use valid characters only (letters, numbers, `[]{}|_^`)

2. Don't start with a number

3. Keep length under 30 characters

## TLS/SSL Problems

### Certificate Verification Fails

**Problem:** TLS handshake fails with certificate error.

**Solutions:**

1. Temporary workaround (not recommended for production):
   ```go
   conn.TLSConfig = &tls.Config{
       InsecureSkipVerify: true,
   }
   ```

2. Specify correct server name:
   ```go
   conn.TLSConfig = &tls.Config{
       ServerName: "irc.libera.chat",
   }
   ```

3. Update system CA certificates:
   ```bash
   # Ubuntu/Debian
   sudo update-ca-certificates
   
   # macOS
   # Usually automatic
   ```

### Client Certificate Not Working

**Problem:** SASL EXTERNAL fails.

**Solutions:**

1. Verify certificate loading:
   ```go
   cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
   if err != nil {
       log.Fatal(err)
   }
   
   conn.TLSConfig = &tls.Config{
       Certificates: []tls.Certificate{cert},
   }
   ```

2. Check certificate format (PEM)

3. Ensure certificate is registered with network

## Proxy Issues

### SOCKS5 Proxy Connection Fails

**Problem:** Cannot connect through Tor or other SOCKS proxy.

**Solutions:**

1. Verify proxy is running:
   ```bash
   curl --socks5 127.0.0.1:9050 https://check.torproject.org
   ```

2. Check proxy config:
   ```go
   conn.ProxyConfig = &irc.ProxyConfig{
       Type:    "socks5",
       Address: "127.0.0.1:9050",
   }
   ```

3. For Tor, ensure tor service is running:
   ```bash
   systemctl status tor
   ```

### Proxy Authentication Failed

**Problem:** Proxy rejects credentials.

**Solutions:**

1. Include credentials:
   ```go
   conn.ProxyConfig = &irc.ProxyConfig{
       Type:     "socks5",
       Address:  "proxy.example.com:1080",
       Username: "user",
       Password: "pass",
   }
   ```

2. Try without credentials if not required

## Reconnection Problems

### Infinite Reconnection Loop

**Problem:** Bot reconnects endlessly.

**Solutions:**

1. Enable smart error handling:
   ```go
   conn.SmartErrorHandling = true
   conn.HandleErrorAsDisconnect = true
   ```

2. Limit reconnection attempts:
   ```go
   conn.MaxRecoverableReconnects = 3
   ```

3. Check ERROR messages:
   ```go
   conn.AddCallback("ERROR", func(e *irc.Event) {
       errorType := irc.AnalyzeErrorMessage(e.Message())
       log.Printf("ERROR type: %s, message: %s", errorType, e.Message())
       
       if errorType == irc.PermanentError {
           log.Println("Permanent ban, exiting")
           os.Exit(1)
       }
   })
   ```

### Won't Reconnect After Disconnect

**Problem:** Bot doesn't reconnect automatically.

**Solutions:**

1. Ensure `Loop()` is being used:
   ```go
   conn.Loop()  // This handles reconnection
   ```

2. Check quit state:
   ```go
   // Don't call Quit() if you want reconnection
   // Use Disconnect() instead
   ```

3. Enable reconnection features:
   ```go
   conn.HandleErrorAsDisconnect = true
   ```

## Performance Issues

### High CPU Usage

**Problem:** Bot uses excessive CPU.

**Solutions:**

1. Check for tight loops in callbacks:
   ```go
   conn.AddCallback("PRIVMSG", func(e *irc.Event) {
       // BAD: Tight loop
       for {
           // ...
       }
       
       // GOOD: Spawn goroutine for long work
       go processMessage(e.Message())
   })
   ```

2. Disable verbose logging in production:
   ```go
   conn.Debug = false
   conn.VerboseCallbackHandler = false
   ```

### High Memory Usage

**Problem:** Memory grows over time.

**Solutions:**

1. Clear old callbacks:
   ```go
   conn.ClearCallback("PRIVMSG")
   ```

2. Close old DCC chats:
   ```go
   for _, nick := range conn.ListActiveDCCChats() {
       conn.CloseDCCChat(nick)
   }
   ```

3. Don't leak goroutines:
   ```go
   // BAD
   conn.AddCallback("PRIVMSG", func(e *irc.Event) {
       go func() {
           // No cleanup, leaked goroutine
           time.Sleep(999 * time.Hour)
       }()
   })
   ```

### Slow Message Delivery

**Problem:** Messages delayed or queued.

**Solutions:**

1. Don't block in callbacks:
   ```go
   conn.AddCallback("PRIVMSG", func(e *irc.Event) {
       go func() {
           // Long operation in goroutine
           result := expensiveOperation()
           conn.Privmsg(e.Arguments[0], result)
       }()
   })
   ```

2. Increase buffer sizes (advanced - requires fork)

## DCC Issues

### DCC CHAT Won't Connect

**Problem:** Incoming or outgoing DCC fails.

**Solutions:**

1. Ensure DCC manager is initialized:
   ```go
   conn.DCCManager = irc.NewDCCManager()
   ```

2. Check firewall allows incoming connections

3. For NAT/firewall issues, DCC may not work without port forwarding

### Can't Send DCC Messages

**Problem:** `SendDCCMessage` fails.

**Solutions:**

1. Verify chat is active:
   ```go
   if !conn.IsDCCChatActive(nick) {
       log.Printf("No active DCC with %s", nick)
   }
   ```

2. Check for errors:
   ```go
   if err := conn.SendDCCMessage(nick, "Hello"); err != nil {
       log.Printf("Send failed: %v", err)
   }
   ```

## Debugging Tips

### Enable Debug Logging

```go
conn.Debug = true
conn.VerboseCallbackHandler = true
```

### Capture Raw Traffic

```go
conn.AddCallback("*", func(e *irc.Event) {
    log.Printf("[RAW] %s: %s", e.Code, e.Raw)
})
```

### Monitor Connection State

```go
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        healthy := conn.ValidateConnectionState()
        connected := conn.Connected()
        nick := conn.GetNick()
        log.Printf("Health: %v, Connected: %v, Nick: %s", healthy, connected, nick)
    }
}()
```

### Check goroutine leaks

```bash
# In your code
import _ "net/http/pprof"
go http.ListenAndServe("localhost:6060", nil)

# Then browse to:
# http://localhost:6060/debug/pprof/goroutine?debug=1
```

## Getting Help

If you can't resolve your issue:

1. Check the [examples/](../examples/) for working code
2. Review [API documentation](API.md)
3. Search [GitHub Issues](https://github.com/kofany/go-ircevo/issues)
4. File a new issue with:
   - Go version
   - Library version
   - Minimal reproducible example
   - Debug logs (sanitize credentials!)
   - Expected vs actual behavior
