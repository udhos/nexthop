package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

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
	optNaws           = 31 // rfc1073
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
	IAC_NONE    = iota
	IAC_CMD     = iota
	IAC_OPT     = iota
	IAC_SUB     = iota
	IAC_SUB_IAC = iota
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
	conn        net.Conn
	wr          *bufio.Writer
	userOut     chan string // outputLoop: read from userOut and write into wr
	userFlush   chan int
	quitInput   chan int
	quitOutput  chan int
	echo        chan bool
	sendLine    chan bool
	onlyLine    bool
	status      int
	serverEcho  bool
	hist        []string
	outputQueue []string
	autoHeight  int
}

type Command struct {
	client *TelnetClient
	line   string
}

type SetHeight struct {
	client *TelnetClient
	height int
}

var cmdInput = make(chan Command)
var keyInput = make(chan Command)
var inputClosed = make(chan TelnetClient)
var autoHeight = make(chan SetHeight)

func charReadLoop(conn net.Conn, read chan<- byte) {

	// when both inputLoop and outputLoop exit,
	// the connection is closed, terminating us.
	// see handleTelnet()

	// we are the only one sender on this channel.
	// so we can use the channel close idiom for
	// signaling EOF.
	defer close(read)

	log.Printf("charReadLoop: starting")

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
}

func reader(conn net.Conn) <-chan byte {
	read := make(chan byte)
	go charReadLoop(conn, read)
	return read
}

func histPrevious() {
	log.Printf("histPrevious")
}

func histNext() {
	log.Printf("histNext")
}

func linePreviousChar() {
	log.Printf("linePreviousChar")
}

func lineNextChar() {
	log.Printf("lineNextChar")
}

func pushSub(buf []byte, size int, b byte) int {
	max := len(buf)

	//log.Printf("pushSub: size=%d cap=%d char=%d", size, max, b)

	if max < 1 {
		log.Printf("pushSub: bad subnegotiation buffer: max=%d", max)
		return size
	}

	if size == 0 {
		buf[0] = b
		return 1
	}

	switch buf[0] {
	case optNaws:
		// we only care about window size
		if size >= max {
			log.Printf("pushSub: subnegotiation buffer overflow: max=%d char=%d", max, b)
			return size
		}
		buf[size] = b
		return size + 1
	}

	return size
}

func flush(client *TelnetClient) {
	client.userFlush <- 1
}

func sendNow(client *TelnetClient, line string) {
	client.userOut <- line
	flush(client)
}

func sendlnNow(client *TelnetClient, line string) {
	sendNow(client, fmt.Sprintf("%s\r\n", line))
}

func sendEcho(client *TelnetClient, line string) {
	if client.serverEcho {
		sendNow(client, line)
	}
}

func inputLoopExit() {
	log.Printf("inputLoop: exiting")
}

func resetReadTimeout(timer *time.Timer, d time.Duration) {
	log.Printf("inputLoop: reset read timeout: %d secs", d/time.Second)
	timer.Reset(d)
}

func inputLoop(client *TelnetClient) {
	//loop:
	//	- read from rd and feed into cli interpreter
	//	- watch idle timeout
	//	- watch quitInput channel

	defer inputLoopExit()

	log.Printf("inputLoop: starting")

	escape := escNone
	iac := IAC_NONE
	buf := [30]byte{} // underlying buffer
	//line := buf[:0]   // position at underlying buffer
	size := 0 // position at underlying buffer

	subBuf := [5]byte{}
	subSize := 0

	read := reader(client.conn)

	timeout := time.Minute * 1
	readTimer := time.NewTimer(timeout)

	resetReadTimeout(readTimer, timeout)

LOOP:
	for {
		select {
		case <-readTimer.C:
			// read timeout
			log.Printf("inputLoop: read timeout, notifying main goroutine")
			sendlnNow(client, "idle timeout")
			client.conn.Close() // intentionally breaks charReadLoop
			inputClosed <- *client
			break LOOP // keep waiting quitInput
		case client.serverEcho = <-client.echo:
			// do nothing
		case client.onlyLine = <-client.sendLine:
			log.Printf("inputLoop: send full line: %v", client.onlyLine)
			// do nothing
		case <-client.quitInput:
			// main goroutine requested us to quit.
			// we are about to finish.
			// then we request outputLoop to quit as well.
			// we can do so directly since the main goroutine won't
			// send any further output to outputLoop channel.
			log.Printf("inputLoop: hit quitInput, requesting outputLoop to quit")
			client.quitOutput <- 1
			return // exit immediately
		case b, ok := <-read:
			if !ok {
				// connection closed.
				// we are about to finish.
				// then we notify the main goroutine to terminate the outputLoop as well.
				// we can't terminate outputLoop directly since the main goroutine may
				// have output pending for the outputLoop (then it could try to write
				// on a closed channel).
				log.Printf("inputLoop: closed channel, notifying main goroutine")
				inputClosed <- *client
				break LOOP // keep waiting quitInput
			}

			resetReadTimeout(readTimer, timeout)

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
						lineNextChar()
					case 'D':
						linePreviousChar()
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

						if !client.onlyLine {
							keyInput <- Command{client, ""}
							continue LOOP
						}

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
						sendEcho(client, "\r\n")
					case ctrlH, keyBackspace:
						// backspace
						if size <= 0 {
							continue
						}
						size--
						// echo backspace to client
						sendEcho(client, string(byte(keyBackspace)))
					case keyEscape:
						escape = escOne
					case ctrlP:
						histPrevious()
					case ctrlN:
						histNext()
					case ctrlB:
						linePreviousChar()
					case ctrlF:
						lineNextChar()
					default:
						log.Printf("inputLoop: unknown control: %d 0x%x", b, b)
					}

					continue

				default:
					// push non-commands bytes into line buffer

					if !client.onlyLine {
						keyInput <- Command{client, string(b)}
						continue LOOP
					}

					if size >= len(buf) {
						/*
							client.userOut <- fmt.Sprintf("\r\nline buffer overflow: size=%d max=%d\r\n", size, len(buf))
							client.userOut <- string(buf[:size]) // redisplay command to user
						*/
						sendNow(client, fmt.Sprintf("\r\nline buffer overflow: size=%d max=%d\r\n", size, len(buf)))
						sendNow(client, string(buf[:size]))
						continue LOOP
					}

					//line = append(buf[:len(line)], b)
					buf[size] = b
					size++

					// echo char back to client
					sendEcho(client, string(b))
				}

			case IAC_CMD:

				switch b {
				case cmdSB:
					log.Printf("inputLoop: telnet SUB begin")
					subSize = 0
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

			case IAC_SUB_IAC:

				if b != cmdSE {
					subSize = pushSub(subBuf[:], subSize, b)
					iac = IAC_SUB
					continue
				}

				// subnegotiation end

				log.Printf("inputLoop: telnet SUB end")
				log.Printf("inputLoop: telnet IAC end")
				iac = IAC_NONE

				if subSize < 1 {
					log.Printf("inputLoop: no subnegotiation char received")
					continue
				}

				if subBuf[0] == optNaws {
					if subSize != 5 {
						log.Printf("inputLoop: invalid telnet NAWS size=%d", subSize)
						continue
					}

					width := int(subBuf[1])<<8 + int(subBuf[2])
					height := int(subBuf[3])<<8 + int(subBuf[4])

					log.Printf("inputLoop: window size: width=%d height=%d", width, height)

					autoHeight <- SetHeight{client, height}
				}

			case IAC_SUB:

				if b == cmdIAC {
					iac = IAC_SUB_IAC
					continue
				}

				subSize = pushSub(subBuf[:], subSize, b)

			default:
				log.Panicf("inputLoop: unexpected state iac=%d", iac)
			}

		}

		log.Printf("inputLoop: buf len=%d [%s]", size, buf[:size])
	}

	log.Printf("inputLoop: waiting quitInput")
WAIT:
	for {
		select {
		case <-client.quitInput:
			log.Printf("inputLoop: quitInput received")
			break WAIT
		}
	}
}

func outputLoop(client *TelnetClient) {
	//loop:
	//	- read from userOut channel and write into wr
	//	- watch quitOutput channel

	// the only way the outputLoop is terminated is thru
	// request on the quitOutput channel.
	// quitOutput is always requested from the main
	// goroutine.
	// when the inputLoop hits a closed connection, it
	// notifies the main goroutine.

	log.Printf("outputLoop: starting")

LOOP:
	for {
		select {
		case msg := <-client.userOut:
			if n, err := client.wr.WriteString(msg); err != nil {
				log.Printf("outputLoop: written=%d from=%d: %v", n, len(msg), err)
			}
		case <-client.userFlush:
			if err := client.wr.Flush(); err != nil {
				log.Printf("outputLoop: flush: %v", err)
			}
		case <-client.quitOutput:
			log.Printf("outputLoop: quit received")
			break LOOP
		}
	}

	log.Printf("outputLoop: exiting")
}

func charMode(conn net.Conn) {
	cmd := []byte{cmdIAC, cmdWill, optEcho, cmdIAC, cmdWill, optSupressGoAhead, cmdIAC, cmdDont, optLinemode}
	if wr, err := conn.Write(cmd); err != nil {
		log.Printf("charMode: len=%d err=%v", wr, err)
	}
}

func windowSize(conn net.Conn) {
	cmd := []byte{cmdIAC, cmdDo, optNaws}
	if wr, err := conn.Write(cmd); err != nil {
		log.Printf("windowSize: len=%d err=%v", wr, err)
	}
}

func handleTelnet(conn net.Conn) {
	defer conn.Close()

	log.Printf("handleTelnet: new telnet connection from: %s", conn.RemoteAddr())

	//rd, wr := bufio.NewReader(conn), bufio.NewWriter(conn)

	charMode(conn)
	windowSize(conn)

	client := TelnetClient{conn,
		bufio.NewWriter(conn),
		make(chan string),
		make(chan int),
		make(chan int),
		make(chan int),
		make(chan bool),
		make(chan bool),
		true,
		MOTD,
		true,
		[]string{},
		[]string{},
		24}

	/*
		https://groups.google.com/d/msg/golang-nuts/JB_iiSQkmOk/dJNKSFQXUUQJ

		There is nothing wrong with having arbitrary numbers of senders, but if
		you do then it doesn't work to close the channel.  You need some other
		way to indicate EOF.

		Ian Lance Taylor
	*/
	//defer close(client.userOut)

	//log.Printf("handleTelnet: debug1 remote=%s", conn.RemoteAddr())

	cmdInput <- Command{&client, ""} // mock user input

	//log.Printf("handleTelnet: debug2 remote=%s", conn.RemoteAddr())

	go inputLoop(&client)

	outputLoop(&client)

	log.Printf("handleTelnet: terminating connection: remote=%s", conn.RemoteAddr())
}

func listenTelnet(addr string) {
	telnetServer := telnet.Server{Addr: addr, Handler: handleTelnet}

	log.Printf("serving telnet on TCP %s", addr)

	if err := telnetServer.ListenAndServe(); err != nil {
		log.Fatalf("telnet server on address %s: error: %s", addr, err)
	}
}
