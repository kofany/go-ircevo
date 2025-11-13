# API Reference

Complete API reference for go-ircevo.

## Table of Contents

- [Connection Creation](#connection-creation)
- [Connection Methods](#connection-methods)
- [Event Handling](#event-handling)
- [Nick Management](#nick-management)
- [Channel Operations](#channel-operations)
- [Messaging](#messaging)
- [Connection Control](#connection-control)
- [Configuration](#configuration)
- [DCC Chat](#dcc-chat)
- [Types](#types)

## Connection Creation

### IRC

```go
func IRC(nick, user string) *Connection
```

Creates a new IRC connection with the specified nickname and username.

**Parameters:**
- `nick` - The desired nickname
- `user` - The username (ident)

**Returns:** A new `*Connection` instance

**Example:**
```go
conn := irc.IRC("mybot", "botuser")
```

## Connection Methods

### Connect

```go
func (irc *Connection) Connect(server string) error
```

Connects to an IRC server.

**Parameters:**
- `server` - Server address in format `hostname:port`

**Returns:** `error` if connection fails

**Example:**
```go
err := conn.Connect("irc.libera.chat:6667")
if err != nil {
    log.Fatal(err)
}
```

### Loop

```go
func (irc *Connection) Loop()
```

Starts the main event loop. Blocks until the connection is closed. Handles automatic reconnection.

**Example:**
```go
conn.Loop()
```

### Reconnect

```go
func (irc *Connection) Reconnect() error
```

Reconnects to the IRC server after a disconnection.

**Returns:** `error` if reconnection fails

### Disconnect

```go
func (irc *Connection) Disconnect()
```

Disconnects from the IRC server without sending QUIT.

### Quit

```go
func (irc *Connection) Quit()
```

Sends QUIT message and disconnects gracefully.

**Example:**
```go
conn.QuitMessage = "Goodbye!"
conn.Quit()
```

### Connected

```go
func (irc *Connection) Connected() bool
```

Checks if the connection is fully established and registered.

**Returns:** `true` if fully connected

### IsFullyConnected

```go
func (irc *Connection) IsFullyConnected() bool
```

Checks if connection registration is complete (received 001).

**Returns:** `true` if registration complete

### ValidateConnectionState

```go
func (irc *Connection) ValidateConnectionState() bool
```

Performs comprehensive connection health validation including socket state and activity monitoring.

**Returns:** `true` if connection is healthy

**Example:**
```go
if !conn.ValidateConnectionState() {
    log.Println("Connection unhealthy, reconnecting...")
    conn.Reconnect()
}
```

## Event Handling

### AddCallback

```go
func (irc *Connection) AddCallback(eventcode string, callback func(*Event)) int
```

Registers a callback function for a specific event code.

**Parameters:**
- `eventcode` - IRC command or numeric (e.g., "PRIVMSG", "001", "*" for all events)
- `callback` - Function to call when event occurs

**Returns:** Callback ID for later removal

**Example:**
```go
id := conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    log.Printf("<%s> %s", e.Nick, e.Message())
})
```

### RemoveCallback

```go
func (irc *Connection) RemoveCallback(eventcode string, id int) bool
```

Removes a specific callback by ID.

**Parameters:**
- `eventcode` - Event code the callback was registered for
- `id` - Callback ID returned by `AddCallback`

**Returns:** `true` if callback was removed

### ClearCallback

```go
func (irc *Connection) ClearCallback(eventcode string) bool
```

Removes all callbacks for a specific event code.

**Returns:** `true` if event code was found

### ReplaceCallback

```go
func (irc *Connection) ReplaceCallback(eventcode string, id int, callback func(*Event))
```

Replaces an existing callback with a new one.

## Nick Management

### Nick

```go
func (irc *Connection) Nick(n string)
```

Changes or sets the nickname. Thread-safe with atomic operations.

**Parameters:**
- `n` - New nickname

**Example:**
```go
conn.Nick("newnicknametouse")
```

### GetNick

```go
func (irc *Connection) GetNick() string
```

Returns the current confirmed nickname.

**Returns:** Current nickname

### GetNickStatus

```go
func (irc *Connection) GetNickStatus() *NickStatus
```

Returns detailed information about nickname state.

**Returns:** `*NickStatus` with current, desired, and confirmation status

**Example:**
```go
status := conn.GetNickStatus()
if !status.Confirmed {
    log.Printf("Nick change pending: %s -> %s", status.Current, status.Desired)
}
```

## Channel Operations

### Join

```go
func (irc *Connection) Join(channel string)
```

Joins a channel.

**Parameters:**
- `channel` - Channel name (with # prefix)

**Example:**
```go
conn.Join("#golang")
```

### Part

```go
func (irc *Connection) Part(channel string)
```

Leaves a channel.

**Parameters:**
- `channel` - Channel name

**Example:**
```go
conn.Part("#golang")
```

### Kick

```go
func (irc *Connection) Kick(user, channel, msg string)
```

Kicks a user from a channel.

**Parameters:**
- `user` - Nickname to kick
- `channel` - Channel name
- `msg` - Kick message

**Example:**
```go
conn.Kick("baduser", "#mychannel", "Spamming")
```

### MultiKick

```go
func (irc *Connection) MultiKick(users []string, channel string, msg string)
```

Kicks multiple users from a channel.

**Parameters:**
- `users` - Slice of nicknames to kick
- `channel` - Channel name
- `msg` - Kick message

### Mode

```go
func (irc *Connection) Mode(target string, modestring ...string)
```

Sets channel or user modes.

**Parameters:**
- `target` - Channel or nickname
- `modestring` - Mode changes

**Example:**
```go
conn.Mode("#mychannel", "+o", "username")
conn.Mode("#mychannel", "+m")
```

## Messaging

### Privmsg

```go
func (irc *Connection) Privmsg(target, message string)
```

Sends a message to a channel or user.

**Parameters:**
- `target` - Channel name or nickname
- `message` - Message text

**Example:**
```go
conn.Privmsg("#channel", "Hello, world!")
conn.Privmsg("username", "Private message")
```

### Privmsgf

```go
func (irc *Connection) Privmsgf(target, format string, a ...interface{})
```

Sends a formatted message.

**Example:**
```go
conn.Privmsgf("#channel", "User %s has %d points", nick, points)
```

### Notice

```go
func (irc *Connection) Notice(target, message string)
```

Sends a notice to a channel or user. Notices should not trigger automated responses.

**Example:**
```go
conn.Notice("#channel", "Bot maintenance in 5 minutes")
```

### Noticef

```go
func (irc *Connection) Noticef(target, format string, a ...interface{})
```

Sends a formatted notice.

### Action

```go
func (irc *Connection) Action(target, message string)
```

Sends a CTCP ACTION (/me message).

**Example:**
```go
conn.Action("#channel", "waves hello")
// Displays: * botname waves hello
```

### Actionf

```go
func (irc *Connection) Actionf(target, format string, a ...interface{})
```

Sends a formatted action.

### SendRaw

```go
func (irc *Connection) SendRaw(message string)
```

Sends a raw IRC command.

**Parameters:**
- `message` - Raw IRC command

**Example:**
```go
conn.SendRaw("NAMES #channel")
```

### SendRawf

```go
func (irc *Connection) SendRawf(format string, a ...interface{})
```

Sends a formatted raw IRC command.

**Example:**
```go
conn.SendRawf("PRIVMSG %s :%s", target, message)
```

## Connection Control

### Who

```go
func (irc *Connection) Who(nick string)
```

Sends WHO query for a user.

### Whois

```go
func (irc *Connection) Whois(nick string)
```

Sends WHOIS query for a user.

### SetLocalIP

```go
func (irc *Connection) SetLocalIP(ip string)
```

Sets the local IP address to bind to when connecting.

**Parameters:**
- `ip` - Local IP address

**Example:**
```go
conn.SetLocalIP("192.168.1.100")
```

### SetProxy

```go
func (irc *Connection) SetProxy(proxyType, address, username, password string)
```

Configures proxy settings (legacy method, prefer using `ProxyConfig`).

**Parameters:**
- `proxyType` - "socks5" or "http"
- `address` - Proxy address (host:port)
- `username` - Proxy username (optional)
- `password` - Proxy password (optional)

### ErrorChan

```go
func (irc *Connection) ErrorChan() chan error
```

Returns the error channel for monitoring connection errors.

**Returns:** `chan error`

## DCC Chat

### InitiateDCCChat

```go
func (irc *Connection) InitiateDCCChat(target string) error
```

Initiates a DCC CHAT session with a user.

**Parameters:**
- `target` - Nickname to chat with

**Returns:** `error` if setup fails

### SendDCCMessage

```go
func (irc *Connection) SendDCCMessage(nick, message string) error
```

Sends a message over an active DCC CHAT connection.

**Parameters:**
- `nick` - Nickname of the DCC chat partner
- `message` - Message to send

**Returns:** `error` if no active chat or send fails

### GetDCCMessage

```go
func (irc *Connection) GetDCCMessage(nick string) (string, error)
```

Retrieves a message from a DCC CHAT connection.

**Parameters:**
- `nick` - Nickname of the DCC chat partner

**Returns:** Message and error

### IsDCCChatActive

```go
func (irc *Connection) IsDCCChatActive(nick string) bool
```

Checks if there's an active DCC CHAT with a user.

**Returns:** `true` if chat is active

### ListActiveDCCChats

```go
func (irc *Connection) ListActiveDCCChats() []string
```

Returns a list of all active DCC CHAT nicknames.

**Returns:** Slice of nicknames

### CloseDCCChat

```go
func (irc *Connection) CloseDCCChat(nick string) error
```

Closes a DCC CHAT connection.

**Returns:** `error` if chat not found

### SetDCCChatTimeout

```go
func (irc *Connection) SetDCCChatTimeout(timeout time.Duration)
```

Sets timeout for all active DCC CHAT connections.

## Configuration

### Connection Struct Fields

```go
type Connection struct {
    // Basic settings
    Debug            bool              // Enable debug logging
    Password         string            // Server password
    RealName         string            // Real name (GECOS)
    Version          string            // CTCP VERSION response
    
    // TLS/SSL settings
    UseTLS           bool              // Enable TLS
    TLSConfig        *tls.Config       // TLS configuration
    
    // SASL authentication
    UseSASL          bool              // Enable SASL
    SASLLogin        string            // SASL username
    SASLPassword     string            // SASL password
    SASLMech         string            // SASL mechanism ("PLAIN" or "EXTERNAL")
    
    // Timing
    Timeout          time.Duration     // Connection timeout (default: 1 minute)
    CallbackTimeout  time.Duration     // Callback execution timeout
    PingFreq         time.Duration     // PING frequency (default: 15 minutes)
    KeepAlive        time.Duration     // Keep-alive timeout (default: 4 minutes)
    
    // Error handling
    SmartErrorHandling       bool      // Enable intelligent error analysis
    HandleErrorAsDisconnect  bool      // Treat ERROR as disconnect
    MaxRecoverableReconnects int       // Limit reconnection attempts (0 = unlimited)
    EnableTimeoutFallback    bool      // Enable timeout-based detection (default: false)
    
    // Proxy
    ProxyConfig      *ProxyConfig      // Proxy configuration
    
    // WebIRC
    WebIRC           string            // WebIRC password/configuration
    
    // IRCv3
    RequestCaps      []string          // Capabilities to request
    AcknowledgedCaps []string          // Server-acknowledged capabilities
    CapVersion       string            // CAP version ("302" for CAP v3.2)
    
    // Behavior
    QuitMessage              string    // Custom quit message
    VerboseCallbackHandler   bool      // Verbose callback logging
    RegistrationAfterCapEnd  bool      // Send NICK/USER after CAP END
    Respect020Pacing         bool      // Add delay after numeric 020
    
    // DCC
    DCCManager       *DCCManager       // DCC chat manager
}
```

### ProxyConfig Struct

```go
type ProxyConfig struct {
    Type     string  // "socks5", "http"
    Address  string  // "host:port"
    Username string  // Optional
    Password string  // Optional
}
```

## Types

### Event

```go
type Event struct {
    Code       string            // Event code (e.g., "PRIVMSG", "001")
    Raw        string            // Raw IRC message
    Nick       string            // Sender's nickname
    Host       string            // Full host (nick!user@host)
    Source     string            // Source hostname
    User       string            // Sender's username
    Arguments  []string          // Event arguments
    Tags       map[string]string // IRCv3 message tags
    Connection *Connection       // Reference to connection
    Ctx        context.Context   // Context for the event
}
```

#### Event Methods

```go
func (e *Event) Message() string
```

Returns the last argument (typically the message text).

```go
func (e *Event) MessageWithoutFormat() string
```

Returns the message with IRC formatting codes removed.

### NickStatus

```go
type NickStatus struct {
    Current        string        // Currently confirmed nickname
    Desired        string        // Desired nickname
    Confirmed      bool          // Whether current nick is confirmed
    LastChangeTime time.Time     // Timestamp of last change
    PendingChange  bool          // Whether a change is pending
    Error          string        // Last nick-related error
}
```

### ErrorType

```go
type ErrorType int

const (
    RecoverableError ErrorType = iota  // Temporary, allow reconnect
    PermanentError                      // Permanent ban, block reconnect
    ServerError                         // Server issues, retry with delay
    NetworkError                        // Network issues, immediate retry
)
```

#### AnalyzeErrorMessage

```go
func AnalyzeErrorMessage(errorMsg string) ErrorType
```

Analyzes an IRC ERROR message and categorizes it.

**Parameters:**
- `errorMsg` - The ERROR message text

**Returns:** `ErrorType` classification

**Example:**
```go
errorType := irc.AnalyzeErrorMessage("Closing Link: banned")
if errorType == irc.PermanentError {
    log.Println("Permanently banned, not reconnecting")
}
```

### DCCManager

```go
type DCCManager struct {
    // Internal fields (managed automatically)
}

func NewDCCManager() *DCCManager
```

Creates a new DCC manager for handling DCC CHAT connections.

### CallbackID

```go
type CallbackID struct {
    EventCode string
    ID        int
}
```

Identifies a specific callback for management.

## Constants

```go
const VERSION = "go-ircevo v1.2.3"
```

Library version string.

```go
const CAP_TIMEOUT = time.Second * 15
```

Timeout for CAP negotiation.

## Error Values

```go
var ErrDisconnected = errors.New("Disconnect Called")
```

Error returned when disconnect is intentional.

## Thread Safety

The following methods are thread-safe:

- `Nick()` - Uses mutex for atomic nickname operations
- `GetNick()` - Safe concurrent reads
- `GetNickStatus()` - Safe concurrent reads
- `AddCallback()`, `RemoveCallback()`, `ClearCallback()` - Event callback management
- `SendRaw()`, `SendRawf()` - Message sending
- All DCC methods - Protected by DCCManager mutex
- All channel/messaging methods

## Callback Execution

Callbacks are executed synchronously in the order they were registered. Long-running callbacks may block other event processing. For long operations, spawn a goroutine:

```go
conn.AddCallback("PRIVMSG", func(e *irc.Event) {
    go func() {
        // Long-running operation
        processMessage(e.Message())
    }()
})
```

## Best Practices

1. **Always handle numeric 001**: This indicates successful connection
2. **Use GetNick() not internal state**: The confirmed nick may differ from desired
3. **Enable SmartErrorHandling**: Better behavior for production deployments
4. **Set MaxRecoverableReconnects**: Prevent infinite reconnection loops
5. **Use thread-safe methods**: Don't access internal fields directly
6. **Spawn goroutines for long operations**: Keep callbacks fast
7. **Handle ERROR events**: Know when to stop reconnecting
