package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	irc "github.com/kofany/go-ircevo"
)

// probeServer tries to connect, negotiates CAP, sends registration and disconnects.
func probeServer(name, addr string, wg *sync.WaitGroup) {
	defer wg.Done()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[%s] bad addr %q: %v", name, addr, err)
		return
	}

	tlsPort := port == "6697" || strings.Contains(name, "-tls")

	nick := fmt.Sprintf("probe_%d", time.Now().UnixNano()%100000)
	conn := irc.IRC(nick, nick)
	conn.Debug = true
	// For probe, avoid reconnect loops on throttling networks
	conn.SmartErrorHandling = false
	conn.HandleErrorAsDisconnect = true

	if tlsPort {
		conn.UseTLS = true
		conn.TLSConfig = &tls.Config{ServerName: host}
	}

	done := make(chan struct{})
	connected := make(chan struct{}, 1)

	conn.AddCallback("001", func(e *irc.Event) {
		log.Printf("[%s] CONNECTED: nick=%s", name, e.Arguments[0])
		select { // signal connected once
		case connected <- struct{}{}:
		default:
		}
		conn.QuitMessage = "probe-ok"
		go func() {
			time.Sleep(500 * time.Millisecond)
			conn.Quit()
		}()
	})

	conn.AddCallback("ERROR", func(e *irc.Event) {
		log.Printf("[%s] ERROR: %s", name, e.Message())
		// Stop probing this server to avoid throttling / reconnect storms
		go conn.Disconnect()
	})

	conn.AddCallback("DISCONNECTED", func(e *irc.Event) {
		log.Printf("[%s] DISCONNECTED", name)
		select {
		case <-done:
		default:
			close(done)
		}
	})

	log.Printf("[%s] Dialing %s", name, addr)
	if err := conn.Connect(addr); err != nil {
		log.Printf("[%s] CONNECT FAIL: %v", name, err)
		return
	}

	// Run the loop in background and wait for disconnect or timeout
	go conn.Loop()

	// If we don't get 001 within 20s, stop probing this server politely
	connectDeadline := time.NewTimer(20 * time.Second)

	select {
	case <-connected:
		// wait for graceful disconnect or 30s max
		select {
		case <-done:
			log.Printf("[%s] Finished", name)
		case <-time.After(30 * time.Second):
			log.Printf("[%s] Timeout after connect; forcing disconnect", name)
			conn.Disconnect()
		}
	case <-connectDeadline.C:
		log.Printf("[%s] No 001 within 20s; aborting probe", name)
		conn.Disconnect()
		<-done
	}
}

func main() {
	servers := map[string]string{
		"irc.al":            "irc.al:6667",
		"irc.atw-inter.net": "irc.atw-inter.net:6667",
		"irc.dal.net-plain": "irc.dal.net:6667",
		"irc.dal.net-tls":   "irc.dal.net:6697",
		"libera.chat-plain": "irc.libera.chat:6667",
		"libera.chat-tls":   "irc.libera.chat:6697",
	}

	var wg sync.WaitGroup
	wg.Add(len(servers))
	for name, addr := range servers {
		go probeServer(name, addr, &wg)
	}
	wg.Wait()
	log.Printf("All probes finished")
}
