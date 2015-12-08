package cli

import (
	"log"
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
	escNone  = iota
	escOne   = iota
	escTwo   = iota
	escThree = iota
)

const (
	IAC_NONE    = iota
	IAC_CMD     = iota
	IAC_OPT     = iota
	IAC_SUB     = iota
	IAC_SUB_IAC = iota
)

type telnetBuf struct {
	escape   int
	iac      int
	lineBuf  [30]byte
	lineSize int
	subBuf   [5]byte
	subSize  int
}

func newTelnetBuf() *telnetBuf {
	return &telnetBuf{
		escape:   escNone,
		iac:      IAC_NONE,
		lineBuf:  [30]byte{},
		lineSize: 0,
		subBuf:   [5]byte{},
		subSize:  0,
	}
}

func telnetHandleByte(s *Server, c *Client, buf *telnetBuf, b byte) bool {

	switch {
	case b == '\n':

		sendEveryChar := c.SendEveryChar()
		if sendEveryChar {
			s.CommandChannel <- Command{Client: c, Cmd: "", IsLine: false}
			return false
		}

		cmdLine := string(buf.lineBuf[:buf.lineSize]) // string is safe for sharing (immutable)
		log.Printf("cli.InputLoop: size=%d cmdLine=[%v]", buf.lineSize, cmdLine)
		s.CommandChannel <- Command{Client: c, Cmd: cmdLine, IsLine: true}
		buf.lineSize = 0 // reset reading buffer position

		//c.echoSend("\r\n") // echo newline back to client

	case b == ctrlH, b == keyBackspace:
		// backspace
		if buf.lineSize <= 0 {
			return false
		}
		buf.lineSize--                         // erase backspace key from input buffer
		c.echoSend(string(byte(keyBackspace))) // echo backspace to client
	case b < 32, b > 127:
		// discard control bytes (includes '\r')
	default:
		// push non-commands bytes into line buffer

		everyChar := c.SendEveryChar()
		if everyChar {
			s.CommandChannel <- Command{Client: c, Cmd: string(b), IsLine: false}
			return false
		}

		if buf.lineSize >= len(buf.lineBuf) {
			// line buffer overflow
			return false
		}

		buf.lineBuf[buf.lineSize] = b
		buf.lineSize++

		c.echoSend(string(b)) // echo key back to terminal

		log.Printf("cli.InputLoop: line=[%v]", string(buf.lineBuf[:buf.lineSize]))
	}

	return false
}
