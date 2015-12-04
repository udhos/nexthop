package main

import (
	"cli"
	"log"
	"net"
	//"time"

	"telnet"
)

// https://tools.ietf.org/html/rfc854 TELNET PROTOCOL SPECIFICATION
const (
	cmdSE   = 240
	cmdNOP  = 241
	cmdSB   = 250
	cmdWill = 251
	cmdWont = 252
	cmdDo   = 253
	cmdDont = 254
	cmdIAC  = 255
)

const (
	optEcho           = 1
	optSupressGoAhead = 3
	optNaws           = 31 // rfc1073
	optLinemode       = 34
)

func listenTelnet(addr string, cliServer *cli.Server) {

	handler := func(conn net.Conn) {
		handleTelnet(conn, cliServer)
	}

	telnetServer := telnet.Server{Addr: addr, Handler: handler}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}

func handleTelnet(conn net.Conn, cliServer *cli.Server) {
	defer conn.Close()

	log.Printf("handleTelnet: new telnet connection from: %s", conn.RemoteAddr())

	charMode(conn)

	cliClient := cli.NewClient(conn)

	// mock user input in order to get server MOTD response
	cliServer.CommandChannel <- cli.Command{Client: cliClient, Cmd: "", IsLine: true}

	go cli.InputLoop(cliServer, cliClient)

	cli.OutputLoop(cliClient)

	log.Printf("handleTelnet: terminating connection: remote=%s", conn.RemoteAddr())
}

func charMode(conn net.Conn) {
	log.Printf("charMode: entering telnet character mode")
	cmd := []byte{cmdIAC, cmdWill, optEcho, cmdIAC, cmdWill, optSupressGoAhead, cmdIAC, cmdDont, optLinemode}
	if wr, err := conn.Write(cmd); err != nil {
		log.Printf("charMode: len=%d err=%v", wr, err)
	}
}
