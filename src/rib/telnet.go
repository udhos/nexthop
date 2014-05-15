package main

import (
	"bufio"
	"log"
	"net"

	"telnet"
)

func inputLoop(rd *bufio.Reader) {
	log.Printf("FIXME WRITEME inputLoop")
}

func outputLoop(wr *bufio.Writer) {
	log.Printf("FIXME WRITEME outputLoop")
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	rd, wr := bufio.NewReader(conn), bufio.NewWriter(conn)

	//create userOut channel: will send messages to user

	//create cli interpreter: will write to userOut channel when needed

	//go routine loop:
	//	- read from userOut channel and write into wr
	//	- watch quitOutput channel
	go inputLoop(rd)

	//loop:
	//	- read from rd and feed into cli interpreter
	//	- watch idle timeout
	//	- watch quitInput channel
	outputLoop(wr)
}

func listenTelnet(addr string) {
	telnetServer := telnet.Server{Addr: addr, Handler: handleTelnet}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}
