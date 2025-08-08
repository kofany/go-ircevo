// Copyright (c) 2024 Jerzy Dąbrowski
// Based on original work by Thomas Jager, 2009. All rights reserved.
//
// This project is a fork of the original go-ircevent library created by Thomas Jager.
// Redistribution and use in source and binary forms, with or without modification, are permitted provided
// that the following conditions are met:
//
//    - Redistributions of source code must retain the above copyright notice, this list of conditions,
//      and the following disclaimer.
//    - Redistributions in binary form must reproduce the above copyright notice, this list of conditions,
//      and the following disclaimer in the documentation and/or other materials provided with the distribution.
//    - Neither the name of the original authors nor the names of its contributors may be used to endorse
//      or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED "AS IS" WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT
// LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE COPYRIGHT HOLDERS OR CONTRIBUTORS BE LIABLE FOR ANY CLAIM, DAMAGES, OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT, OR OTHERWISE, ARISING FROM, OUT OF, OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package irc

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"regexp"
	"sync"
	"time"

	"golang.org/x/text/encoding"
)

// Connection represents an IRC connection.
type Connection struct {
	sync.Mutex
	sync.WaitGroup
	Debug            bool
	Error            chan error
	WebIRC           string
	Password         string
	UseTLS           bool
	UseSASL          bool
	RequestCaps      []string
	AcknowledgedCaps []string
	SASLLogin        string
	SASLPassword     string
	SASLMech         string
	TLSConfig        *tls.Config
	Version          string
	Timeout          time.Duration
	CallbackTimeout  time.Duration
	PingFreq         time.Duration
	KeepAlive        time.Duration
	Server           string
	Encoding         encoding.Encoding
	ProxyConfig      *ProxyConfig

	RealName string // The real name we want to display.
	// If zero-value defaults to the user.

	socket                 net.Conn
	pwrite                 chan string
	end                    chan struct{}
	nick                   string // The nickname we want.
	nickcurrent            string // The nickname we currently have.
	user                   string
	events                 map[string]map[int]func(*Event)
	eventsMutex            sync.Mutex
	QuitMessage            string
	lastMessage            time.Time
	lastMessageMutex       sync.Mutex
	VerboseCallbackHandler bool
	Log                    *log.Logger
	stopped                bool
	quit                   bool
	idCounter              int
	localIP                string        // Local IP to bind when connecting
	fullyConnected         bool          // Indicates if the connection is fully established
	lastNickChange         time.Time     // Timestamp of the last nickname change
	nickError              string        // Last error related to nickname
	registrationSteps      int           // Counter for registration steps
	registrationStartTime  time.Time     // Time when registration started
	registrationTimeout    time.Duration // Timeout for registration process

	// NEW: Configuration for timeout fallback behavior
	EnableTimeoutFallback bool // Allow timeout-based connection detection (default: false)

	// NEW: NICK change coordination to prevent race conditions
	nickChangeInProgress bool      // Indicates if a NICK change is currently in progress
	nickChangeTimeout    time.Time // Timeout for NICK change operations

	// NEW: CAP negotiation behavior
	CapVersion              string    // e.g., "302" or "" for plain CAP LS
	RegistrationAfterCapEnd bool      // When true, send NICK/USER only after CAP END
	Respect020Pacing        bool      // When true, add small backoff after numeric 020
	got020                  bool      // internal: have we seen numeric 020
	last020                 time.Time // internal: last time 020 was received
	sentRegistration        bool      // internal: have we sent NICK/USER yet

	DCCManager              *DCCManager // DCC chat support
	HandleErrorAsDisconnect bool        // Fix reconnection loop after ERROR event if user have own reconnect implementation

	// NEW: Smart ERROR handling - analyze ERROR messages to determine if reconnect should be attempted
	SmartErrorHandling bool // Enable intelligent ERROR message analysis (default: true)

	// NEW: Limit the number of reconnection attempts after a RecoverableError
	// 0 means unlimited attempts (default). Set to a positive value to cap retries.
	MaxRecoverableReconnects int

	// internal counter for recoverable reconnect attempts within current session
	recoverableReconnects int
}

// ErrorType represents different categories of IRC ERROR messages
type ErrorType int

const (
	// RecoverableError - temporary issues that should allow reconnection
	RecoverableError ErrorType = iota
	// PermanentError - permanent bans/blocks that should prevent reconnection
	PermanentError
	// ServerError - server-side issues (too many connections, etc.)
	ServerError
	// NetworkError - network connectivity issues
	NetworkError
)

// String returns a string representation of the ErrorType
func (e ErrorType) String() string {
	switch e {
	case RecoverableError:
		return "RecoverableError"
	case PermanentError:
		return "PermanentError"
	case ServerError:
		return "ServerError"
	case NetworkError:
		return "NetworkError"
	default:
		return "UnknownError"
	}
}

type ProxyConfig struct {
	Type     string // "socks5", "http", etc....
	Address  string
	Username string
	Password string
}

// Event represents an IRC event.
type Event struct {
	Code       string
	Raw        string
	Nick       string //<nick>
	Host       string //<nick>!<usr>@<host>
	Source     string //<host>
	User       string //<usr>
	Arguments  []string
	Tags       map[string]string
	Connection *Connection
	Ctx        context.Context
}

// Message retrieves the last message from Event arguments.
// This function leaves the arguments untouched and
// returns an empty string if there are none.
func (e *Event) Message() string {
	if len(e.Arguments) == 0 {
		return ""
	}
	return e.Arguments[len(e.Arguments)-1]
}

// ircFormat is a regex for IRC formatting codes.
var ircFormat = regexp.MustCompile(`[\x02\x1F\x0F\x16\x1D\x1E]|\x03(\d\d?(,\d\d?)?)?`)

// MessageWithoutFormat retrieves the last message from Event arguments,
// but without IRC formatting (e.g., colors).
// This function leaves the arguments untouched and
// returns an empty string if there are none.
func (e *Event) MessageWithoutFormat() string {
	if len(e.Arguments) == 0 {
		return ""
	}
	return ircFormat.ReplaceAllString(e.Arguments[len(e.Arguments)-1], "")
}

// NickStatus represents the current status of a nickname in the IRC connection.
// It provides detailed information about the nickname state, including whether
// it has been confirmed by the server and any pending changes.
//
// According to RFC 2812 section 3.1.2, a nickname change is only confirmed
// when the server sends a NICK message in the format:
// :OLD_NICK!user@host NICK NEW_NICK
type NickStatus struct {
	// Current is the nickname currently in use according to the server.
	// This is the nickname that has been confirmed by the server.
	Current string

	// Desired is the nickname that the user wants to use.
	// This is the nickname that was requested with Nick().
	Desired string

	// Confirmed indicates whether the server has confirmed the current nickname.
	// This is true after receiving the 001 welcome message or a successful NICK change.
	Confirmed bool

	// LastChangeTime is the timestamp of the last nickname change attempt.
	LastChangeTime time.Time

	// PendingChange indicates if there's a nickname change in progress.
	// This is true when Current and Desired are different.
	PendingChange bool

	// Error contains any error related to the nickname (e.g., already in use).
	// This is set when the server rejects a nickname change.
	Error string
}
