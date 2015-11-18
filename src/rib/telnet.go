package main

import (
	"cli"
	"log"
	"net"
	//"time"

	"telnet"
)

var cliServer *cli.Server

func listenTelnet(addr string) {
	cliServer = &cli.Server{}

	telnetServer := telnet.Server{Addr: addr, Handler: handleTelnet}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	log.Printf("handleTelnet: new telnet connection from: %s", conn.RemoteAddr())

	cliClient := cli.NewClient(conn)

	go cli.InputLoop(cliServer, cliClient)

	cli.OutputLoop(cliServer, cliClient)

	log.Printf("handleTelnet: terminating connection: remote=%s", conn.RemoteAddr())
}
