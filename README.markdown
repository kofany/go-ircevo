
# Important Update: Handling `ERROR` Events as Disconnects

In the latest release of our IRC library, we've introduced a new feature that changes how the library handles `ERROR` events received from the IRC server.

## New Feature: `HandleErrorAsDisconnect` Flag

We've added a new boolean flag to the `Connection` struct called `HandleErrorAsDisconnect`. This flag allows you to control how the library reacts when an `ERROR` event is received from the server.

- **Default Value in Latest Release:** `true`
- **Behavior When `true`:**
  - The library treats `ERROR` events as connection errors.
  - It **does not automatically attempt to reconnect** after receiving an `ERROR`.
  - You have the flexibility to implement your own reconnection logic in response to `ERROR` events.
- **Behavior When `false`:**
  - The library behaves as in previous releases.
  - It may automatically attempt to reconnect after an `ERROR`, potentially leading to reconnection loops.

## Impact on Existing Bots and Applications

Since the default value is now set to `true`, existing bots and applications using this library may experience changes in behavior:

- **If your application relies on automatic reconnection after an `ERROR`:**
  - You'll need to explicitly set `HandleErrorAsDisconnect` to `false` in your code to retain the previous behavior.
  - Alternatively, consider using the previous release of the library.

## How to Set the `HandleErrorAsDisconnect` Flag

To adjust the behavior of your application, set the `HandleErrorAsDisconnect` flag when initializing your IRC connection:

```go
irccon := irc.IRC("nick", "user")
// Set to false to retain previous behavior
irccon.HandleErrorAsDisconnect = false
```

## Recommendation for Users

- **For Users Who Want the New Behavior:**
  - No action is needed. The library will, by default, treat `ERROR` events as disconnects and will not automatically reconnect.
  - Implement your own logic to handle reconnections if desired.

- **For Users Who Prefer the Previous Behavior:**
  - Set `HandleErrorAsDisconnect` to `false` in your code.
  - Alternatively, you can continue using the previous release of the library.

## Example: Custom Reconnection Logic

If you choose to handle reconnections manually, you can add a callback for the `ERROR` event:

```go
irccon := irc.IRC("nick", "user")
irccon.HandleErrorAsDisconnect = true

irccon.AddCallback("ERROR", func(e *irc.Event) {
    irccon.Log.Printf("Received ERROR: %s", e.Message())
    // Implement your reconnection logic here
    time.Sleep(30 * time.Second)
    err := irccon.Reconnect()
    if err != nil {
        irccon.Log.Printf("Reconnect failed: %s", err)
    }
})
```

## Summary

- **New Feature:** `HandleErrorAsDisconnect` flag added.
- **Default Behavior Changed:** The library now treats `ERROR` events as disconnects by default.
- **Action Required:** If you rely on the old behavior, set `HandleErrorAsDisconnect` to `false` or use the previous release.

We believe this change will give developers better control over how their applications handle server `ERROR` events, preventing unwanted reconnection loops and allowing for more robust error handling.

If you have any questions or need assistance with this update, please feel free to reach out.

# go-ircevo

go-ircevo is an evolved and extended version of the original [go-ircevent](https://github.com/thoj/go-ircevent) library by Thomas Jager. This library provides an enhanced framework for interacting with IRC servers, supporting additional features like DCC, SASL authentication, proxy support, and more.

Originally, this library started as a fork of go-ircevent, but it has been significantly expanded to meet the needs I encountered while developing my own IRC bot. Due to the number of changes and the direction I've taken, I'm continuing this project under a new name, go-ircevo. Nevertheless, I intend to keep contributing important features back to the original go-ircevent fork, ensuring backward compatibility so that they can be integrated into the original library if desired.

## Features

- Support for direct client-to-client (DCC) communication
- Enhanced IRC command handling
- SASL authentication support
- Proxy integration for IRC connections
- Extended message handling
- Compatible with the original go-ircevent API for easy migration

## Installation

```bash
go get github.com/kofany/go-ircevo
```

## Usage

Here is an example of how to use go-ircevo to connect to an IRC server:

```go
package main

import (
    "log"
    "github.com/kofany/go-ircevo"
)

func main() {
    conn := ircevo.IRC("nickname", "username")
    err := conn.Connect("irc.freenode.net:6667")
    if err != nil {
        log.Fatal(err)
    }

    conn.AddCallback("001", func(e *ircevo.Event) {
        conn.Join("#channel")
    })

    conn.Loop()
}
```

## License

This project is licensed under the BSD license, based on the original go-ircevent [library](https://github.com/thoj/go-ircevent) by Thomas Jager. Please see the LICENSE file for more details.
