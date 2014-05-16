package main

import (
	"bufio"
	"log"
	"net"

	"telnet"
)

type TelnetClient struct {
	rd      *bufio.Reader
	wr      *bufio.Writer
	userOut chan string // outputLoop: read from userOut and write into wr
}

type Command struct {
	client *TelnetClient
	line   string
}

var cmdInput = make(chan Command)

func inputLoop(client *TelnetClient) {
	//loop:
	//	- read from rd and feed into cli interpreter
	//	- watch idle timeout
	//	- watch quitInput channel

	scanner := bufio.NewScanner(client.rd)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("inputLoop: [%v]", line)
		cmdInput <- Command{client, line}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("inputLoop: %v", err)
	}

	log.Printf("inputLoop: exiting")
}

func outputLoop(client *TelnetClient) {
	//loop:
	//	- read from userOut channel and write into wr
	//	- watch quitOutput channel

	for msg := range client.userOut {
		if n, err := client.wr.WriteString(msg); err != nil {
			log.Printf("outputLoop: written=%d from=%d: %v", n, len(msg), err)
		}
		if err := client.wr.Flush(); err != nil {
			log.Printf("outputLoop: flush: %v", err)
		}
	}

	log.Printf("outputLoop: exiting")
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	rd, wr := bufio.NewReader(conn), bufio.NewWriter(conn)

	client := TelnetClient{rd, wr, make(chan string)}

	defer close(client.userOut)

	//create cli interpreter: will write to userOut channel when needed

	go inputLoop(&client)

	outputLoop(&client)
}

func listenTelnet(addr string) {
	telnetServer := telnet.Server{Addr: addr, Handler: handleTelnet}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}
