package irc

import (
	"testing"
	"time"
)

func TestGetNickStatus(t *testing.T) {
	// Create a new IRC connection
	irccon := IRC("testnick", "testuser")
	if irccon == nil {
		t.Fatal("Failed to create IRC connection")
	}

	// Test initial state
	status := irccon.GetNickStatus()
	if status.Current != "testnick" {
		t.Errorf("Expected current nickname to be 'testnick', got '%s'", status.Current)
	}
	if status.Desired != "testnick" {
		t.Errorf("Expected desired nickname to be 'testnick', got '%s'", status.Desired)
	}
	if status.Confirmed {
		t.Error("Expected confirmed to be false initially")
	}
	if status.PendingChange {
		t.Error("Expected pending change to be false initially")
	}
	if status.Error != "" {
		t.Errorf("Expected error to be empty initially, got '%s'", status.Error)
	}

	// Test changing the nickname
	irccon.nick = "newnick"
	status = irccon.GetNickStatus()
	if status.Current != "testnick" {
		t.Errorf("Expected current nickname to remain 'testnick', got '%s'", status.Current)
	}
	if status.Desired != "newnick" {
		t.Errorf("Expected desired nickname to be 'newnick', got '%s'", status.Desired)
	}
	if !status.PendingChange {
		t.Error("Expected pending change to be true after changing desired nickname")
	}

	// Test setting an error
	irccon.nickError = "Nickname already in use"
	status = irccon.GetNickStatus()
	if status.Error != "Nickname already in use" {
		t.Errorf("Expected error to be 'Nickname already in use', got '%s'", status.Error)
	}

	// Test confirming the connection
	irccon.fullyConnected = true
	status = irccon.GetNickStatus()
	if !status.Confirmed {
		t.Error("Expected confirmed to be true after setting fullyConnected")
	}

	// Test last change time
	initialTime := irccon.lastNickChange
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time difference

	// Directly update the fields instead of calling Nick() to avoid network operations
	irccon.Lock()
	irccon.nick = "anothernick"
	irccon.lastNickChange = time.Now()
	irccon.Unlock()

	status = irccon.GetNickStatus()
	if !status.LastChangeTime.After(initialTime) {
		t.Error("Expected last change time to be updated after nickname change")
	}
}

func TestGetNickStatusUsesIRCCasemapping(t *testing.T) {
	irccon := IRC("[Nick]", "testuser")
	irccon.nickcurrent = "[Nick]"
	irccon.nick = "{nIcK}"

	status := irccon.GetNickStatus()
	if status.PendingChange {
		t.Fatalf("expected no pending change for RFC-equivalent nicknames, got %+v", status)
	}
}

func TestNickDoesNotSendForEquivalentIRCNick(t *testing.T) {
	irccon := IRC("[Nick]", "testuser")
	irccon.pwrite = make(chan string, 1)
	irccon.nickcurrent = "[Nick]"

	irccon.Nick("{nIcK}")

	select {
	case got := <-irccon.pwrite:
		t.Fatalf("expected no NICK command for equivalent nickname, got %q", got)
	default:
	}

	if irccon.nickPending != "" {
		t.Fatalf("expected no pending nick, got %q", irccon.nickPending)
	}
	if irccon.nickChangeInProgress {
		t.Fatal("expected no nick change in progress")
	}
}
