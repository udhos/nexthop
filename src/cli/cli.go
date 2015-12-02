package cli

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"

	"command"
)

type Server struct {
	CommandChannel chan Command
}

// cli.Client is shared between 2 goroutines: cli.InputLoop and main
type Client struct {
	mutex         *sync.RWMutex
	conn          net.Conn
	sendEveryChar bool
	status        int

	outputChannel chan string // outputLoop: read from outputChannel and write into outputWriter
	outputFlush   chan int    // request flush
	outputQuit    chan int    // request quit
	outputWriter  *bufio.Writer

	configPath string
}

func (c *Client) ConfigPath() string {
	c.mutex.RLock()
	result := c.configPath
	c.mutex.RUnlock()
	return result
}

func (c *Client) ConfigPathSet(path string) {
	c.mutex.Lock()
	c.configPath = path
	c.mutex.Unlock()

}

func (c *Client) SendEveryChar() bool {
	c.mutex.RLock()
	result := c.sendEveryChar
	c.mutex.RUnlock()
	return result
}

func (c *Client) SetSendEveryChar(mode bool) {
	c.mutex.Lock()
	c.sendEveryChar = mode
	c.mutex.Unlock()
}

func (c *Client) Status() int {
	c.mutex.RLock()
	result := c.status
	c.mutex.RUnlock()
	return result
}

func (c *Client) StatusEnable() {
	c.mutex.Lock()
	c.status = command.ENAB
	c.mutex.Unlock()
}

func (c *Client) StatusConf() {
	c.mutex.Lock()
	c.status = command.CONF
	c.mutex.Unlock()
}

func (c *Client) StatusExit() {
	c.mutex.Lock()
	if c.status > command.EXEC {
		c.status--
	}
	c.mutex.Unlock()
}

// Command is copied from cli.InputLoop goroutine to main goroutine
type Command struct {
	Client *Client
	Cmd    string
	IsLine bool // true=line false=char
}

func NewServer() *Server {
	return &Server{CommandChannel: make(chan Command)}
}

func NewClient(conn net.Conn) *Client {
	return &Client{mutex: &sync.RWMutex{}, conn: conn, status: command.EXEC, outputWriter: bufio.NewWriter(conn)}
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

				everyChar := c.SendEveryChar()
				if everyChar {
					s.CommandChannel <- Command{Client: c, Cmd: "", IsLine: false}
					continue LOOP
				}

				cmdLine := string(lineBuf[:lineSize]) // string is safe for sharing (immutable)
				log.Printf("cli.InputLoop: size=%d cmdLine=[%v]", lineSize, cmdLine)
				s.CommandChannel <- Command{Client: c, Cmd: cmdLine, IsLine: true}
				lineSize = 0 // reset reading buffer position
			case b < 32, b > 127:
				// discard control bytes (includes '\r')
			default:
				// push non-commands bytes into line buffer

				everyChar := c.SendEveryChar()
				if everyChar {
					s.CommandChannel <- Command{Client: c, Cmd: string(b), IsLine: false}
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

func OutputLoop(c *Client) {
	//loop:
	//	- read from userOut channel and write into wr
	//	- watch quitOutput channel

	// the only way the outputLoop is terminated is thru
	// request on the quitOutput channel.
	// quitOutput is always requested from the main
	// goroutine.
	// when the inputLoop hits a closed connection, it
	// notifies the main goroutine.

	log.Printf("cli.OutputLoop: starting")

LOOP:
	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("cli.OutputLoop: tick ERASEME")
		case msg := <-c.outputChannel:
			if n, err := c.outputWriter.WriteString(msg); err != nil {
				log.Printf("cli.OutputLoop: written=%d from=%d: %v", n, len(msg), err)
			}
		case <-c.outputFlush:
			if err := c.outputWriter.Flush(); err != nil {
				log.Printf("cli.OutputLoop: flush: %v", err)
			}
		case <-c.outputQuit:
			log.Printf("cli.OutputLoop: quit received")
			break LOOP
		}
	}

	log.Printf("cli.OutputLoop: exiting")
}
