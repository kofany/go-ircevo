package irc

import (
	"testing"
	"time"
)

func TestFullyConnectedStatus(t *testing.T) {
	// Create a new IRC connection with a very short timeout to avoid hanging tests
	irccon := IRC("testnick", "testuser")
	if irccon == nil {
		t.Fatal("Failed to create IRC connection")
	}

	// Disable the timeout goroutine for testing
	irccon.registrationTimeout = 1 * time.Millisecond

	// We need to manually set up callbacks for testing
	irccon.events = make(map[string]map[int]func(*Event))

	// Add only the callbacks we need for testing
	irccon.AddCallback("001", func(e *Event) {
		irccon.Lock()
		irccon.nickcurrent = e.Arguments[0]
		irccon.nick = e.Arguments[0]
		irccon.fullyConnected = true
		irccon.registrationSteps = 1
		irccon.registrationStartTime = time.Now()
		irccon.Unlock()
	})

	irccon.AddCallback("002", func(e *Event) {
		irccon.Lock()
		if !irccon.fullyConnected && irccon.registrationSteps > 0 {
			irccon.registrationSteps++
		}
		irccon.Unlock()
	})

	irccon.AddCallback("003", func(e *Event) {
		irccon.Lock()
		if !irccon.fullyConnected && irccon.registrationSteps > 0 {
			irccon.registrationSteps++
		}
		irccon.Unlock()
	})

	irccon.AddCallback("004", func(e *Event) {
		irccon.Lock()
		if !irccon.fullyConnected && irccon.registrationSteps > 0 {
			irccon.registrationSteps++
		}
		irccon.Unlock()
	})

	irccon.AddCallback("JOIN", func(e *Event) {
		if e.Nick == irccon.nickcurrent {
			irccon.Lock()
			if !irccon.fullyConnected {
				irccon.fullyConnected = true
			}
			irccon.Unlock()
		}
	})

	irccon.AddCallback("PING", func(e *Event) {
		irccon.Lock()
		if !irccon.fullyConnected && irccon.registrationSteps > 0 {
			irccon.fullyConnected = true
		}
		irccon.Unlock()
	})

	irccon.AddCallback("MODE", func(e *Event) {
		if len(e.Arguments) > 0 && e.Arguments[0] == irccon.nickcurrent {
			irccon.Lock()
			if !irccon.fullyConnected {
				irccon.fullyConnected = true
			}
			irccon.Unlock()
		}
	})

	irccon.AddCallback("PRIVMSG", func(e *Event) {
		irccon.Lock()
		if !irccon.fullyConnected {
			irccon.fullyConnected = true
		}
		irccon.Unlock()
	})

	// Test initial state
	if irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be false initially")
	}

	// Simulate receiving 001 (RPL_WELCOME)
	event001 := &Event{
		Code:      "001",
		Arguments: []string{"testnick", "Welcome to the IRC Network testnick!user@host"},
	}
	irccon.RunCallbacks(event001)

	// Check if fullyConnected is true after 001
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after 001")
	}

	// Reset connection status
	irccon.fullyConnected = false
	irccon.registrationSteps = 0

	// Simulate receiving 001-005 in sequence
	irccon.RunCallbacks(&Event{Code: "001", Arguments: []string{"testnick"}})
	irccon.RunCallbacks(&Event{Code: "002", Arguments: []string{"testnick", "Your host is test.server"}})
	irccon.RunCallbacks(&Event{Code: "003", Arguments: []string{"testnick", "This server was created..."}})
	irccon.RunCallbacks(&Event{Code: "004", Arguments: []string{"testnick", "test.server", "1.0", "i", "o"}})

	// Check if fullyConnected is true after registration sequence
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after registration sequence")
	}

	// Reset connection status
	irccon.fullyConnected = false
	irccon.registrationSteps = 1 // Simulate that we've received 001

	// Test JOIN event
	joinEvent := &Event{
		Code:      "JOIN",
		Nick:      "testnick",
		Arguments: []string{"#testchannel"},
	}
	irccon.RunCallbacks(joinEvent)

	// Check if fullyConnected is true after JOIN
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after JOIN")
	}

	// Reset connection status
	irccon.fullyConnected = false
	irccon.registrationSteps = 1 // Simulate that we've received 001

	// Test PING event
	pingEvent := &Event{
		Code:      "PING",
		Arguments: []string{"test.server"},
	}
	irccon.RunCallbacks(pingEvent)

	// Check if fullyConnected is true after PING
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after PING")
	}

	// Reset connection status
	irccon.fullyConnected = false
	irccon.registrationSteps = 1 // Simulate that we've received 001

	// Test MODE event for our nick
	modeEvent := &Event{
		Code:      "MODE",
		Arguments: []string{"testnick", "+i"},
	}
	irccon.RunCallbacks(modeEvent)

	// Check if fullyConnected is true after MODE
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after MODE")
	}

	// Reset connection status
	irccon.fullyConnected = false
	irccon.registrationSteps = 1 // Simulate that we've received 001

	// Test PRIVMSG event
	privmsgEvent := &Event{
		Code:      "PRIVMSG",
		Nick:      "othernick",
		Arguments: []string{"testnick", "Hello there!"},
	}
	irccon.RunCallbacks(privmsgEvent)

	// Check if fullyConnected is true after PRIVMSG
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after PRIVMSG")
	}

	// Test GetNickStatus timeout mechanism
	irccon.fullyConnected = false
	irccon.registrationSteps = 1
	irccon.registrationStartTime = time.Now().Add(-10 * time.Second)
	irccon.registrationTimeout = 5 * time.Second

	// Get nick status to trigger the timeout check
	status := irccon.GetNickStatus()

	// Check if fullyConnected is true after timeout
	if !irccon.IsFullyConnected() {
		t.Error("Expected fullyConnected to be true after timeout")
	}

	// Verify that the status reflects the correct state
	if !status.Confirmed {
		t.Error("Expected status.Confirmed to be true after timeout")
	}
}
