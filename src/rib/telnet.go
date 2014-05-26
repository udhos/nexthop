package main

import (
	"bufio"
	"log"
	"net"

	"telnet"
)

const (
	cmdWill = 251
	cmdWont = 252
	cmdDo   = 253
	cmdDont = 254
	cmdIAC  = 255
)

const (
	optEcho           = 1
	optSupressGoAhead = 3
	optLinemode       = 34
)

const (
	QUIT = iota
	MOTD = iota
	USER = iota
	PASS = iota
	EXEC = iota
	ENAB = iota
	CONF = iota
)

type TelnetClient struct {
	//rd      *bufio.Reader
	conn       net.Conn
	wr         *bufio.Writer
	userOut    chan string // outputLoop: read from userOut and write into wr
	quit       chan int
	status     int
	serverEcho bool
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

	/*
		scanner := bufio.NewScanner(client.rd)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("inputLoop: [%v]", line)
			cmdInput <- Command{client, line}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("inputLoop: %v", err)
		}
	*/

	iac := false
	opt := false
	buf := [30]byte{} // underlying buffer
	line := buf[:0] // position at underlying buffer
	input := make([]byte, 10) // last input

	for {
		rd, err := client.conn.Read(input)
		if err != nil {
			log.Printf("inputLoop: net.Read: %v", err)
			break
		}
		curr := input[:rd]
		log.Printf("inputLoop: read len=%d [%s]", rd, curr)

		for _, b := range curr {
			if iac {
				// consume telnet commands
				if opt {
					opt = false
				} else {
					switch b {
					case cmdWill, cmdWont, cmdDo, cmdDont:
						opt = true
					}
				}
			} else {
				if b == cmdIAC {
					iac = true
				} else {
					// push non-commands bytes into line buffer
					line = append(buf[:len(line)], b)
				}
			}
		}

		log.Printf("inputLoop: buf len=%d [%s]", len(line), line)
	}

	log.Printf("inputLoop: exiting")
}

func outputLoop(client *TelnetClient) {
	//loop:
	//	- read from userOut channel and write into wr
	//	- watch quitOutput channel

LOOP:
	for {
		select {
		case msg := <-client.userOut:
			if n, err := client.wr.WriteString(msg); err != nil {
				log.Printf("outputLoop: written=%d from=%d: %v", n, len(msg), err)
			}
			if err := client.wr.Flush(); err != nil {
				log.Printf("outputLoop: flush: %v", err)
			}
		case _, ok := <-client.quit:
			if !ok {
				break LOOP
			}
		}
	}

	log.Printf("outputLoop: exiting")
}

func charMode(conn net.Conn) {
	cmd := []byte{cmdIAC, cmdWill, optEcho, cmdIAC, cmdWill, optSupressGoAhead, cmdIAC, cmdDont, optLinemode}

	wr, err := conn.Write(cmd)

	log.Printf("charMode: len=%d err=%v", wr, err)
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	log.Printf("new telnet connection from: %s", conn.RemoteAddr())

	//rd, wr := bufio.NewReader(conn), bufio.NewWriter(conn)

	charMode(conn)

	client := TelnetClient{conn, bufio.NewWriter(conn), make(chan string), make(chan int), MOTD, true}

	defer close(client.userOut)

	cmdInput <- Command{&client, ""} // mock user input

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
