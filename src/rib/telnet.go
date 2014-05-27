package main

import (
	"bufio"
	"log"
	"net"

	"telnet"
)

const (
	cmdSE   = 240
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
	optLinemode       = 34
)

const (
	IAC_NONE = iota
	IAC_CMD  = iota
	IAC_OPT  = iota
	IAC_SUB  = iota
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
	echo       chan bool
	status     int
	serverEcho bool
}

type Command struct {
	client *TelnetClient
	line   string
}

var cmdInput = make(chan Command)

func charReadLoop(conn net.Conn, read chan<- byte) {
	input := make([]byte, 10) // last input

	for {
		rd, err := conn.Read(input)
		if err != nil {
			log.Printf("charReadLoop: net.Read: %v", err)
			break
		}
		curr := input[:rd]
		log.Printf("charReadLoop: read len=%d [%s]", rd, curr)
		for _, b := range curr {
			read <- b
		}
	}

	log.Printf("charReadLoop: exiting")

	close(read)
}

func reader(conn net.Conn) <-chan byte {
	read := make(chan byte)
	go charReadLoop(conn, read)
	return read
}

func inputLoop(client *TelnetClient) {
	//loop:
	//	- read from rd and feed into cli interpreter
	//	- watch idle timeout
	//	- watch quitInput channel

	iac := IAC_NONE
	buf := [30]byte{} // underlying buffer
	line := buf[:0]   // position at underlying buffer

	read := reader(client.conn)

LOOP:
	for {
		select {
		case client.serverEcho = <-client.echo:
			// do nothing
		case b, ok := <-read:
			if !ok {
				log.Printf("inputLoop: closed channel")
				break LOOP
			}

			switch iac {
			case IAC_NONE:
				switch b {
				case cmdIAC:
					// hit IAC mark?
					log.Printf("inputLoop: telnet IAC begin")
					iac = IAC_CMD
					continue
				case '\r':
					cmdLine := string(line) // string is safe for sharing (immutable)
					log.Printf("inputLoop: cmdLine len=%d [%s]", len(cmdLine), cmdLine)
					cmdInput <- Command{client, cmdLine}
					line = buf[:0] // reset reading buffer position
				default:
					// push non-commands bytes into line buffer
					line = append(buf[:len(line)], b)

					// echo char back to client
					if client.serverEcho {
						client.userOut <- string(b)
					}
				}

			case IAC_CMD:

				switch b {
				case cmdSB:
					log.Printf("inputLoop: telnet SUB begin")
					iac = IAC_SUB
				case cmdWill, cmdWont, cmdDo, cmdDont:
					log.Printf("inputLoop: telnet OPT begin")
					iac = IAC_OPT
				default:
					log.Printf("inputLoop: telnet IAC end")
					iac = IAC_NONE
				}

			case IAC_OPT:

				log.Printf("inputLoop: telnet OPT end")
				log.Printf("inputLoop: telnet IAC end")
				iac = IAC_NONE

			case IAC_SUB:

				if b == cmdSE {
					log.Printf("inputLoop: telnet SUB end")
					log.Printf("inputLoop: telnet IAC end")
					iac = IAC_NONE
				}

			default:
				log.Panicf("inputLoop: unexpected state iac=%d", iac)
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

	client := TelnetClient{conn, bufio.NewWriter(conn), make(chan string), make(chan int), make(chan bool), MOTD, true}

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
