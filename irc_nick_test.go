package irc

import (
	"strings"
	"testing"
	"time"
)

func TestNickChangeConfirmationUsesIRCCasemapping(t *testing.T) {
	irccon := IRC("[Test]", "testuser")
	irccon.fullyConnected = true
	irccon.nick = "newnick"
	irccon.nickcurrent = "[Test]"
	irccon.nickError = "Some error"

	event, err := parseToEvent(":{tESt}!testuser@host NICK newnick")
	if err != nil {
		t.Fatalf("parseToEvent failed: %v", err)
	}
	event.Connection = irccon
	irccon.RunCallbacks(event)

	if irccon.nickcurrent != "newnick" {
		t.Fatalf("expected current nick to be updated to newnick, got %q", irccon.nickcurrent)
	}
	if irccon.nick != "newnick" {
		t.Fatalf("expected desired nick to stay at newnick, got %q", irccon.nick)
	}
	if irccon.nickError != "" {
		t.Fatalf("expected nick error to be cleared, got %q", irccon.nickError)
	}
}

func TestErroneousNicknameRecoveryUsesSafeAlternative(t *testing.T) {
	irccon := IRC("123 bad nick", "testuser")
	irccon.pwrite = make(chan string, 1)
	irccon.nickcurrent = "123 bad nick"

	irccon.RunCallbacks(&Event{
		Code:      "432",
		Arguments: []string{"server", "123 bad nick", "Erroneous nickname"},
	})

	alternative := generateAlternativeNick("123 bad nick")
	if irccon.nick != alternative {
		t.Fatalf("expected desired nick to switch to %q, got %q", alternative, irccon.nick)
	}
	if irccon.nickPending != alternative {
		t.Fatalf("expected pending nick to be %q, got %q", alternative, irccon.nickPending)
	}
	if irccon.nickcurrent != alternative {
		t.Fatalf("expected current nick to move to %q during registration recovery, got %q", alternative, irccon.nickcurrent)
	}
	if !isValidRFCNick(alternative) {
		t.Fatalf("expected alternative nick %q to be RFC-valid", alternative)
	}

	select {
	case got := <-irccon.pwrite:
		want := "NICK " + alternative + "\r\n"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	default:
		t.Fatal("expected a recovery NICK command to be sent")
	}
}

func TestNicknameInUseMatchingUsesIRCCasemapping(t *testing.T) {
	irccon := IRC("[Nick]", "testuser")
	irccon.pwrite = make(chan string, 1)
	irccon.nickcurrent = "[Nick]"

	irccon.RunCallbacks(&Event{
		Code:      "433",
		Arguments: []string{"server", "{nIcK}", "Nickname already in use"},
	})

	alternative := generateAlternativeNick("{nIcK}")
	if irccon.nick != "[Nick]" {
		t.Fatalf("expected desired nick to remain original for later retry, got %q", irccon.nick)
	}
	if irccon.nickPending != alternative {
		t.Fatalf("expected pending nick to be %q, got %q", alternative, irccon.nickPending)
	}
	if irccon.nickcurrent != alternative {
		t.Fatalf("expected current nick to track registration fallback %q, got %q", alternative, irccon.nickcurrent)
	}
}

func TestPostRegistrationAlternativeNickBecomesDesired(t *testing.T) {
	irccon := IRC("wanted", "testuser")
	irccon.fullyConnected = true
	irccon.nick = "wanted"
	irccon.nickcurrent = "current"
	irccon.pwrite = make(chan string, 1)

	irccon.RunCallbacks(&Event{
		Code:      "433",
		Arguments: []string{"server", "wanted", "Nick unavailable"},
	})

	alternative := generateAlternativeNick("wanted")
	select {
	case got := <-irccon.pwrite:
		want := "NICK " + alternative + "\r\n"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	default:
		t.Fatal("expected a recovery NICK command to be sent")
	}

	event, err := parseToEvent(":current!testuser@host NICK " + alternative)
	if err != nil {
		t.Fatalf("parseToEvent failed: %v", err)
	}
	event.Connection = irccon
	irccon.RunCallbacks(event)

	if irccon.nick != alternative {
		t.Fatalf("expected desired nick to follow confirmed alternative %q, got %q", alternative, irccon.nick)
	}
	if irccon.nickcurrent != alternative {
		t.Fatalf("expected current nick to be %q, got %q", alternative, irccon.nickcurrent)
	}
}

func TestPostRegistrationNickRecoveryCanBeDisabled(t *testing.T) {
	for _, code := range []string{"432", "433", "436", "437"} {
		t.Run(code, func(t *testing.T) {
			irccon := IRC("wanted", "testuser")
			irccon.AutoNickRecoveryPostRegistration = false
			irccon.fullyConnected = true
			irccon.nick = "wanted"
			irccon.nickcurrent = "current"
			irccon.nickPending = "wanted"
			irccon.nickChangeInProgress = true
			irccon.pwrite = make(chan string, 1)

			irccon.RunCallbacks(&Event{
				Code:      code,
				Arguments: []string{"server", "wanted", "Nick unavailable"},
			})

			select {
			case got := <-irccon.pwrite:
				t.Fatalf("expected no outbound NICK command, got %q", got)
			default:
			}

			if irccon.nick != "current" {
				t.Fatalf("expected desired nick to fall back to current, got %q", irccon.nick)
			}
			if irccon.nickcurrent != "current" {
				t.Fatalf("expected current nick to remain current, got %q", irccon.nickcurrent)
			}
			if irccon.nickPending != "" {
				t.Fatalf("expected pending nick to remain empty, got %q", irccon.nickPending)
			}
			if irccon.nickChangeInProgress {
				t.Fatal("expected nick change state to be cleared")
			}
		})
	}
}

func TestPingLoopDoesNotRetryDesiredNick(t *testing.T) {
	irccon := IRC("wanted", "testuser")
	irccon.PingFreq = 5 * time.Millisecond
	irccon.pwrite = make(chan string, 20)
	irccon.end = make(chan struct{})
	irccon.nick = "wanted"
	irccon.nickcurrent = "current"

	irccon.Add(1)
	go irccon.pingLoop()
	defer func() {
		close(irccon.end)
		irccon.Wait()
	}()

	gotPing := false
	deadline := time.After(30 * time.Millisecond)
	for {
		select {
		case got := <-irccon.pwrite:
			if strings.HasPrefix(got, "NICK ") {
				t.Fatalf("expected pingLoop not to send NICK, got %q", got)
			}
			if strings.HasPrefix(got, "PING ") {
				gotPing = true
			}
		case <-deadline:
			if !gotPing {
				t.Fatal("expected pingLoop to send PING")
			}
			return
		}
	}
}

func TestPreRegistrationNickRecoveryIgnoresPostRegistrationDisable(t *testing.T) {
	for _, code := range []string{"433", "437"} {
		t.Run(code, func(t *testing.T) {
			irccon := IRC("wanted", "testuser")
			irccon.AutoNickRecoveryPostRegistration = false
			irccon.fullyConnected = false
			irccon.nickcurrent = "wanted"
			irccon.pwrite = make(chan string, 1)

			irccon.RunCallbacks(&Event{
				Code:      code,
				Arguments: []string{"server", "wanted", "Nick unavailable"},
			})

			alternative := generateAlternativeNick("wanted")
			select {
			case got := <-irccon.pwrite:
				want := "NICK " + alternative + "\r\n"
				if got != want {
					t.Fatalf("expected %q, got %q", want, got)
				}
			default:
				t.Fatal("expected a recovery NICK command to be sent")
			}

			if irccon.nickPending != alternative {
				t.Fatalf("expected pending nick to be %q, got %q", alternative, irccon.nickPending)
			}
			if irccon.nickcurrent != alternative {
				t.Fatalf("expected current nick to track registration fallback %q, got %q", alternative, irccon.nickcurrent)
			}
		})
	}
}

func TestRestrictedNicknameStopsFurtherRetry(t *testing.T) {
	irccon := IRC("wanted", "testuser")
	irccon.fullyConnected = true
	irccon.nickcurrent = "current"
	irccon.nick = "wanted"
	irccon.nickPending = "wanted"
	irccon.nickChangeInProgress = true

	irccon.RunCallbacks(&Event{Code: "484", Arguments: []string{"server", "wanted", "Restricted"}})

	if irccon.nick != "current" {
		t.Fatalf("expected desired nick to fall back to current nick, got %q", irccon.nick)
	}
	if irccon.nickPending != "" {
		t.Fatalf("expected pending nick to be cleared, got %q", irccon.nickPending)
	}
	if irccon.nickChangeInProgress {
		t.Fatal("expected nick change retry state to be cleared")
	}
}

func isValidRFCNick(nick string) bool {
	if nick == "" || len(nick) > maxRFCNickLen {
		return false
	}
	if !isRFCNickFirstChar(nick[0]) {
		return false
	}
	for i := 1; i < len(nick); i++ {
		if !isRFCNickChar(nick[i]) {
			return false
		}
	}
	return true
}
