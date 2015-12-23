package cli

import (
	"log"
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

	switch buf.iac {
	case IAC_NONE:
		return iacNone(s, c, buf, b)
	case IAC_CMD:
	case IAC_OPT:
	case IAC_SUB_IAC:
	case IAC_SUB:
	default:
		log.Printf("telnetHandleByte: unexpected state iac=%d", buf.iac)
		return true // stop
	}

	return false
}

func iacNone(s *Server, c *Client, buf *telnetBuf, b byte) bool {

	if buf.escape != escNone {
		if handleEscape(s, c, buf, b) {
			return false
		}
	}

	switch {
	/*
		case b == cmdIAC:
			// hit IAC mark?
			log.Printf("iacNone: telnet IAC begin")
			buf.iac = IAC_CMD
	*/
	case b == keyBackspace, b < 32:
		controlChar(s, c, buf, b)
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

		log.Printf("iacNone: line=[%v]", string(buf.lineBuf[:buf.lineSize]))
	}

	return false
}

func controlChar(s *Server, c *Client, buf *telnetBuf, b byte) {

	switch b {
	case '\r':
		// discard
	case '\n':

		sendEveryChar := c.SendEveryChar()
		if sendEveryChar {
			s.CommandChannel <- Command{Client: c, Cmd: "", IsLine: false}
			return
		}

		cmdLine := string(buf.lineBuf[:buf.lineSize]) // string is safe for sharing (immutable)
		log.Printf("controlChar: size=%d cmdLine=[%v]", buf.lineSize, cmdLine)
		s.CommandChannel <- Command{Client: c, Cmd: cmdLine, IsLine: true}
		buf.lineSize = 0 // reset reading buffer position

		//c.echoSend("\r\n") // echo newline back to client
		c.SendlnNow("") // echo newline back to client

	case ctrlH, keyBackspace:
		// backspace
		if buf.lineSize <= 0 {
			return
		}
		buf.lineSize--                         // erase backspace key from input buffer
		c.echoSend(string(byte(keyBackspace))) // echo backspace to client

	case ctrlA:
		lineBegin()
	case ctrlE:
		lineEnd()
	case keyEscape:
		buf.escape = escOne
	case ctrlP:
		histPrevious()
	case ctrlN:
		histNext()
	case ctrlB:
		linePreviousChar()
	case ctrlF:
		lineNextChar()
	case ctrlD:
		lineDelChar()

	default:
		log.Printf("controlChar: unknown control: %d 0x%x", b, b)
	}
}

func handleEscape(s *Server, c *Client, buf *telnetBuf, b byte) bool {

	switch buf.escape {
	case escOne:
		switch b {
		case '[':
			buf.escape = escTwo
		default:
			log.Printf("handleEscape: unsupported char=%d for escape stage: %d", b, buf.escape)
			buf.escape = escNone
		}
	case escTwo:
		switch b {
		case '1':
			lineHome()
			buf.escape = escThree
		case '3':
			lineDelChar()
			buf.escape = escThree
		case '4':
			lineEnd()
			buf.escape = escThree
		case 'A':
			histPrevious()
			buf.escape = escNone
		case 'B':
			histNext()
			buf.escape = escNone
		case 'C':
			lineNextChar()
			buf.escape = escNone
		case 'D':
			linePreviousChar()
			buf.escape = escNone
		default:
			log.Printf("handleEscape: unsupported char=%d for escape stage: %d", b, buf.escape)
			buf.escape = escNone
		}
	case escThree:
		switch b {
		case '~':
		default:
			log.Printf("handleEscape: unexpected char=%d for escape: %d", b, buf.escape)
		}
		buf.escape = escNone
	default:
		log.Printf("handleEscape: bad escape status: %d", buf.escape)
		return false
	}

	return true
}

func lineBegin() {
	log.Printf("lineBegin")
}

func lineEnd() {
	log.Printf("lineEnd")
}

func lineHome() {
	log.Printf("lineHome")
}

func lineDelChar() {
	log.Printf("lineDelChar")
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
