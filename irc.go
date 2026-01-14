// Copyright (c) 2009 Thomas Jager. All rights reserved.
// Copyright (c) 2024 Jerzy Dąbrowski. All rights reserved.
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

/*
This package provides an event-based IRC client library. It allows you to
register callbacks for the events you need to handle. Its features
include handling standard CTCP, reconnecting on errors, and detecting
stone servers.
Details of the IRC protocol can be found in the following RFCs:
https://tools.ietf.org/html/rfc1459
https://tools.ietf.org/html/rfc2810
https://tools.ietf.org/html/rfc2811
https://tools.ietf.org/html/rfc2812
https://tools.ietf.org/html/rfc2813
The details of the client-to-client protocol (CTCP) can be found here: http://www.irchelp.org/irchelp/rfc/ctcpspec.html
*/

package irc

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
	"golang.org/x/text/encoding"
	"h12.io/socks"
)

const (
	VERSION = "go-ircevo v1.2.3"
)

const CAP_TIMEOUT = time.Second * 15

var ErrDisconnected = errors.New("Disconnect Called")

type socks4Dialer struct {
	dialFunc func(string, string) (net.Conn, error)
}

func (d *socks4Dialer) Dial(network, addr string) (net.Conn, error) {
	return d.dialFunc(network, addr)
}

// Read data from a connection. To be used as a goroutine.
func (irc *Connection) readLoop() {
	defer irc.Done()
	r := irc.Encoding.NewDecoder().Reader(irc.socket)
	br := bufio.NewReaderSize(r, 512)

	errChan := irc.ErrorChan()

	for {
		select {
		case <-irc.end:
			return
		default:
			// Set a read deadline based on the combined timeout and ping frequency
			// We should ALWAYS have received a response from the server within the timeout
			// after our own pings
			if irc.socket != nil {
				irc.socket.SetReadDeadline(time.Now().Add(irc.Timeout + irc.PingFreq))
			}

			msg, err := br.ReadString('\n')

			// We got past our blocking read, so clear timeout
			if irc.socket != nil {
				var zero time.Time
				irc.socket.SetReadDeadline(zero)
			}

			if err != nil {
				errChan <- err
				return
			}

			if irc.Debug {
				irc.Log.Printf("<-- %s\n", strings.TrimSpace(msg))
			}

			irc.lastMessageMutex.Lock()
			irc.lastMessage = time.Now()
			irc.lastMessageMutex.Unlock()
			event, err := parseToEvent(msg)
			if err == nil {
				event.Connection = irc
				if irc.HandleErrorAsDisconnect && strings.ToUpper(event.Code) == "ERROR" {
					errorMsg := event.Message()

					// ENHANCED: Smart ERROR handling - analyze message to determine if reconnect should be blocked
					if irc.SmartErrorHandling {
						errorType := AnalyzeErrorMessage(errorMsg)

						if irc.Debug {
							irc.Log.Printf("ERROR analysis: %s -> %s", errorMsg, errorType.String())
						}

						// Handle different error types appropriately
						switch errorType {
						case PermanentError:
							// Block reconnection for permanent errors
							errChan <- errors.New("Received permanent ERROR from server: " + errorMsg)
							return

						case RecoverableError:
							// Treat recoverable errors as disconnect triggers to allow controlled reconnects
							if irc.Debug {
								irc.Log.Printf("Recoverable ERROR detected, will attempt controlled reconnect")
							}
							irc.RunCallbacks(event)
							errChan <- errors.New("Received RecoverableError from server: " + errorMsg)
							return

						default:
							// For server/network errors, signal error but allow reconnection
							irc.RunCallbacks(event)
							errChan <- errors.New("Received " + errorType.String() + " from server: " + errorMsg)
							return
						}
					} else {
						// Original behavior - block all ERROR messages
						errChan <- errors.New("Received ERROR from server: " + errorMsg)
						return
					}
				}
				irc.RunCallbacks(event)
			}
		}
	}
}

// Unescape tag values as defined in the IRCv3.2 message tags spec
// http://ircv3.net/specs/core/message-tags-3.2.html
func unescapeTagValue(value string) string {
	value = strings.Replace(value, "\\:", ";", -1)
	value = strings.Replace(value, "\\s", " ", -1)
	value = strings.Replace(value, "\\\\", "\\", -1)
	value = strings.Replace(value, "\\r", "\r", -1)
	value = strings.Replace(value, "\\n", "\n", -1)
	return value
}

// Parse raw IRC messages
func parseToEvent(msg string) (*Event, error) {
	msg = strings.TrimSuffix(msg, "\n") // Remove \r\n
	msg = strings.TrimSuffix(msg, "\r")
	event := &Event{Raw: msg}
	// Minimum valid IRC message is a single command (e.g., "PING")
	// Changed from 5 to 1 to support all valid IRC messages
	if len(msg) < 1 {
		return nil, errors.New("malformed msg from server")
	}

	if msg[0] == '@' {
		// IRCv3 Message Tags
		if i := strings.Index(msg, " "); i > -1 {
			event.Tags = make(map[string]string)
			tags := strings.Split(msg[1:i], ";")
			for _, data := range tags {
				parts := strings.SplitN(data, "=", 2)
				if len(parts) == 1 {
					event.Tags[parts[0]] = ""
				} else {
					event.Tags[parts[0]] = unescapeTagValue(parts[1])
				}
			}
			msg = msg[i+1:]
		} else {
			return nil, errors.New("malformed msg from server")
		}
	}

	if msg[0] == ':' {
		if i := strings.Index(msg, " "); i > -1 {
			event.Source = msg[1:i]
			msg = msg[i+1:]
		} else {
			return nil, errors.New("malformed msg from server")
		}

		if i, j := strings.Index(event.Source, "!"), strings.Index(event.Source, "@"); i > -1 && j > -1 && i < j {
			event.Nick = event.Source[0:i]
			event.User = event.Source[i+1 : j]
			event.Host = event.Source[j+1:]
		}
	}

	split := strings.SplitN(msg, " :", 2)
	args := strings.Split(split[0], " ")
	event.Code = strings.ToUpper(args[0])
	event.Arguments = args[1:]
	if len(split) > 1 {
		event.Arguments = append(event.Arguments, split[1])
	}
	return event, nil

}

// Loop to write to a connection. To be used as a goroutine.
func (irc *Connection) writeLoop() {
	defer irc.Done()
	w := irc.Encoding.NewEncoder().Writer(irc.socket)
	errChan := irc.ErrorChan()
	for {
		select {
		case <-irc.end:
			return
		case b, ok := <-irc.pwrite:
			if !ok || b == "" || irc.socket == nil {
				return
			}

			if irc.Debug {
				irc.Log.Printf("--> %s\n", strings.TrimSpace(b))
			}

			// Set a write deadline based on the timeout
			irc.socket.SetWriteDeadline(time.Now().Add(irc.Timeout))

			_, err := w.Write([]byte(b))

			// Clear the write deadline
			var zero time.Time
			irc.socket.SetWriteDeadline(zero)

			if err != nil {
				errChan <- err
				return
			}
		}
	}
}

// Pings the server if we have not received any messages for 5 minutes
// to keep the connection alive. To be used as a goroutine.
func (irc *Connection) pingLoop() {
	defer irc.Done()
	ticker := time.NewTicker(1 * time.Minute) // Tick every minute for monitoring
	ticker2 := time.NewTicker(irc.PingFreq)   // Tick at the ping frequency.
	for {
		select {
		case <-ticker.C:
			// Ping if we haven't received anything from the server within the keep-alive period
			irc.lastMessageMutex.Lock()
			if time.Since(irc.lastMessage) >= irc.KeepAlive {
				irc.SendRawf("PING %d", time.Now().UnixNano())
			}
			irc.lastMessageMutex.Unlock()
		case <-ticker2.C:
			// Ping at the ping frequency
			irc.SendRawf("PING %d", time.Now().UnixNano())
			// Check if there's a pending nickname change
			irc.Lock()
			if irc.nick != irc.nickcurrent {
				desiredNick := irc.nick
				irc.nickPending = desiredNick
				irc.nickChangeInProgress = true
				irc.nickChangeTimeout = time.Now()
				irc.Unlock()
				// Send a NICK command to try to change to the desired nickname
				// The actual change will only happen when the server confirms it
				irc.SendRawf("NICK %s", desiredNick)
				irc.Lock()
			}
			irc.Unlock()
		case <-irc.end:
			ticker.Stop()
			ticker2.Stop()
			return
		}
	}
}

func (irc *Connection) isQuitting() bool {
	irc.Lock()
	defer irc.Unlock()
	return irc.quit
}

// Main loop to control the connection.
func (irc *Connection) Loop() {
	errChan := irc.ErrorChan()
	for !irc.isQuitting() {
		err := <-errChan
		// Decide reconnection strategy based on error content
		errStr := err.Error()
		if irc.HandleErrorAsDisconnect {
			// Permanent errors should not reconnect
			if strings.Contains(errStr, "Received permanent ERROR from server:") {
				irc.Log.Printf("Received permanent ERROR event, not attempting automatic reconnect.")
				return
			}
			// Limit recoverable reconnection attempts if configured
			if strings.Contains(errStr, "RecoverableError") {
				if irc.MaxRecoverableReconnects > 0 && irc.recoverableReconnects >= irc.MaxRecoverableReconnects {
					irc.Log.Printf("Max recoverable reconnect attempts reached (%d); stopping.", irc.MaxRecoverableReconnects)
					return
				}
			}
		}
		if irc.end != nil {
			close(irc.end)
		}
		irc.Wait()
		for !irc.isQuitting() {
			irc.Log.Printf("Error, disconnected: %s\n", err)
			// Track recoverable reconnect attempts
			if strings.Contains(errStr, "RecoverableError") {
				irc.recoverableReconnects++
			}
			if err = irc.Reconnect(); err != nil {
				irc.Log.Printf("Error while reconnecting: %s\n", err)
				time.Sleep(60 * time.Second)
			} else {
				errChan = irc.ErrorChan()
				// Reset counters after successful reconnect
				irc.recoverableReconnects = 0
				break
			}
		}
	}
}

// Quit the current connection and disconnect from the server
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.1.6
func (irc *Connection) Quit() {
	quit := "QUIT"

	if irc.QuitMessage != "" {
		quit = fmt.Sprintf("QUIT :%s", irc.QuitMessage)
	}

	// NEW: Add 1 second delay before sending QUIT to IRC server
	// This helps ensure proper cleanup and reduces server-side race conditions
	time.Sleep(1 * time.Second)

	irc.SendRaw(quit)
	irc.Lock()
	irc.stopped = true
	irc.quit = true
	irc.Unlock()
}

// Use the connection to join a given channel.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.2.1
func (irc *Connection) Join(channel string) {
	irc.pwrite <- fmt.Sprintf("JOIN %s\r\n", channel)
}

// Leave a given channel.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.2.2
func (irc *Connection) Part(channel string) {
	irc.pwrite <- fmt.Sprintf("PART %s\r\n", channel)
}

// Send a notification to a nickname. This is similar to Privmsg but must not receive replies.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.4.2
func (irc *Connection) Notice(target, message string) {
	irc.pwrite <- fmt.Sprintf("NOTICE %s :%s\r\n", target, message)
}

// Send a formatted notification to a nickname.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.4.2
func (irc *Connection) Noticef(target, format string, a ...interface{}) {
	irc.Notice(target, fmt.Sprintf(format, a...))
}

// Send (action) message to a target (channel or nickname).
// No clear RFC on this one...
func (irc *Connection) Action(target, message string) {
	irc.pwrite <- fmt.Sprintf("PRIVMSG %s :\001ACTION %s\001\r\n", target, message)
}

// Send formatted (action) message to a target (channel or nickname).
func (irc *Connection) Actionf(target, format string, a ...interface{}) {
	irc.Action(target, fmt.Sprintf(format, a...))
}

// Send (private) message to a target (channel or nickname).
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.4.1
func (irc *Connection) Privmsg(target, message string) {
	irc.pwrite <- fmt.Sprintf("PRIVMSG %s :%s\r\n", target, message)
}

// Send formatted string to specified target (channel or nickname).
func (irc *Connection) Privmsgf(target, format string, a ...interface{}) {
	irc.Privmsg(target, fmt.Sprintf(format, a...))
}

// Kick <user> from <channel> with <msg>. For no message, pass empty string ("")
func (irc *Connection) Kick(user, channel, msg string) {
	var cmd bytes.Buffer
	cmd.WriteString(fmt.Sprintf("KICK %s %s", channel, user))
	if msg != "" {
		cmd.WriteString(fmt.Sprintf(" :%s", msg))
	}
	cmd.WriteString("\r\n")
	irc.pwrite <- cmd.String()
}

// Kick all <users> from <channel> with <msg>. For no message, pass
// empty string ("")
func (irc *Connection) MultiKick(users []string, channel string, msg string) {
	var cmd bytes.Buffer
	cmd.WriteString(fmt.Sprintf("KICK %s %s", channel, strings.Join(users, ",")))
	if msg != "" {
		cmd.WriteString(fmt.Sprintf(" :%s", msg))
	}
	cmd.WriteString("\r\n")
	irc.pwrite <- cmd.String()
}

// Send raw string.
func (irc *Connection) SendRaw(message string) {
	irc.pwrite <- message + "\r\n"
}

// Send raw formatted string.
func (irc *Connection) SendRawf(format string, a ...interface{}) {
	irc.SendRaw(fmt.Sprintf(format, a...))
}

// Set (new) nickname.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.1.2
// RFC 2812 details: https://tools.ietf.org/html/rfc2812#section-3.1.2
//
// This function sends a NICK command to the server to request a nickname change.
// The actual nickname change is only confirmed when the server sends back a
// NICK message in the format: :OLD_NICK!user@host NICK NEW_NICK
//
// The function updates the desired nickname (irc.nick) but does not update
// the current nickname (irc.nickcurrent) until confirmation is received.
func (irc *Connection) Nick(n string) {
	irc.Lock()
	defer irc.Unlock()

	// ENHANCED: Prevent multiple simultaneous nick changes (race condition fix)
	if irc.nickChangeInProgress && time.Since(irc.nickChangeTimeout) < 30*time.Second {
		// Update desired nickname so the pingLoop can retry once the current change completes
		irc.nick = n
		irc.nickPending = n
		irc.lastNickChange = time.Now()

		if irc.Debug {
			irc.Log.Printf("NICK change already in progress, queuing desired nick %s", n)
		}
		return
	}

	// Update only the desired nickname
	irc.nick = n
	// Record when we attempted to change the nickname
	irc.lastNickChange = time.Now()

	// Only send NICK command if it's different from current
	if irc.nickcurrent != n {
		irc.nickChangeInProgress = true
		irc.nickChangeTimeout = time.Now()
		irc.nickPending = n

		// Send the NICK command to the server (unlock first to avoid deadlock)
		irc.Unlock()
		irc.SendRawf("NICK %s", n)
		irc.Lock()

		if irc.Debug {
			irc.Log.Printf("Sent NICK change request: %s -> %s", irc.nickcurrent, n)
		}
		return
	}

	// No change needed; clear any pending state
	irc.nickPending = ""
	irc.nickChangeInProgress = false
}

// GetNick returns the current nickname used in the IRC connection.
// This method is thread-safe.
//
// Note: This method only returns the current nickname and does not provide
// information about whether the nickname has been confirmed by the server
// or if there are any pending nickname changes. For more detailed nickname
// status information, use GetNickStatus() instead.
func (irc *Connection) GetNick() string {
	irc.Lock()
	defer irc.Unlock()
	return irc.nickcurrent
}

// GetNickStatus returns detailed information about the current nickname status.
// This includes whether the nickname has been confirmed by the server,
// any pending nickname changes, and error states.
//
// According to RFC 2812 section 3.1.2, a nickname change is only confirmed
// when the server sends a NICK message in the format:
// :OLD_NICK!user@host NICK NEW_NICK
//
// The Current field contains the nickname that has been confirmed by the server.
// The Desired field contains the nickname that was requested with Nick().
// The PendingChange field is true when Current and Desired are different.
//
// This method is thread-safe and provides more comprehensive information
// than GetNick().
func (irc *Connection) GetNickStatus() *NickStatus {
	irc.Lock()
	defer irc.Unlock()

	// If lastNickChange is zero, use the current time
	lastChangeTime := irc.lastNickChange
	if lastChangeTime.IsZero() {
		lastChangeTime = time.Now()
	}

	// FIXED: Only use timeout fallback if explicitly enabled (prevents ghost bots)
	// This was the main source of ghost bots in mass deployments
	if irc.EnableTimeoutFallback && !irc.fullyConnected && irc.nickcurrent != "" && irc.registrationSteps > 0 {
		// Check if registration timeout has elapsed
		if !irc.registrationStartTime.IsZero() && time.Since(irc.registrationStartTime) >= irc.registrationTimeout {
			irc.fullyConnected = true
			if irc.Debug {
				irc.Log.Printf("Setting fullyConnected=true in GetNickStatus due to timeout fallback (EnableTimeoutFallback=true)\n")
			}
		}
	}

	return &NickStatus{
		Current:        irc.nickcurrent,
		Desired:        irc.nick,
		Confirmed:      irc.fullyConnected,
		LastChangeTime: lastChangeTime,
		PendingChange:  irc.nick != irc.nickcurrent,
		Error:          irc.nickError,
	}
}

// Query information about a particular nickname.
// RFC 1459: https://tools.ietf.org/html/rfc1459#section-4.5.2
func (irc *Connection) Whois(nick string) {
	irc.SendRawf("WHOIS %s", nick)
}

// Query information about a given nickname in the server.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.5.1
func (irc *Connection) Who(nick string) {
	irc.SendRawf("WHO %s", nick)
}

// Set different modes for a target (channel or nickname).
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.2.3
func (irc *Connection) Mode(target string, modestring ...string) {
	if len(modestring) > 0 {
		mode := strings.Join(modestring, " ")
		irc.SendRawf("MODE %s %s", target, mode)
		return
	}
	irc.SendRawf("MODE %s", target)
}

func (irc *Connection) ErrorChan() chan error {
	return irc.Error
}

// Returns true if the connection is connected to an IRC server.
func (irc *Connection) Connected() bool {
	return !irc.stopped
}

// A disconnect sends all buffered messages (if possible),
// stops all goroutines and then closes the socket.
func (irc *Connection) Disconnect() {
	irc.Lock()
	irc.fullyConnected = false
	irc.registrationSteps = 0
	irc.registrationStartTime = time.Time{}
	irc.nickPending = ""
	irc.nickChangeInProgress = false
	defer irc.Unlock()

	if irc.end != nil {
		close(irc.end)
	}

	irc.Wait()

	irc.end = nil

	if irc.pwrite != nil {
		close(irc.pwrite)
	}

	if irc.socket != nil {
		irc.socket.Close()
	}
	irc.ErrorChan() <- ErrDisconnected
}

// Reconnect to a server using the current connection.
func (irc *Connection) Reconnect() error {
	irc.Lock()
	irc.fullyConnected = false
	irc.registrationSteps = 0
	irc.registrationStartTime = time.Time{}
	irc.nickPending = ""
	irc.nickChangeInProgress = false
	irc.Unlock()
	irc.end = make(chan struct{})
	return irc.Connect(irc.Server)
}

// Connect to a given server using the current connection configuration.
// This function also takes care of identification if a password is provided.
// RFC 1459 details: https://tools.ietf.org/html/rfc1459#section-4.1
func (irc *Connection) Connect(server string) error {
	irc.Server = server
	// Mark Server as stopped since there can be an error during connect
	irc.stopped = true

	// Reset registration status
	irc.Lock()
	irc.fullyConnected = false
	irc.registrationSteps = 0
	irc.registrationStartTime = time.Time{}
	irc.nickPending = ""
	irc.nickChangeInProgress = false
	irc.Unlock()

	// Make sure everything is ready for connection
	if len(irc.Server) == 0 {
		return errors.New("empty 'server'")
	}
	if strings.Index(irc.Server, ":") == 0 {
		return errors.New("hostname is missing")
	}
	if strings.Index(irc.Server, ":") == len(irc.Server)-1 {
		return errors.New("port missing")
	}
	_, ports, err := net.SplitHostPort(irc.Server)
	if err != nil {
		return errors.New("wrong address string")
	}
	port, err := strconv.Atoi(ports)
	if err != nil {
		return errors.New("extracting port failed")
	}
	if !((port >= 0) && (port <= 65535)) {
		return errors.New("port number outside valid range")
	}
	if irc.Log == nil {
		return errors.New("'Log' points to nil")
	}
	if len(irc.nick) == 0 {
		return errors.New("empty 'nick'")
	}
	if len(irc.user) == 0 {
		return errors.New("empty 'user'")
	}

	var localAddr net.Addr
	if irc.localIP != "" {
		localAddr = &net.TCPAddr{
			IP:   net.ParseIP(irc.localIP),
			Port: 0,
		}
	}

	var dialer proxy.Dialer
	if irc.ProxyConfig != nil {
		switch irc.ProxyConfig.Type {
		case "socks4":
			socks4Proxy := socks.Dial(fmt.Sprintf("socks4://%s:%s@%s", irc.ProxyConfig.Username, irc.ProxyConfig.Password, irc.ProxyConfig.Address))
			dialer = &socks4Dialer{dialFunc: socks4Proxy}
		case "socks5":
			auth := &proxy.Auth{
				User:     irc.ProxyConfig.Username,
				Password: irc.ProxyConfig.Password,
			}
			socks5Proxy, err := proxy.SOCKS5("tcp", irc.ProxyConfig.Address, auth, proxy.Direct)
			if err != nil {
				return err
			}
			dialer = socks5Proxy
		case "http":
			proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s", irc.ProxyConfig.Username, irc.ProxyConfig.Password, irc.ProxyConfig.Address))
			if err != nil {
				return err
			}
			httpProxy, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				return err
			}
			dialer = httpProxy
		default:
			return fmt.Errorf("unsupported proxy type: %s", irc.ProxyConfig.Type)
		}
	} else {
		dialer = &net.Dialer{
			LocalAddr: localAddr,
			Timeout:   irc.Timeout,
		}
	}

	irc.socket, err = dialer.Dial("tcp", irc.Server)
	if err != nil {
		return err
	}
	if irc.UseTLS {
		irc.socket = tls.Client(irc.socket, irc.TLSConfig)
	}

	if irc.Encoding == nil {
		irc.Encoding = encoding.Nop
	}

	irc.stopped = false
	irc.Log.Printf("Connected to %s (%s)\n", irc.Server, irc.socket.RemoteAddr())

	irc.pwrite = make(chan string, 10)
	irc.Error = make(chan error, 10)
	irc.Add(3)
	go irc.readLoop()
	go irc.writeLoop()
	go irc.pingLoop()

	if len(irc.WebIRC) > 0 {
		irc.pwrite <- fmt.Sprintf("WEBIRC %s\r\n", irc.WebIRC)
	}

	if len(irc.Password) > 0 {
		irc.pwrite <- fmt.Sprintf("PASS %s\r\n", irc.Password)
	}

	err = irc.negotiateCaps()
	if err != nil {
		return err
	}

	// Registration will be sent automatically during CAP negotiation
	return nil
}

func (irc *Connection) SetProxy(proxyType, address, username, password string) {
	irc.ProxyConfig = &ProxyConfig{
		Type:     proxyType,
		Address:  address,
		Username: username,
		Password: password,
	}
}

// Negotiate IRCv3 capabilities
func (irc *Connection) negotiateCaps() error {
	irc.RequestCaps = nil
	irc.AcknowledgedCaps = nil

	var negotiationCallbacks []CallbackID
	defer func() {
		for _, callback := range negotiationCallbacks {
			irc.RemoveCallback(callback.EventCode, callback.ID)
		}
	}()

	saslResChan := make(chan *SASLResult)
	if irc.UseSASL {
		irc.RequestCaps = append(irc.RequestCaps, "sasl")
		negotiationCallbacks = irc.setupSASLCallbacks(saslResChan)
	}

	if len(irc.RequestCaps) == 0 {
		// No capabilities to negotiate: send registration automatically
		irc.Lock()
		if !irc.sentRegistration {
			respectPacing := irc.Respect020Pacing && irc.got020
			realname := irc.user
			if irc.RealName != "" {
				realname = irc.RealName
			}
			irc.sentRegistration = true
			irc.nickPending = irc.nick
			irc.Unlock()
			if respectPacing {
				time.Sleep(250 * time.Millisecond)
			}
			irc.pwrite <- "NICK " + irc.nick + "\r\n"
			irc.pwrite <- "USER " + irc.user + " 0 * :" + realname + "\r\n"
		} else {
			irc.Unlock()
		}
		return nil
	}

	cap_chan := make(chan bool, len(irc.RequestCaps))
	id := irc.AddCallback("CAP", func(e *Event) {
		if len(e.Arguments) < 2 {
			return
		}
		command := e.Arguments[1]

		if command == "LS" {
			// When we see LS, server is ready; send registration now if not sent yet
			missing_caps := len(irc.RequestCaps)
			if len(e.Arguments) >= 3 {
				for _, cap_name := range strings.Split(e.Arguments[2], " ") {
					for _, req_cap := range irc.RequestCaps {
						if cap_name == req_cap {
							irc.pwrite <- fmt.Sprintf("CAP REQ :%s\r\n", cap_name)
							missing_caps--
						}
					}
				}
			}
			// try to send registration early once LS seen
			irc.Lock()
			if !irc.sentRegistration {
				respectPacing := irc.Respect020Pacing && irc.got020
				realname := irc.user
				if irc.RealName != "" {
					realname = irc.RealName
				}
				irc.sentRegistration = true
				irc.Unlock()
				if respectPacing {
					time.Sleep(250 * time.Millisecond)
				}
				irc.pwrite <- "NICK " + irc.nick + "\r\n"
				irc.pwrite <- "USER " + irc.user + " 0 * :" + realname + "\r\n"
			} else {
				irc.Unlock()
			}

			for i := 0; i < missing_caps; i++ {
				cap_chan <- true
			}
		} else if command == "ACK" || command == "NAK" {
			for _, cap_name := range strings.Split(strings.TrimSpace(e.Arguments[len(e.Arguments)-1]), " ") {
				if cap_name == "" {
					continue
				}

				if command == "ACK" {
					irc.AcknowledgedCaps = append(irc.AcknowledgedCaps, cap_name)
				}
				cap_chan <- true
			}
		}
	})
	negotiationCallbacks = append(negotiationCallbacks, CallbackID{"CAP", id})

	// Send CAP LS once, with configured version
	if irc.CapVersion != "" {
		irc.pwrite <- fmt.Sprintf("CAP LS %s\r\n", irc.CapVersion)
	} else {
		irc.pwrite <- "CAP LS\r\n"
	}

	// Fallback: if no CAP LS seen quickly (or handler didn't send registration),
	// send NICK/USER automatically to avoid ping timeouts on some networks
	go func() {
		time.Sleep(800 * time.Millisecond)
		irc.Lock()
		if irc.sentRegistration {
			irc.Unlock()
			return
		}
		respectPacing := irc.Respect020Pacing && irc.got020
		realname := irc.user
		if irc.RealName != "" {
			realname = irc.RealName
		}
		irc.sentRegistration = true
		irc.nickPending = irc.nick
		irc.Unlock()
		if respectPacing {
			time.Sleep(250 * time.Millisecond)
		}
		irc.pwrite <- "NICK " + irc.nick + "\r\n"
		irc.pwrite <- "USER " + irc.user + " 0 * :" + realname + "\r\n"
	}()

	if irc.UseSASL {
		select {
		case res := <-saslResChan:
			if res.Failed {
				return res.Err
			}
		case <-time.After(CAP_TIMEOUT):
			// Raise an error if we can't authenticate with SASL.
			return errors.New("SASL setup timed out. Does the server support SASL?")
		}
	}

	remaining_caps := len(irc.RequestCaps)

	select {
	case <-cap_chan:
		remaining_caps--
	case <-time.After(CAP_TIMEOUT):
		// The server probably doesn't implement CAP LS, which is "normal".
		return nil
	}

	// Wait for all capabilities to be ACKed or NAKed before ending negotiation
	for remaining_caps > 0 {
		<-cap_chan
		remaining_caps--
	}

	irc.pwrite <- "CAP END\r\n"

	return nil
}

// Create a connection with the (publicly visible) nickname and username.
// The nickname is later used to address the user. Returns nil if nick
// or user are empty.
func IRC(nick, user string) *Connection {
	// Catch invalid values
	if len(nick) == 0 {
		return nil
	}
	if len(user) == 0 {
		return nil
	}

	irc := &Connection{
		nick:                  nick,
		nickcurrent:           nick,
		user:                  user,
		Log:                   log.New(os.Stdout, "", log.LstdFlags),
		end:                   make(chan struct{}),
		Version:               VERSION,
		KeepAlive:             4 * time.Minute,
		Timeout:               1 * time.Minute,
		PingFreq:              15 * time.Minute,
		SASLMech:              "PLAIN",
		QuitMessage:           "",
		fullyConnected:        false,            // Initialize to false
		lastNickChange:        time.Now(),       // Initialize to current time
		nickError:             "",               // Initialize to empty string
		registrationSteps:     0,                // Initialize registration steps counter
		registrationStartTime: time.Time{},      // Zero time initially
		registrationTimeout:   30 * time.Second, // Longer timeout for mass deployments

		// NEW: Disable timeout fallback by default to prevent ghost bots
		EnableTimeoutFallback: false, // Disabled by default for reliability

		// NEW: Enable smart ERROR handling by default
		SmartErrorHandling: true, // Analyze ERROR messages intelligently

		DCCManager:              NewDCCManager(), // DCC chat support
		ProxyConfig:             nil,
		HandleErrorAsDisconnect: true, // Default to true to not reconnect after ERROR event

		// NEW: CAP negotiation defaults
		CapVersion:              "302", // Use CAP LS 302 by default
		RegistrationAfterCapEnd: false, // Default to compat mode (send NICK/USER during CAP)
		Respect020Pacing:        true,  // Be nice to networks that send 020

		// NEW: Default limit of 3 recoverable reconnect attempts
		MaxRecoverableReconnects: 3,
	}
	irc.setupCallbacks()
	return irc
}

// SetLocalIP sets the local IP address to bind when connecting.
// This allows the client to specify which local interface/IP to use.
func (irc *Connection) SetLocalIP(ip string) {
	irc.localIP = ip
}

// ValidateConnectionState performs comprehensive connection validation
// This method should be used for periodic verification in mass deployments
// Returns true only if the connection is genuinely active and responsive
func (irc *Connection) ValidateConnectionState() bool {
	irc.Lock()
	defer irc.Unlock()

	// Basic state checks
	if !irc.fullyConnected {
		return false
	}

	if irc.socket == nil {
		irc.fullyConnected = false
		return false
	}

	// Check socket health with minimal timeout
	irc.socket.SetReadDeadline(time.Now().Add(1 * time.Second))
	testBuf := make([]byte, 0)
	_, err := irc.socket.Read(testBuf)

	// Reset deadline
	var zero time.Time
	irc.socket.SetReadDeadline(zero)

	if err != nil && !isTimeoutError(err) && err != io.EOF {
		if irc.Debug {
			irc.Log.Printf("Socket validation failed: %v", err)
		}
		irc.fullyConnected = false
		return false
	}

	// Check if we have recent activity (within 10 minutes)
	irc.lastMessageMutex.Lock()
	lastActivity := irc.lastMessage
	irc.lastMessageMutex.Unlock()

	if time.Since(lastActivity) > 10*time.Minute {
		if irc.Debug {
			irc.Log.Printf("No recent activity for %.1f minutes, connection may be stale",
				time.Since(lastActivity).Minutes())
		}
		// Don't mark as disconnected yet, but return false for verification
		return false
	}

	return true
}

// Helper function to identify timeout errors
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return false
}

// AnalyzeErrorMessage categorizes IRC ERROR messages to determine reconnection strategy
func AnalyzeErrorMessage(errorMsg string) ErrorType {
	errorLower := strings.ToLower(errorMsg)

	// Permanent errors - should prevent reconnection
	permanentPatterns := []string{
		"k-lined", "k-line", "klined",
		"g-lined", "g-line", "glined",
		"banned", "you are banned", "user is banned",
		"unauthorized connection",
		"connection refused",
		"access denied",
		"you are not authorized",
		"blacklisted",
		"throttled", "throttling",
		"flood", "flooding",
		"spam", "spamming",
	}

	for _, pattern := range permanentPatterns {
		if strings.Contains(errorLower, pattern) {
			return PermanentError
		}
	}

	// Server errors - temporary server-side issues
	serverPatterns := []string{
		"too many connections",
		"too many host connections",
		"too many host connections (local)",
		"too many host connections (global)",
		"too many global connections",
		"connection limit exceeded",
		"server full",
		"max connections reached",
		"too many connections from this ip",
		"too many connections from your host",
		"connection limit",
		"host limit",
		"ip limit",
		"clone limit",
		"too many clones",
	}

	for _, pattern := range serverPatterns {
		if strings.Contains(errorLower, pattern) {
			return ServerError
		}
	}

	// Network errors - connectivity issues
	networkPatterns := []string{
		"connection reset",
		"connection timed out",
		"network unreachable",
		"no route to host",
		"connection lost",
		"broken pipe",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errorLower, pattern) {
			return NetworkError
		}
	}

	// Recoverable errors - temporary issues that should allow reconnection
	recoverablePatterns := []string{
		"registration timeout",
		"ping timeout",
		"server shutting down",
		"server restart",
	}

	for _, pattern := range recoverablePatterns {
		if strings.Contains(errorLower, pattern) {
			return RecoverableError
		}
	}

	// Default to recoverable for unknown errors (allow reconnection attempts)
	// Unknown errors are more likely to be temporary than permanent
	return RecoverableError
}

// IsFullyConnected returns whether the connection is fully established with the IRC server.
// The connection is considered fully established in the following cases:
// 1. After receiving the RPL_WELCOME (001) message from the server
// 2. After receiving a sequence of registration messages (001, 002, 003, 004, 005)
// 3. After receiving the end of MOTD (376) or no MOTD (422) messages
// 4. After receiving certain activity events like JOIN, PART, MODE, PRIVMSG, or PING
// 5. After a timeout period since the start of registration
//
// This method is thread-safe.
func (irc *Connection) IsFullyConnected() bool {
	irc.Lock()
	defer irc.Unlock()
	return irc.fullyConnected
}
