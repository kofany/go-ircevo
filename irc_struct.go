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

	socket net.Conn
	pwrite chan string
	end    chan struct{}

	nick        string // The nickname we want.
	nickcurrent string // The nickname we currently have.
	user        string
	registered  bool
	events      map[string]map[int]func(*Event)
	eventsMutex sync.Mutex

	QuitMessage      string
	lastMessage      time.Time
	lastMessageMutex sync.Mutex

	VerboseCallbackHandler bool
	Log                    *log.Logger

	stopped bool
	quit    bool // User called Quit, do not reconnect.

	idCounter int // Assign unique IDs to callbacks

	// New fields added for binding to a specific local IP and connection status
	localIP        string      // Local IP to bind when connecting
	fullyConnected bool        // Indicates if the connection is fully established
	DCCManager     *DCCManager // DCC chat support

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
