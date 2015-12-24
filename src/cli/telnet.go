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
		iacNone(s, c, buf, b)
	case IAC_CMD:
		iacCmd(buf, b)
	case IAC_OPT:
		log.Printf("telnetHandleByte: telnet OPT end")
		log.Printf("telnetHandleByte: telnet IAC end")
		buf.iac = IAC_NONE
	case IAC_SUB_IAC:
		iacSubIac(c, buf, b)
	case IAC_SUB:
		if b == cmdIAC {
			buf.iac = IAC_SUB_IAC
			return false
		}
		buf.subSize = pushSub(buf.subBuf[:], buf.subSize, b)
	default:
		log.Printf("telnetHandleByte: unexpected state iac=%d", buf.iac)
		return true // stop
	}

	return false
}

func iacSubIac(c *Client, buf *telnetBuf, b byte) {
	if b != cmdSE {
		buf.subSize = pushSub(buf.subBuf[:], buf.subSize, b)
		buf.iac = IAC_SUB
		return
	}

	// subnegotiation end

	log.Printf("iacSubIac: telnet SUB end")
	log.Printf("iacSubIac: telnet IAC end")
	buf.iac = IAC_NONE

	if buf.subSize < 1 {
		log.Printf("iacSubIac: no subnegotiation char received")
		return
	}

	if buf.subBuf[0] == optNaws {
		if buf.subSize != 5 {
			log.Printf("iacSubIac: invalid telnet NAWS size=%d", buf.subSize)
			return
		}

		width := int(buf.subBuf[1])<<8 + int(buf.subBuf[2])
		height := int(buf.subBuf[3])<<8 + int(buf.subBuf[4])

		log.Printf("iacSubIac: window size: width=%d height=%d", width, height)

		c.TermSizeSet(width, height)
	}
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

func iacCmd(buf *telnetBuf, b byte) {

	switch b {
	case cmdSB:
		log.Printf("iacCmd: telnet SUB begin")
		buf.subSize = 0
		buf.iac = IAC_SUB
	case cmdWill, cmdWont, cmdDo, cmdDont:
		log.Printf("iacCmd: telnet OPT begin")
		buf.iac = IAC_OPT
	default:
		log.Printf("iacCmd: telnet IAC end")
		buf.iac = IAC_NONE
	}
}

func iacNone(s *Server, c *Client, buf *telnetBuf, b byte) {

	if buf.escape != escNone {
		if handleEscape(s, c, buf, b) {
			return
		}
	}

	switch {
	case b == cmdIAC:
		// hit IAC mark?
		log.Printf("iacNone: telnet IAC begin")
		buf.iac = IAC_CMD
	case b == keyBackspace, b < 32:
		controlChar(s, c, buf, b)
	default:
		// push non-commands bytes into line buffer

		everyChar := c.SendEveryChar()
		if everyChar {
			s.CommandChannel <- Command{Client: c, Cmd: string(b), IsLine: false}
			return
		}

		if buf.lineSize >= len(buf.lineBuf) {
			// line buffer overflow
			return
		}

		buf.lineBuf[buf.lineSize] = b
		buf.lineSize++

		c.echoSend(string(b)) // echo key back to terminal

		log.Printf("iacNone: line=[%v]", string(buf.lineBuf[:buf.lineSize]))
	}
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
