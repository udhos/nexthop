package main

import (
	"log"
	"net"

	"telnet"
)

func listenTelnet(addr string) {
	telnetServer := telnet.Server{Addr: addr, Handler: handleTelnet}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	log.Printf("handleTelnet: new telnet connection from: %s", conn.RemoteAddr())

	log.Printf("handleTelnet: terminating connection: remote=%s", conn.RemoteAddr())
}
