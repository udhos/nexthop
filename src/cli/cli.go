package cli

import (
	"log"
	"net"
	"time"
)

type Server struct {
	CommandChannel chan Command
}

type Client struct {
	conn          net.Conn
	sendEveryChar bool
}

type Command struct {
	client *Client
	Cmd    string
	IsLine bool // true=line false=char
}

func NewServer() *Server {
	return &Server{CommandChannel: make(chan Command)}
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

func InputLoop(s *Server, c *Client) {
	log.Printf("cli.InputLoop: starting")

	readCh := spawnReadLoop(c.conn)

	lineBuf := [30]byte{} // underlying buffer
	lineSize := 0         // position at underlying buffer

LOOP:
	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("cli.InputLoop: tick")
		case b, ok := <-readCh:
			if !ok {
				// connection closed.
				log.Printf("cli.InputLoop: closed channel")
				break LOOP
			}
			log.Printf("cli.InputLoop: input=[%v]", b)

			switch {
			case b == '\n':

				if c.sendEveryChar {
					s.CommandChannel <- Command{client: c, Cmd: "", IsLine: false}
					continue LOOP
				}

				cmdLine := string(lineBuf[:lineSize]) // string is safe for sharing (immutable)
				log.Printf("cli.InputLoop: size=%d cmdLine=[%v]", lineSize, cmdLine)
				s.CommandChannel <- Command{client: c, Cmd: cmdLine, IsLine: true}
				lineSize = 0 // reset reading buffer position
			case b < 32:
				// discard control bytes (includes '\r')
			default:
				// push non-commands bytes into line buffer

				if c.sendEveryChar {
					s.CommandChannel <- Command{client: c, Cmd: string(b), IsLine: false}
					continue LOOP
				}

				if lineSize >= len(lineBuf) {
					// line buffer overflow
					continue LOOP
				}

				lineBuf[lineSize] = b
				lineSize++

				log.Printf("cli.InputLoop: line=[%v]", string(lineBuf[:lineSize]))
			}
		}
	}

	log.Printf("cli.InputLoop: exiting")
}

func spawnReadLoop(conn net.Conn) <-chan byte {
	readCh := make(chan byte)
	go charReadLoop(conn, readCh)
	return readCh
}

func charReadLoop(conn net.Conn, readCh chan<- byte) {

	// we are the only one sender on this channel.
	// so we can use the channel close idiom for
	// signaling EOF.
	defer close(readCh)

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
			readCh <- b
		}
	}

	log.Printf("charReadLoop: exiting")
}

func OutputLoop(s *Server, c *Client) {
	log.Printf("cli.InputLoop: starting")
	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("cli.OutputLoop: tick")
		}
	}
}
