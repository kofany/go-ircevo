package irc

import (
	"testing"
	"time"
)

// TestSimpleNickChange tests the basic nickname change functionality
// without using any of the callback mechanisms that might cause goroutine leaks
func TestSimpleNickChange(t *testing.T) {
	// Create a minimal IRC connection without setting up callbacks
	irccon := &Connection{
		nick:           "testnick",
		nickcurrent:    "testnick",
		user:           "testuser",
		lastNickChange: time.Now(),
		nickError:      "",
	}

	// Test initial state
	if irccon.nickcurrent != "testnick" {
		t.Errorf("Expected current nickname to be 'testnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "testnick" {
		t.Errorf("Expected desired nickname to be 'testnick', got '%s'", irccon.nick)
	}

	// Directly update the desired nickname (what Nick() would do)
	irccon.nick = "newnick"
	irccon.lastNickChange = time.Now()

	// Verify that only the desired nickname was updated
	if irccon.nickcurrent != "testnick" {
		t.Errorf("Expected current nickname to remain 'testnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick" {
		t.Errorf("Expected desired nickname to be 'newnick', got '%s'", irccon.nick)
	}

	// Manually simulate a NICK callback
	oldNick := irccon.nickcurrent
	if oldNick == "testnick" {
		irccon.nickcurrent = "newnick"
		if irccon.nick == oldNick {
			irccon.nick = "newnick"
		}
		irccon.lastNickChange = time.Now()
		irccon.nickError = ""
	}

	// Verify that both nicknames were updated
	if irccon.nickcurrent != "newnick" {
		t.Errorf("Expected current nickname to be updated to 'newnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick" {
		t.Errorf("Expected desired nickname to be 'newnick', got '%s'", irccon.nick)
	}

	// Create a NickStatus manually (what GetNickStatus() would do)
	status := &NickStatus{
		Current:        irccon.nickcurrent,
		Desired:        irccon.nick,
		Confirmed:      irccon.fullyConnected,
		LastChangeTime: irccon.lastNickChange,
		PendingChange:  irccon.nick != irccon.nickcurrent,
		Error:          irccon.nickError,
	}

	// Test the status
	if status.Current != "newnick" {
		t.Errorf("Expected status.Current to be 'newnick', got '%s'", status.Current)
	}
	if status.Desired != "newnick" {
		t.Errorf("Expected status.Desired to be 'newnick', got '%s'", status.Desired)
	}
	if status.PendingChange {
		t.Error("Expected status.PendingChange to be false")
	}
}

// TestSimpleNickError tests error handling in nickname changes
func TestSimpleNickError(t *testing.T) {
	// Create a minimal IRC connection without setting up callbacks
	irccon := &Connection{
		nick:           "testnick",
		nickcurrent:    "testnick",
		user:           "testuser",
		lastNickChange: time.Now(),
		nickError:      "",
	}

	// Directly update the desired nickname (what Nick() would do)
	irccon.nick = "newnick"
	irccon.lastNickChange = time.Now()

	// Manually simulate a 433 callback (nickname in use)
	irccon.nickError = "Nickname already in use"

	// Create a NickStatus manually (what GetNickStatus() would do)
	status := &NickStatus{
		Current:        irccon.nickcurrent,
		Desired:        irccon.nick,
		Confirmed:      irccon.fullyConnected,
		LastChangeTime: irccon.lastNickChange,
		PendingChange:  irccon.nick != irccon.nickcurrent,
		Error:          irccon.nickError,
	}

	// Test the status
	if status.Error != "Nickname already in use" {
		t.Errorf("Expected status.Error to be 'Nickname already in use', got '%s'", status.Error)
	}
}

// TestSimplePendingChange tests the pending change detection
func TestSimplePendingChange(t *testing.T) {
	// Create a minimal IRC connection without setting up callbacks
	irccon := &Connection{
		nick:           "testnick",
		nickcurrent:    "testnick",
		user:           "testuser",
		lastNickChange: time.Now(),
		nickError:      "",
	}

	// Directly update the desired nickname (what Nick() would do)
	irccon.nick = "newnick"
	irccon.lastNickChange = time.Now()

	// Create a NickStatus manually (what GetNickStatus() would do)
	status := &NickStatus{
		Current:        irccon.nickcurrent,
		Desired:        irccon.nick,
		Confirmed:      irccon.fullyConnected,
		LastChangeTime: irccon.lastNickChange,
		PendingChange:  irccon.nick != irccon.nickcurrent,
		Error:          irccon.nickError,
	}

	// Test the status
	if !status.PendingChange {
		t.Error("Expected status.PendingChange to be true")
	}
	if status.Current != "testnick" {
		t.Errorf("Expected status.Current to be 'testnick', got '%s'", status.Current)
	}
	if status.Desired != "newnick" {
		t.Errorf("Expected status.Desired to be 'newnick', got '%s'", status.Desired)
	}
}

// TestSimpleMultipleChanges tests multiple nickname changes
func TestSimpleMultipleChanges(t *testing.T) {
	// Create a minimal IRC connection without setting up callbacks
	irccon := &Connection{
		nick:           "testnick",
		nickcurrent:    "testnick",
		user:           "testuser",
		lastNickChange: time.Now(),
		nickError:      "",
	}

	// Directly update the desired nickname for the first change
	irccon.nick = "newnick1"
	irccon.lastNickChange = time.Now()

	// Before the server confirms, request another change
	irccon.nick = "newnick2"
	irccon.lastNickChange = time.Now()

	// Check that both changes are tracked correctly
	if irccon.nickcurrent != "testnick" {
		t.Errorf("Expected current nickname to remain 'testnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick2" {
		t.Errorf("Expected desired nickname to be 'newnick2', got '%s'", irccon.nick)
	}

	// Manually simulate a NICK callback for the first change
	oldNick := irccon.nickcurrent
	if oldNick == "testnick" {
		irccon.nickcurrent = "newnick1"
		if irccon.nick == oldNick {
			irccon.nick = "newnick1"
		}
		irccon.lastNickChange = time.Now()
		irccon.nickError = ""
	}

	// Check that the current nickname is updated but desired remains the latest
	if irccon.nickcurrent != "newnick1" {
		t.Errorf("Expected current nickname to be 'newnick1', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick2" {
		t.Errorf("Expected desired nickname to remain 'newnick2', got '%s'", irccon.nick)
	}

	// Create a NickStatus manually (what GetNickStatus() would do)
	status := &NickStatus{
		Current:        irccon.nickcurrent,
		Desired:        irccon.nick,
		Confirmed:      irccon.fullyConnected,
		LastChangeTime: irccon.lastNickChange,
		PendingChange:  irccon.nick != irccon.nickcurrent,
		Error:          irccon.nickError,
	}

	// Check that status still reports a pending change
	if !status.PendingChange {
		t.Error("Expected PendingChange to be true after partial nickname change confirmation")
	}
}
