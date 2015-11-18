package cli

import (
	"log"
	"net"
	"time"
)

type Server struct {
	commandChannel chan Command
}

type Client struct {
	conn net.Conn
}

type Command struct {
	client Client
	cmd    string
}

func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

func InputLoop(s *Server, c *Client) {
	log.Printf("cli.InputLoop: starting")

	readCh := spawnReadLoop(c.conn)

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
