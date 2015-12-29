package cli

import (
	"log"
	"net"

	"telnet"
)

func ListenTelnet(addr string, cliServer *Server) {

	handler := func(conn net.Conn) {
		handleTelnet(conn, cliServer)
	}

	telnetServer := telnet.Server{Addr: addr, Handler: handler}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}

func handleTelnet(conn net.Conn, cliServer *Server) {
	defer conn.Close()

	log.Printf("handleTelnet: new telnet connection from: %s", conn.RemoteAddr())

	charMode(conn)
	windowSize(conn)

	cliClient := NewClient(conn)

	// mock user input in order to get server MOTD response
	cliServer.CommandChannel <- Command{Client: cliClient, Cmd: "", IsLine: true}

	notifyAppInputClosed := true

	go InputLoop(cliServer, cliClient, notifyAppInputClosed)

	OutputLoop(cliClient)

	log.Printf("handleTelnet: terminating connection: remote=%s", conn.RemoteAddr())
}

func charMode(conn net.Conn) {
	log.Printf("charMode: entering telnet character mode")
	cmd := []byte{cmdIAC, cmdWill, optEcho, cmdIAC, cmdWill, optSupressGoAhead, cmdIAC, cmdDont, optLinemode}
	if wr, err := conn.Write(cmd); err != nil {
		log.Printf("charMode: len=%d err=%v", wr, err)
	}
}

func windowSize(conn net.Conn) {
	log.Printf("windowSize: requesting telnet window size")
	cmd := []byte{cmdIAC, cmdDo, optNaws}
	if wr, err := conn.Write(cmd); err != nil {
		log.Printf("windowSize: len=%d err=%v", wr, err)
	}
}
