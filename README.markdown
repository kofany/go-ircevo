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
