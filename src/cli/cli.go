package cli

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"command"
)

type Server struct {
	CommandChannel chan Command
	InputClosed    chan *Client
}

type Client struct {
	mutex         *sync.RWMutex // mutex: cli.Client is shared between 2 goroutines: cli.InputLoop and main
	conn          net.Conn
	sendEveryChar bool
	status        int
	echo          bool

	outputChannel chan string // outputLoop: read from outputChannel and write into outputWriter
	outputFlush   chan int    // request flush
	outputQuit    chan int    // request quit
	outputWriter  *bufio.Writer
	outputQueue   []string
	outputBuf     string

	configPath string
	width      int
	height     int

	history      []string
	historyLimit int
	historyPos   int

	telnetLine *telnetBuf
}

func (c *Client) HistoryAdd(cmd string) {

	//log.Printf("HistoryAdd: size=%d max=%d cmd=[%s]", len(c.history), c.historyLimit, cmd)

	size := len(c.history)
	if size > 0 && c.history[size-1] == cmd {
		// do not insert duplicate of last inserted command
		return
	}

	if size >= c.historyLimit {
		// pop first command
		last := c.historyLimit - 1
		for i := 0; i < last; i++ {
			c.history[i] = c.history[i+1]
		}
		c.history = c.history[:last]
	}

	c.history = append(c.history, cmd)
}

func (c *Client) HistoryShow() {
	for _, h := range c.history {
		c.Sendln(fmt.Sprintf("[%s]", h))
	}
}

func (c *Client) HistoryReset() {
	c.historyPos = -1
}

func (c *Client) HistoryPrevious() string {
	size := len(c.history)
	if size < 1 {
		return ""
	}
	if c.historyPos == 0 {
		return ""
	}
	if c.historyPos < 0 {
		c.historyPos = size
	}
	c.historyPos--
	return c.history[c.historyPos]
}

func (c *Client) HistoryNext() string {
	size := len(c.history)
	if size < 1 {
		return ""
	}
	if c.historyPos >= size-1 {
		return ""
	}
	if c.historyPos < 0 {
		return ""
	}
	c.historyPos++
	return c.history[c.historyPos]
}

func (c *Client) DiscardOutputQueue() {
	c.mutex.Lock()
	c.outputQueue = nil
	c.mutex.Unlock()
}

func (c *Client) Output() chan<- string {
	return c.outputChannel
}

func (c *Client) InputQuit() {
	c.conn.Close() // breaks InputLoop goroutine -> InputLoop then sends quit request to OutputLoop
}

func (c *Client) TermSize() (int, int) {
	c.mutex.RLock()
	w := c.width
	h := c.height
	c.mutex.RUnlock()
	return w, h
}

func (c *Client) TermSizeSet(w, h int) {
	c.mutex.Lock()
	c.width = w
	c.height = h
	c.mutex.Unlock()
}

func (c *Client) SendlnNow(msg string) {
	c.SendNow(fmt.Sprintf("%s\r\n", msg))
}

func (c *Client) SendNow(msg string) {
	c.outputChannel <- msg
	c.Flush()
}

func (c *Client) Sendln(msg string) {
	c.Send(fmt.Sprintf("%s\r\n", msg))
}

// enqueue message for client
// break messages into LF-terminated lines
// append every line to outputQueue
func (c *Client) Send(msg string) {
	c.outputBuf += msg

	for {
		i := strings.IndexByte(c.outputBuf, '\n') // find end of line
		if i < 0 {
			// end of line not found
			break
		}
		// end of line found
		// i: points to LF
		i++
		// i: points just past LF
		c.outputQueue = append(c.outputQueue, c.outputBuf[:i]) // push line into output queue
		c.outputBuf = c.outputBuf[i:]                          // skip line
	}
}

// send lines from outputQueue, paging on terminal height
func (c *Client) SendQueue() bool {
	width, height := c.TermSize()
	termMax := height - 2
	if termMax < 1 {
		termMax = 1
	}
	sent := 0
	termLines := 0
	for i, m := range c.outputQueue {
		lineWidth := len(strings.TrimRight(m, "\r\n"))
		lineHeight := getLineHeight(lineWidth, width)
		if termLines+lineHeight >= termMax {
			break
		}
		c.outputChannel <- m
		c.outputQueue[i] = "" // release line immediately - no need to depend on future append()
		termLines += lineHeight
		sent++
	}

	c.outputQueue = c.outputQueue[sent:]
	paging := len(c.outputQueue) > 0

	return paging
}

// getLineWidth: how many terminal lines will be consumed
func getLineHeight(lineWidth, termWidth int) int {
	if lineWidth < 1 {
		return 1
	}

	return (lineWidth-1)/termWidth + 1
}

func (c *Client) SendPrompt(host string, paging bool) {
	if paging {
		c.outputChannel <- "\r\nENTER=more q=quit>"
		return
	}

	defer c.RedrawLineBuffer()

	path := c.ConfigPath()
	if path != "" {
		path = fmt.Sprintf(":%s", path)
	}

	var p string

	status := c.Status()
	switch status {
	case command.USER:
		p = " login:"
	case command.PASS:
		host = ""
		p = "password:"
	case command.EXEC:
		p = ">"
	case command.ENAB:
		p = "#"
	case command.CONF:
		p = "(conf)#"
	default:
		p = "?"
	}

	// can't use send() since sendQueue() runs before sendPrompt().
	// output is flushed by caller
	//c.outputChannel <- fmt.Sprintf("\r\n%s%s%s ", host, path, p)
	c.outputChannel <- fmt.Sprintf("%s%s%s ", host, path, p)
}

// redraw line edit buffer
func (c *Client) RedrawLineBuffer() {

	var buf []byte

	c.mutex.RLock()
	pos := c.telnetLine.linePos
	size := c.telnetLine.lineSize
	{
		s := c.telnetLine.lineBuf[:size] // slice
		buf = append(buf, s...)          // clone slice
	}
	c.mutex.RUnlock()

	// draw full line (will put cursor at end of line)
	for i := 0; i < size; i++ {
		drawByte(c, buf[i])
	}

	// move cursor back to original position
	for i := size; i > pos; i-- {
		cursorLeft(c)
	}
}

func (c *Client) Flush() {
	c.outputFlush <- 1
}

func (c *Client) echoSend(msg string) {
	if c.Echo() {
		c.SendNow(msg)
	}
}

func (c *Client) Echo() bool {
	c.mutex.RLock()
	result := c.echo
	c.mutex.RUnlock()
	return result
}

func (c *Client) EchoEnable() {
	c.mutex.Lock()
	c.echo = true
	c.mutex.Unlock()
}

func (c *Client) EchoDisable() {
	c.mutex.Lock()
	c.echo = false
	c.mutex.Unlock()
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

func (c *Client) StatusSet(status int) {
	c.mutex.Lock()
	c.status = status
	c.mutex.Unlock()
}

func (c *Client) StatusEnable() {
	c.StatusSet(command.ENAB)
}

func (c *Client) StatusConf() {
	c.StatusSet(command.CONF)
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
	Client          *Client
	Cmd             string
	IsLine          bool // true=line false=char
	HideFromHistory bool
}

func NewServer() *Server {
	return &Server{
		CommandChannel: make(chan Command),
		InputClosed:    make(chan *Client),
	}
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		mutex:         &sync.RWMutex{},
		conn:          conn,
		status:        command.MOTD,
		outputWriter:  bufio.NewWriter(conn),
		outputChannel: make(chan string),
		outputFlush:   make(chan int),
		outputQuit:    make(chan int),
		height:        20,
		width:         80,
		echo:          true,
		historyLimit:  20,
		historyPos:    -1,
		telnetLine:    newTelnetBuf(),
	}
}

func InputLoop(s *Server, c *Client, notifyAppInputClosed bool) {
	log.Printf("cli.InputLoop: starting")

	readCh := spawnReadLoop(c.conn)

	buf := c.telnetLine

	timeout := time.Minute * 1
	readTimer := time.NewTimer(timeout)
	resetReadTimeout(readTimer, timeout)

LOOP:
	for {
		select {
		case <-time.After(time.Second * 5):
			log.Printf("cli.InputLoop: tick")
		case <-readTimer.C:
			// read timeout
			log.Printf("InputLoop: read timeout, closing socket")
			c.SendlnNow("idle timeout")
			break LOOP
		case b, ok := <-readCh:
			if !ok {
				// connection closed
				log.Printf("cli.InputLoop: closed channel")
				break LOOP
			}
			//log.Printf("cli.InputLoop: input=[%v]", b)

			resetReadTimeout(readTimer, timeout)

			if stop := telnetHandleByte(s, c, buf, b); stop {
				log.Printf("cli.InputLoop: telnetHandleByte requested termination")
				break LOOP
			}
		}
	}

	if notifyAppInputClosed {
		s.InputClosed <- c // notify main goroutine
	}
	c.outputQuit <- 1 // request OutputLoop termination

	log.Printf("cli.InputLoop: exiting")
}

func resetReadTimeout(timer *time.Timer, d time.Duration) {
	//log.Printf("InputLoop: reset read timeout: %d secs", d/time.Second)
	timer.Reset(d)
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

	log.Printf("cli.OutputLoop: starting")

LOOP:
	for {
		select {
		case <-time.After(time.Second * 5):
			log.Printf("cli.OutputLoop: tick")
		case msg := <-c.outputChannel:
			if n, err := c.outputWriter.WriteString(msg); err != nil {
				log.Printf("cli.OutputLoop: written=%d from=%d: %v", n, len(msg), err)
			}
		case <-c.outputFlush:
			if err := c.outputWriter.Flush(); err != nil {
				log.Printf("cli.OutputLoop: flush: %v", err)
			}
		case <-c.outputQuit:
			// when the InputLoop goroutine hits a closed connection,
			// it sends quit request to OutputLoop outputQuit channel
			log.Printf("cli.OutputLoop: quit request received (from InputLoop)")
			break LOOP
		}
	}

	log.Printf("cli.OutputLoop: exiting")
}
