# IRC Nick Tracking RFC Compliance Audit Report

## Executive Summary

This document provides a comprehensive audit of the IRC library's nickname tracking implementation against RFC 1459 and RFC 2812 specifications. The audit identified several critical bugs and RFC non-compliance issues that have been fixed to ensure proper synchronization between the library's internal nick state and the actual nickname on the IRC server.

## RFC Requirements

### RFC 2812 Section 3.1.2 - Nick Message

**Initial Registration:**
- Client sends: `NICK nickname`
- Server confirms with: `001 nickname :Welcome...` (RPL_WELCOME)
- The first argument of 001 contains the actual confirmed nickname

**Nick Change After Registration:**
- Client sends: `NICK new_nickname`
- Server confirms with: `:old_nick!user@host NICK :new_nick`
- Only this NICK message format confirms the nick change

**Nick Error Codes (RFC 2812 Section 5.1):**
- **431 ERR_NONICKNAMEGIVEN**: No nickname given
- **432 ERR_ERRONEUSNICKNAME**: Erroneous nickname (invalid characters/format)
- **433 ERR_NICKNAMEINUSE**: Nickname is already in use
- **436 ERR_NICKCOLLISION**: Nickname collision during connection
- **437 ERR_UNAVAILRESOURCE**: Nickname temporarily unavailable
- **484 ERR_RESTRICTED**: User is restricted (cannot change nick)

**Critical RFC Requirements:**
1. Nick confirmation ONLY happens via 001 (initial) or NICK message (post-registration)
2. Error numerics do NOT confirm a nickname - they reject it
3. The NICK message format `:oldnick!user@host NICK :newnick` uses the OLD nick as e.Nick
4. Client must handle nick collisions gracefully and try alternatives

## Bugs Found and Fixed

### 1. **CRITICAL: Post-Registration Nick Tracking Bug**

**Location:** `irc_callback.go` line 436 (NICK callback)

**Issue:** The NICK callback was unconditionally updating `irc.nick` (desired nickname) to match the confirmed nickname. This broke the automatic retry mechanism for desired nicknames.

**Scenario:**
```
1. User calls Nick("alice")
   - nick = "alice" (desired)
   - nickcurrent = "bob" (confirmed)
2. Server responds with 433 (nick in use)
3. Library tries "alice_" (alternative)
4. Server confirms with `:bob NICK :alice_`
5. NICK callback updates BOTH:
   - nickcurrent = "alice_" ✓ CORRECT
   - nick = "alice_" ✗ BUG!
6. Now nick == nickcurrent, so pingLoop never retries "alice"
```

**Fix:** Modified NICK callback to only update `irc.nick` during initial registration (before 001). For post-registration changes, the desired nick is preserved so pingLoop can automatically retry.

```go
// RFC COMPLIANT: Only update desired nick during initial registration (before 001)
if !irc.fullyConnected {
    // During initial registration, accept whatever nick we get
    irc.nick = newNick
} else {
    // Post-registration: only update desired nick if we got exactly what we wanted
    if newNick == irc.nick {
        // Success! We got the nick we wanted
    } else {
        // We got an alternative (due to error recovery), keep retrying desired nick
    }
}
```

### 2. **CRITICAL: Error Handler State Corruption**

**Location:** `irc_callback.go` lines 272-276 (433 handler and similar in other error handlers)

**Issue:** Error handlers were modifying `nickcurrent` before server confirmation, violating the invariant that `nickcurrent` should only contain server-confirmed nicknames.

**Scenario:**
```
1. User connected as "bob_"
2. User calls Nick("alice")
   - nick = "alice", nickcurrent = "bob_"
3. Server sends 433 for "alice"
4. OLD CODE:
   - Sets nickcurrent = "alice" (not confirmed!)
   - Modifies nickcurrent to "alice_"
   - Sends NICK alice_
5. Server confirms with `:bob_ NICK :alice_`
6. NICK callback checks: e.Nick == nickcurrent
   - "bob_" == "alice_"? NO! Callback doesn't trigger!
7. Internal state is now corrupted
```

**Fix:** Introduced `nickPending` field to track pending nick changes without corrupting `nickcurrent`. Error handlers now:
1. Generate alternatives based on the rejected nickname
2. Store in `nickPending` (not `nickcurrent`)
3. During initial registration only, update `nickcurrent` as a staging value
4. After registration, `nickcurrent` remains the confirmed nick

```go
if attemptedNick == irc.nick || attemptedNick == irc.nickPending {
    alternative := generateAlternativeNick(attemptedNick)
    irc.nickPending = alternative
    irc.nickChangeInProgress = true
    irc.nickChangeTimeout = time.Now()
    
    // During initial registration (before 001), also update nickcurrent
    // to keep track of what we're trying, since we don't have a confirmed nick yet
    if !irc.fullyConnected && irc.nickcurrent == attemptedNick {
        irc.nickcurrent = alternative
    }
    
    irc.SendRawf("NICK %s", alternative)
}
```

### 3. **Race Condition in Nick Change Tracking**

**Issue:** The library wasn't properly tracking which nickname change was in progress, leading to potential race conditions when multiple nick change attempts occurred in quick succession.

**Fix:** Enhanced nick change tracking with:
- `nickPending`: The nickname currently pending confirmation from the server
- `nickChangeInProgress`: Boolean flag indicating if a change is in progress
- `nickChangeTimeout`: Timestamp to detect stale pending changes

All three fields are now properly synchronized across:
- Initial NICK command send
- Error handler alternative generation
- NICK confirmation callback
- Connection state transitions (Connect/Disconnect/Reconnect)

### 4. **Incomplete State Management in Connection Lifecycle**

**Issue:** The `nickPending` and `nickChangeInProgress` fields weren't being reset during connection state transitions, potentially causing stale state to affect subsequent connections.

**Fix:** Added proper state resets in:
- `Connect()`: Reset at connection start
- `Disconnect()`: Reset at disconnect
- `Reconnect()`: Reset before reconnect
- `001 handler`: Reset at successful registration

## Implementation Details

### New Fields Added to Connection Struct

```go
type Connection struct {
    // ... existing fields ...
    
    nick                   string // The nickname we want (desired)
    nickcurrent            string // The nickname we currently have (confirmed by server)
    nickPending            string // The nickname currently pending confirmation from the server
    
    // ... existing fields ...
}
```

### Nick State Machine

The library now properly implements a three-state nick tracking system:

1. **Desired Nick (`nick`)**: What the user wants
   - Set by `Nick()` function
   - Updated by 001 handler (accept initial registration result)
   - NOT updated by NICK callback after registration (to allow retries)

2. **Current Nick (`nickcurrent`)**: What the server has confirmed
   - Only updated by 001 (initial) or NICK callback (post-registration)
   - Never modified by error handlers (except during initial registration)
   - Always reflects the actual server-confirmed nickname

3. **Pending Nick (`nickPending`)**: What we're currently trying
   - Set when sending NICK command
   - Set by error handlers when trying alternatives
   - Cleared when NICK is confirmed or on disconnect

### Error Handling Flow

**Initial Registration (before 001):**
```
1. Send NICK desired_nick
2. If error 433/436/437:
   - Generate alternative
   - Update nickcurrent (used as staging area before confirmation)
   - Update nickPending
   - Send NICK alternative
3. Receive 001:
   - Set both nick and nickcurrent to confirmed value
   - Clear nickPending
```

**Post-Registration Nick Change:**
```
1. User calls Nick(desired_nick)
   - Set nick = desired_nick
   - Set nickPending = desired_nick  
   - Send NICK desired_nick
2. If error 433/436/437:
   - Generate alternative
   - Set nickPending = alternative
   - DO NOT modify nickcurrent (keep confirmed nick)
   - Send NICK alternative
3. If success:
   - Receive `:oldnick NICK :newnick`
   - Update nickcurrent = newnick
   - Clear nickPending
   - If newnick != nick, pingLoop will periodically retry desired nick
```

### Automatic Retry Mechanism

The `pingLoop` function now properly supports automatic retry of desired nicknames:

```go
// In pingLoop, every PingFreq interval:
if irc.nick != irc.nickcurrent {
    // We don't have our desired nick, try to get it
    irc.SendRawf("NICK %s", irc.nick)
}
```

This allows the library to:
- Accept alternatives during error recovery
- Keep track of the user's desired nickname
- Periodically retry to reclaim the desired nickname if it becomes available

## Testing

All changes have been validated to:
1. Compile without errors
2. Pass existing unit tests
3. Properly track nick state through various scenarios:
   - Initial registration with nick collision
   - Post-registration nick changes
   - Multiple consecutive nick errors
   - Reconnection scenarios

## RFC Compliance Summary

| RFC Requirement | Status | Notes |
|----------------|--------|-------|
| 001 confirms initial nick | ✅ COMPLIANT | Properly handled in 001 callback |
| NICK message confirms changes | ✅ COMPLIANT | Properly handled in NICK callback |
| Error codes don't confirm nicks | ✅ COMPLIANT | Fixed - errors no longer corrupt nickcurrent |
| e.Nick is OLD nick in NICK msg | ✅ COMPLIANT | Properly matched against nickcurrent |
| Handle 431-437, 484 errors | ✅ COMPLIANT | All error handlers updated |
| Try alternatives on collision | ✅ COMPLIANT | Implemented with generateAlternativeNick |
| Preserve desired nick | ✅ COMPLIANT | Nick field no longer overwritten post-registration |

## Recommendations

1. **For Library Users:**
   - Use `GetNick()` to get the current confirmed nickname
   - Use `GetNickStatus()` to see detailed nick state including pending changes
   - The library will automatically retry your desired nickname if it had to accept an alternative

2. **For Future Development:**
   - Consider adding a callback for "nick not available, using alternative"
   - Consider adding configurable retry interval for desired nick
   - Consider adding max retry limit for desired nick reclaim attempts

## Conclusion

The IRC library's nickname tracking implementation has been thoroughly audited against RFC 1459 and RFC 2812 specifications. Critical bugs that caused desynchronization between the library's internal state and the actual IRC server state have been identified and fixed. The implementation now properly:

1. Tracks three distinct nick states (desired, current, pending)
2. Only updates confirmed nick on server confirmation
3. Preserves user's desired nickname for automatic retry
4. Handles all RFC-specified error conditions correctly
5. Maintains consistency through connection lifecycle

The library is now fully RFC-compliant for nickname tracking and will always maintain synchronization with the IRC server.

---

**Audit Date:** 2025-11-03  
**Auditor:** AI Code Auditor  
**RFC Documents:** RFC 1459, RFC 2812  
**Files Modified:**
- `irc_struct.go`: Added nickPending field
- `irc_callback.go`: Fixed NICK callback and all error handlers
- `irc.go`: Updated Nick(), Connect(), Disconnect(), Reconnect(), pingLoop()
