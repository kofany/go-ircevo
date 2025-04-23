package irc

import (
	"testing"
	"time"
)

func TestSimpleNickChange(t *testing.T) {
	// Create a new IRC connection
	irccon := IRC("testnick", "testuser")
	if irccon == nil {
		t.Fatal("Failed to create IRC connection")
	}

	// Test initial state
	if irccon.nickcurrent != "testnick" {
		t.Errorf("Expected current nickname to be 'testnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "testnick" {
		t.Errorf("Expected desired nickname to be 'testnick', got '%s'", irccon.nick)
	}

	// Call Nick() to request a nickname change
	irccon.Nick("newnick")

	// Verify that only the desired nickname was updated
	if irccon.nickcurrent != "testnick" {
		t.Errorf("Expected current nickname to remain 'testnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick" {
		t.Errorf("Expected desired nickname to be 'newnick', got '%s'", irccon.nick)
	}

	// Manually simulate a NICK callback
	irccon.Lock()
	oldNick := irccon.nickcurrent
	if oldNick == "testnick" {
		irccon.nickcurrent = "newnick"
		if irccon.nick == oldNick {
			irccon.nick = "newnick"
		}
		irccon.lastNickChange = time.Now()
		irccon.nickError = ""
	}
	irccon.Unlock()

	// Verify that both nicknames were updated
	if irccon.nickcurrent != "newnick" {
		t.Errorf("Expected current nickname to be updated to 'newnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick" {
		t.Errorf("Expected desired nickname to be 'newnick', got '%s'", irccon.nick)
	}

	// Test GetNickStatus
	status := irccon.GetNickStatus()
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

func TestSimpleNickError(t *testing.T) {
	// Create a new IRC connection
	irccon := IRC("testnick", "testuser")
	if irccon == nil {
		t.Fatal("Failed to create IRC connection")
	}

	// Call Nick() to request a nickname change
	irccon.Nick("newnick")

	// Manually simulate a 433 callback (nickname in use)
	irccon.Lock()
	irccon.nickError = "Nickname already in use"
	irccon.Unlock()

	// Test GetNickStatus
	status := irccon.GetNickStatus()
	if status.Error != "Nickname already in use" {
		t.Errorf("Expected status.Error to be 'Nickname already in use', got '%s'", status.Error)
	}
}

func TestSimplePendingChange(t *testing.T) {
	// Create a new IRC connection
	irccon := IRC("testnick", "testuser")
	if irccon == nil {
		t.Fatal("Failed to create IRC connection")
	}

	// Call Nick() to request a nickname change
	irccon.Nick("newnick")

	// Test GetNickStatus
	status := irccon.GetNickStatus()
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

func TestSimpleMultipleChanges(t *testing.T) {
	// Create a new IRC connection
	irccon := IRC("testnick", "testuser")
	if irccon == nil {
		t.Fatal("Failed to create IRC connection")
	}

	// Request first nickname change
	irccon.Nick("newnick1")

	// Before the server confirms, request another change
	irccon.Nick("newnick2")

	// Check that both changes are tracked correctly
	if irccon.nickcurrent != "testnick" {
		t.Errorf("Expected current nickname to remain 'testnick', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick2" {
		t.Errorf("Expected desired nickname to be 'newnick2', got '%s'", irccon.nick)
	}

	// Manually simulate a NICK callback for the first change
	irccon.Lock()
	oldNick := irccon.nickcurrent
	if oldNick == "testnick" {
		irccon.nickcurrent = "newnick1"
		if irccon.nick == oldNick {
			irccon.nick = "newnick1"
		}
		irccon.lastNickChange = time.Now()
		irccon.nickError = ""
	}
	irccon.Unlock()

	// Check that the current nickname is updated but desired remains the latest
	if irccon.nickcurrent != "newnick1" {
		t.Errorf("Expected current nickname to be 'newnick1', got '%s'", irccon.nickcurrent)
	}
	if irccon.nick != "newnick2" {
		t.Errorf("Expected desired nickname to remain 'newnick2', got '%s'", irccon.nick)
	}

	// Check that GetNickStatus() still reports a pending change
	status := irccon.GetNickStatus()
	if !status.PendingChange {
		t.Error("Expected PendingChange to be true after partial nickname change confirmation")
	}
}
