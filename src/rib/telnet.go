package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"

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
	keyEscape    = 27
	keyBackspace = 127
)

const (
	optEcho           = 1
	optSupressGoAhead = 3
	optLinemode       = 34
)

const (
	ctrlA = 'A' - '@'
	ctrlB = 'B' - '@'
	ctrlC = 'C' - '@'
	ctrlD = 'D' - '@'
	ctrlE = 'E' - '@'
	ctrlF = 'F' - '@'
	ctrlH = 'H' - '@'
	ctrlK = 'K' - '@'
	ctrlN = 'N' - '@'
	ctrlP = 'P' - '@'
	ctrlZ = 'Z' - '@'
)

const (
	escNone = iota
	escOne  = iota
	escTwo  = iota
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
	hist       []string
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
		log.Printf("charReadLoop: read len=%d [%v]", rd, curr)
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

func histPrevious() {
	log.Printf("hist: previous")
}

func histNext() {
	log.Printf("hist: next")
}

func inputLoop(client *TelnetClient) {
	//loop:
	//	- read from rd and feed into cli interpreter
	//	- watch idle timeout
	//	- watch quitInput channel

	escape := escNone
	iac := IAC_NONE
	buf := [30]byte{} // underlying buffer
	//line := buf[:0]   // position at underlying buffer
	size := 0 // position at underlying buffer

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

				switch escape {
				case escNone:
					// proceed below
				case escOne:
					switch b {
					case '[':
						escape = escTwo
					default:
						escape = escNone
					}
					continue
				case escTwo:
					switch b {
					case 'A':
						histPrevious()
					case 'B':
						histNext()
					case 'C':
						log.Printf("inputLoop: RIGHT")
					case 'D':
						log.Printf("inputLoop: LEFT")
					}
					escape = escNone
					continue
				default:
					log.Printf("inputLoop: bad escape status: %d", escape)
				}

				switch {
				case b == cmdIAC:
					// hit IAC mark?
					log.Printf("inputLoop: telnet IAC begin")
					iac = IAC_CMD

				case b == keyBackspace, b < 32:

					// handle control char

					switch b {
					case '\r':
						// discard
					case '\n':
						//cmdLine := string(line) // string is safe for sharing (immutable)
						cmdLine := string(buf[:size]) // string is safe for sharing (immutable)
						log.Printf("inputLoop: cmdLine len=%d [%s]", len(cmdLine), cmdLine)
						cmdInput <- Command{client, cmdLine}

						// save into history
						if len(strings.TrimSpace(cmdLine)) > 0 {
							hlen := len(client.hist)
							if hlen >= 10 {
								i := 0
								copy(client.hist[i:], client.hist[i+1:]) // left shift at i
								client.hist = client.hist[:hlen-1]       // shrink
							}
							client.hist = append(client.hist, cmdLine) // push

							log.Printf("history: size=%d %v", len(client.hist), client.hist)
						}

						//line = buf[:0] // reset reading buffer position
						size = 0 // reset reading buffer position

						// echo newline back to client
						if client.serverEcho {
							client.userOut <- "\r\n"
						}
					case ctrlH, keyBackspace:
						// backspace
						if size <= 0 {
							continue
						}
						size--
						// echo backspace to client
						if client.serverEcho {
							client.userOut <- string(byte(keyBackspace))
						}
					case keyEscape:
						escape = escOne
					case ctrlP:
						histPrevious()
					case ctrlN:
						histNext()
					default:
						log.Printf("inputLoop: unknown control: %d 0x%x", b, b)
					}

					continue

				default:
					// push non-commands bytes into line buffer

					if size >= len(buf) {
						client.userOut <- fmt.Sprintf("\r\nline buffer overflow: size=%d max=%d\r\n", size, len(buf))
						client.userOut <- string(buf[:size]) // redisplay command to user
						continue LOOP
					}

					//line = append(buf[:len(line)], b)
					buf[size] = b
					size++

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

		log.Printf("inputLoop: buf len=%d [%s]", size, buf[:size])
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

	client := TelnetClient{conn, bufio.NewWriter(conn), make(chan string), make(chan int), make(chan bool), MOTD, true, []string{}}

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
