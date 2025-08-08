// Copyright (c) 2024 Jerzy Dąbrowski
// Based on original work by Thomas Jager, 2009. All rights reserved.
//
// This project is a fork of the original go-ircevent library created by Thomas Jager.
// Redistribution and use in source and binary forms, with or without modification, are permitted provided
// that the following conditions are met:
//
//   - Redistributions of source code must retain the above copyright notice, this list of conditions,
//     and the following disclaimer.
//   - Redistributions in binary form must reproduce the above copyright notice, this list of conditions,
//     and the following disclaimer in the documentation and/or other materials provided with the distribution.
//   - Neither the name of the original authors nor the names of its contributors may be used to endorse
//     or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED "AS IS" WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT
// LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE COPYRIGHT HOLDERS OR CONTRIBUTORS BE LIABLE FOR ANY CLAIM, DAMAGES, OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT, OR OTHERWISE, ARISING FROM, OUT OF, OR IN CONNECTION WITH THE SOFTWARE
// OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
package irc

import (
	"context"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// CallbackID is a tuple type for uniquely identifying callbacks.
type CallbackID struct {
	EventCode string
	ID        int
}

// AddCallback registers a callback to a connection and event code.
// A callback is a function which takes only an Event pointer as a parameter.
// Valid event codes are all IRC/CTCP commands and error/response codes.
// To register a callback for all events, pass "*" as the event code.
// This function returns the ID of the registered callback for later management.
func (irc *Connection) AddCallback(eventcode string, callback func(*Event)) int {
	eventcode = strings.ToUpper(eventcode)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if irc.events == nil {
		irc.events = make(map[string]map[int]func(*Event))
	}

	if _, ok := irc.events[eventcode]; !ok {
		irc.events[eventcode] = make(map[int]func(*Event))
	}
	id := irc.idCounter
	irc.idCounter++
	irc.events[eventcode][id] = callback
	return id
}

// RemoveCallback removes callback i (ID) from the given event code.
// This function returns true upon success, false if any error occurs.
func (irc *Connection) RemoveCallback(eventcode string, i int) bool {
	eventcode = strings.ToUpper(eventcode)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if event, ok := irc.events[eventcode]; ok {
		if _, ok := event[i]; ok {
			delete(event, i)
			return true
		}
		irc.Log.Printf("Event found, but no callback found at id %d\n", i)
		return false
	}

	irc.Log.Println("Event not found")
	return false
}

// ClearCallback removes all callbacks from a given event code.
// It returns true if the given event code is found and cleared.
func (irc *Connection) ClearCallback(eventcode string) bool {
	eventcode = strings.ToUpper(eventcode)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if _, ok := irc.events[eventcode]; ok {
		irc.events[eventcode] = make(map[int]func(*Event))
		return true
	}

	irc.Log.Println("Event not found")
	return false
}

// ReplaceCallback replaces callback i (ID) associated with a given event code with a new callback function.
func (irc *Connection) ReplaceCallback(eventcode string, i int, callback func(*Event)) {
	eventcode = strings.ToUpper(eventcode)

	irc.eventsMutex.Lock()
	defer irc.eventsMutex.Unlock()

	if event, ok := irc.events[eventcode]; ok {
		if _, ok := event[i]; ok {
			event[i] = callback
			return
		}
		irc.Log.Printf("Event found, but no callback found at id %d\n", i)
		return
	}
	irc.Log.Printf("Event not found. Use AddCallback\n")
}

// RunCallbacks executes all callbacks associated with a given event.
func (irc *Connection) RunCallbacks(event *Event) {
	msg := event.Message()
	if event.Code == "PRIVMSG" && len(msg) > 2 && msg[0] == '\x01' {
		event.Code = "CTCP" // Unknown CTCP

		if i := strings.LastIndex(msg, "\x01"); i > 0 {
			msg = msg[1:i]
		} else {
			irc.Log.Printf("Invalid CTCP Message: %s\n", strconv.Quote(msg))
			return
		}

		switch {
		case msg == "VERSION":
			event.Code = "CTCP_VERSION"
		case msg == "TIME":
			event.Code = "CTCP_TIME"
		case strings.HasPrefix(msg, "PING"):
			event.Code = "CTCP_PING"
		case msg == "USERINFO":
			event.Code = "CTCP_USERINFO"
		case msg == "CLIENTINFO":
			event.Code = "CTCP_CLIENTINFO"
		case strings.HasPrefix(msg, "ACTION"):
			event.Code = "CTCP_ACTION"
			if len(msg) > 6 {
				msg = msg[7:]
			} else {
				msg = ""
			}
		}

		event.Arguments[len(event.Arguments)-1] = msg
	}

	irc.eventsMutex.Lock()
	callbacks := make(map[int]func(*Event))
	if eventCallbacks, ok := irc.events[event.Code]; ok {
		for id, callback := range eventCallbacks {
			callbacks[id] = callback
		}
	}
	if allCallbacks, ok := irc.events["*"]; ok {
		for id, callback := range allCallbacks {
			callbacks[id] = callback
		}
	}
	irc.eventsMutex.Unlock()

	if irc.VerboseCallbackHandler {
		irc.Log.Printf("%v (%v) >> %#v\n", event.Code, len(callbacks), event)
	}

	event.Ctx = context.Background()
	if irc.CallbackTimeout != 0 {
		var cancel context.CancelFunc
		event.Ctx, cancel = context.WithTimeout(event.Ctx, irc.CallbackTimeout)
		defer cancel()
	}

	done := make(chan int)
	for id, callback := range callbacks {
		go func(id int, done chan<- int, cb func(*Event), event *Event) {
			start := time.Now()
			cb(event)
			select {
			case done <- id:
			case <-event.Ctx.Done():
				irc.Log.Printf("Canceled callback %s finished in %s >> %#v\n",
					getFunctionName(cb),
					time.Since(start),
					event,
				)
			}
		}(id, done, callback, event)
	}

	for len(callbacks) > 0 {
		select {
		case jobID := <-done:
			delete(callbacks, jobID)
		case <-event.Ctx.Done():
			timedOutCallbacks := []string{}
			for _, cb := range callbacks {
				timedOutCallbacks = append(timedOutCallbacks, getFunctionName(cb))
			}
			irc.Log.Printf("Timeout while waiting for %d callback(s) to finish (%s)\n",
				len(callbacks),
				strings.Join(timedOutCallbacks, ", "),
			)
			return
		}
	}
}

func getFunctionName(f func(*Event)) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// setupCallbacks sets up some initial callbacks to handle the IRC/CTCP protocol.
func (irc *Connection) setupCallbacks() {
	irc.events = make(map[string]map[int]func(*Event))

	// Handle PING events
	irc.AddCallback("PING", func(e *Event) {
		irc.SendRaw("PONG :" + e.Message())

		// REMOVED: Activity-based connection detection (caused false positives in mass deployments)
		// PING events alone don't guarantee full IRC registration completion
	})

	// Version handler
	irc.AddCallback("CTCP_VERSION", func(e *Event) {
		irc.SendRawf("NOTICE %s :\x01VERSION %s\x01", e.Nick, irc.Version)
	})

	// Userinfo handler
	irc.AddCallback("CTCP_USERINFO", func(e *Event) {
		irc.SendRawf("NOTICE %s :\x01USERINFO %s\x01", e.Nick, irc.user)
	})

	// Clientinfo handler
	irc.AddCallback("CTCP_CLIENTINFO", func(e *Event) {
		irc.SendRawf("NOTICE %s :\x01CLIENTINFO PING VERSION TIME USERINFO CLIENTINFO\x01", e.Nick)
	})

	// Time handler
	irc.AddCallback("CTCP_TIME", func(e *Event) {
		ltime := time.Now()
		irc.SendRawf("NOTICE %s :\x01TIME %s\x01", e.Nick, ltime.String())
	})

	// Ping handler
	irc.AddCallback("CTCP_PING", func(e *Event) {
		irc.SendRawf("NOTICE %s :\x01%s\x01", e.Nick, e.Message())
	})

	// Handle nickname in use (433) - RFC 2812 compliant
	irc.AddCallback("433", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// Track the error regardless of connection state
		irc.nickError = "Nickname already in use"

		// FIXED: Handle error regardless of connection state (RFC 2812 requirement)
		// Check if this error is for our desired nickname
		if len(e.Arguments) > 1 {
			attemptedNick := e.Arguments[1]

			// If this was our current desired nick, we need to try a different one
			if attemptedNick == irc.nick {
				if irc.nickcurrent == "" {
					irc.nickcurrent = irc.nick
				}
				irc.modifyNick()
				irc.lastNickChange = time.Now()
				irc.SendRawf("NICK %s", irc.nickcurrent)

				if irc.Debug {
					irc.Log.Printf("NICK 433 error for %s, trying %s (connected: %v)", attemptedNick, irc.nickcurrent, irc.fullyConnected)
				}
			}
		}
	})

	// Handle unavailable resource (437) - RFC 2812 compliant
	irc.AddCallback("437", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// Track the error regardless of connection state
		irc.nickError = "Nickname temporarily unavailable"

		// FIXED: Handle error regardless of connection state (RFC 2812 requirement)
		if len(e.Arguments) > 1 {
			attemptedNick := e.Arguments[1]

			if attemptedNick == irc.nick {
				if irc.nickcurrent == "" {
					irc.nickcurrent = irc.nick
				}
				irc.modifyNick()
				irc.lastNickChange = time.Now()
				irc.SendRawf("NICK %s", irc.nickcurrent)

				if irc.Debug {
					irc.Log.Printf("NICK 437 error for %s, trying %s (connected: %v)", attemptedNick, irc.nickcurrent, irc.fullyConnected)
				}
			}
		}
	})

	// Handle no nickname given (431) - RFC 2812 compliant
	irc.AddCallback("431", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// Track the error regardless of connection state
		irc.nickError = "No nickname given"

		// FIXED: Handle error regardless of connection state (RFC 2812 requirement)
		// For 431, we should always try to send a valid nickname
		if irc.nickcurrent == "" {
			irc.nickcurrent = irc.nick
		}
		if irc.nick != "" {
			irc.lastNickChange = time.Now()
			irc.SendRawf("NICK %s", irc.nick)

			if irc.Debug {
				irc.Log.Printf("NICK 431 error, resending nick %s (connected: %v)", irc.nick, irc.fullyConnected)
			}
		}
	})

	// Handle erroneous nickname (432) - RFC 2812 compliant
	irc.AddCallback("432", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// Track the error regardless of connection state
		irc.nickError = "Erroneous nickname"

		// FIXED: Handle error regardless of connection state (RFC 2812 requirement)
		if len(e.Arguments) > 1 {
			attemptedNick := e.Arguments[1]

			if attemptedNick == irc.nick || attemptedNick == irc.nickcurrent {
				if irc.nickcurrent == "" {
					irc.nickcurrent = irc.nick
				}
				// Add prefix 'Err' to try a different nickname
				irc.nickcurrent = "Err" + irc.nickcurrent
				irc.lastNickChange = time.Now()
				irc.SendRawf("NICK %s", irc.nickcurrent)

				if irc.Debug {
					irc.Log.Printf("NICK 432 error for %s, trying %s (connected: %v)", attemptedNick, irc.nickcurrent, irc.fullyConnected)
				}
			}
		}
	})

	// Handle nickname collision (436) - RFC 2812 compliant
	irc.AddCallback("436", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// Track the error regardless of connection state
		irc.nickError = "Nickname collision"

		// FIXED: Handle error regardless of connection state (RFC 2812 requirement)
		if len(e.Arguments) > 1 {
			attemptedNick := e.Arguments[1]

			if attemptedNick == irc.nick {
				if irc.nickcurrent == "" {
					irc.nickcurrent = irc.nick
				}
				irc.modifyNick()
				irc.lastNickChange = time.Now()
				irc.SendRawf("NICK %s", irc.nickcurrent)

				if irc.Debug {
					irc.Log.Printf("NICK 436 error for %s, trying %s (connected: %v)", attemptedNick, irc.nickcurrent, irc.fullyConnected)
				}
			}
		}
	})

	// Handle restricted nickname (484) - RFC 2812 compliant
	irc.AddCallback("484", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// Track the error regardless of connection state
		irc.nickError = "Restricted nickname"

		// FIXED: Handle error regardless of connection state (RFC 2812 requirement)
		// For 484, we typically don't retry as the nickname is restricted
		// Just log the error for debugging
		if irc.Debug {
			irc.Log.Printf("NICK 484 error: Restricted nickname (connected: %v)", irc.fullyConnected)
		}
		// Keep the current nickname and do not attempt to change it further
	})

	// Handle PONG responses
	irc.AddCallback("PONG", func(e *Event) {
		ns, _ := strconv.ParseInt(e.Message(), 10, 64)
		delta := time.Duration(time.Now().UnixNano() - ns)
		if irc.Debug {
			irc.Log.Printf("Lag: %.3f s\n", delta.Seconds())
		}
	})

	// Handle NICK changes
	// According to RFC 2812 section 3.1.2, the proper format for a nickname change is:
	// :OLD_NICK!user@host NICK NEW_NICK
	irc.AddCallback("NICK", func(e *Event) {
		irc.Lock()
		defer irc.Unlock()

		// If this is our own nickname change
		if e.Nick == irc.nickcurrent {
			// Verify that the message format is correct
			newNick := e.Message()
			if newNick != "" {
				// Update current nickname to the new one
				irc.nickcurrent = newNick

				// ENHANCED: Clear nick change in progress flag (race condition fix)
				irc.nickChangeInProgress = false

				// FIXED: Always update desired nickname on successful change
				// This ensures synchronization between desired and current nick
				irc.nick = newNick

				// Update the last nickname change time
				irc.lastNickChange = time.Now()
				// Clear any nickname error since the change was successful
				irc.nickError = ""

				if irc.Debug {
					irc.Log.Printf("NICK change confirmed: %s -> %s", e.Nick, newNick)
				}
			}
		}
	})

	// Set fullyConnected to true on successful connection (001)
	// This is the server welcome message that confirms our connection and nickname
	irc.AddCallback("001", func(e *Event) {
		irc.Lock()
		// The first argument contains our confirmed nickname
		irc.nickcurrent = e.Arguments[0]
		// Also update the desired nickname to match what the server confirmed
		irc.nick = e.Arguments[0]
		// Mark the connection as fully established
		irc.fullyConnected = true
		// Update the last nickname change time
		irc.lastNickChange = time.Now()
		// Clear any nickname error since we're successfully connected
		irc.nickError = ""
		// Start registration process tracking
		irc.registrationSteps = 1
		irc.registrationStartTime = time.Now()
		irc.Unlock()
	})

	// Handle server pacing notice (some networks use 020)
	irc.AddCallback("020", func(e *Event) {
		if irc.Respect020Pacing {
			irc.Lock()
			irc.got020 = true
			irc.last020 = time.Now()
			irc.Unlock()
		}
	})

	// Handle RPL_YOURHOST (002)
	irc.AddCallback("002", func(e *Event) {
		irc.Lock()
		if !irc.fullyConnected && irc.registrationSteps > 0 {
			irc.registrationSteps++
		} else if irc.registrationSteps > 0 {
			// If we're already fully connected, ensure it stays that way
			irc.fullyConnected = true
		}
		irc.Unlock()
	})

	// Handle RPL_CREATED (003)
	irc.AddCallback("003", func(e *Event) {
		irc.Lock()
		if !irc.fullyConnected && irc.registrationSteps > 0 {
			irc.registrationSteps++
		} else if irc.registrationSteps > 0 {
			// If we're already fully connected, ensure it stays that way
			irc.fullyConnected = true
		}
		irc.Unlock()
	})

	// Handle RPL_MYINFO (004)
	irc.AddCallback("004", func(e *Event) {
		irc.Lock()
		if !irc.fullyConnected && irc.registrationSteps > 0 {
			irc.registrationSteps++
		} else if irc.registrationSteps > 0 {
			// If we're already fully connected, ensure it stays that way
			irc.fullyConnected = true
		}
		irc.Unlock()
	})

	// Handle RPL_ISUPPORT (005)
	irc.AddCallback("005", func(e *Event) {
		irc.Lock()
		if !irc.fullyConnected && irc.registrationSteps > 0 {
			irc.registrationSteps++
			// If we've received enough registration messages, mark as fully connected
			if irc.registrationSteps >= 4 {
				irc.fullyConnected = true
			}
		} else if irc.registrationSteps > 0 {
			// If we're already fully connected, ensure it stays that way
			irc.fullyConnected = true
		}
		irc.Unlock()
	})

	// Handle RPL_ENDOFMOTD (376) - End of MOTD
	irc.AddCallback("376", func(e *Event) {
		irc.Lock()
		// If we've started registration but aren't fully connected yet
		if !irc.fullyConnected && irc.registrationSteps > 0 {
			irc.fullyConnected = true
		}
		irc.Unlock()
	})

	// Handle ERR_NOMOTD (422) - No MOTD
	irc.AddCallback("422", func(e *Event) {
		irc.Lock()
		// If we've started registration but aren't fully connected yet
		if !irc.fullyConnected && irc.registrationSteps > 0 {
			irc.fullyConnected = true
		}
		irc.Unlock()
	})
	// Handle JOIN events
	irc.AddCallback("JOIN", func(e *Event) {
		// REMOVED: Activity-based connection detection (caused false positives)
		// JOIN events can occur during reconnection before full registration
		// Only handle JOIN logic here, not connection state

		// NEW: Lightweight self-validation for our own nick
		if e.Nick != "" {
			irc.ValidateOwnNick(e.Nick)
		}
	})

	// Handle PART events
	irc.AddCallback("PART", func(e *Event) {
		// REMOVED: Activity-based connection detection (caused false positives)
		// PART events can occur during reconnection before full registration
		// Only handle PART logic here, not connection state

		// NEW: Lightweight self-validation for our own nick
		if e.Nick != "" {
			irc.ValidateOwnNick(e.Nick)
		}
	})

	// Handle MODE events
	irc.AddCallback("MODE", func(e *Event) {
		// REMOVED: Activity-based connection detection (caused false positives)
		// MODE events can occur during reconnection before full registration
		// Only handle MODE logic here, not connection state
	})

	// Handle PRIVMSG events
	irc.AddCallback("PRIVMSG", func(e *Event) {
		// REMOVED: Activity-based connection detection (caused false positives in mass deployments)
		// PRIVMSG can arrive from buffers/delays after reconnection, before full registration
		// This was the main source of false positives with 500+ concurrent connections

		// NEW: Lightweight self-validation for our own nick
		if e.Nick != "" {
			irc.ValidateOwnNick(e.Nick)
		}
	})

	// Instead of using a goroutine with sleep, we'll check the timeout in GetNickStatus
	// This avoids potential goroutine leaks in tests and ensures the timeout is checked
	// only when needed

	// DCC Chat support
	irc.addDCCChatCallback()

}

// modifyNick modifies the current nickname to try a different one.
func (irc *Connection) modifyNick() {
	if len(irc.nickcurrent) > 8 {
		irc.nickcurrent = "_" + irc.nickcurrent
	} else {
		irc.nickcurrent = irc.nickcurrent + "_"
	}
}

// DCC chat support
func (irc *Connection) addDCCChatCallback() {
	irc.AddCallback("CTCP_DCC", func(e *Event) {
		if len(e.Arguments) < 5 || e.Arguments[1] != "CHAT" {
			return
		}
		nick := e.Nick
		ip := net.ParseIP(e.Arguments[3])
		port, _ := strconv.Atoi(e.Arguments[4])

		go irc.handleIncomingDCCChat(nick, ip, port)
	})
}
