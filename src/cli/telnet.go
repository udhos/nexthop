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
	keyBackspace = ctrlH // 8
	keyTab       = 9
	keyEscape    = 27
)

const (
	optEcho           = 1
	optSupressGoAhead = 3
	optNaws           = 31 // rfc1073
	optLinemode       = 34
)

const (
	ctrlA        = 'A' - '@'
	ctrlB        = 'B' - '@'
	ctrlC        = 'C' - '@'
	ctrlD        = 'D' - '@'
	ctrlE        = 'E' - '@'
	ctrlF        = 'F' - '@'
	ctrlH        = 'H' - '@'
	ctrlK        = 'K' - '@'
	ctrlN        = 'N' - '@'
	ctrlP        = 'P' - '@'
	ctrlZ        = 'Z' - '@'
	ctrlQuestion = 127
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

func telnetHandleByte(s *Server, c *Client, buf *telnetBuf, b byte) bool {

	switch buf.iacGet() {
	case IAC_NONE:
		iacNone(s, c, buf, b)
	case IAC_CMD:
		iacCmd(buf, b)
	case IAC_OPT:
		log.Printf("telnetHandleByte: telnet OPT end")
		log.Printf("telnetHandleByte: telnet IAC end")
		buf.iacSet(IAC_NONE)
	case IAC_SUB_IAC:
		iacSubIac(c, buf, b)
	case IAC_SUB:
		if b == cmdIAC {
			buf.iacSet(IAC_SUB_IAC)
			return false
		}
		buf.pushSub(b)
	default:
		log.Printf("telnetHandleByte: unexpected state iac=%d", buf.iac)
		return true // stop
	}

	return false
}

func iacSubIac(c *Client, buf *telnetBuf, b byte) {
	if b != cmdSE {
		buf.pushSub(b)
		buf.iacSet(IAC_SUB)
		return
	}

	// subnegotiation end

	log.Printf("iacSubIac: telnet SUB end")
	log.Printf("iacSubIac: telnet IAC end")
	buf.iacSet(IAC_NONE)

	subSize := buf.subSizeGet()

	if subSize < 1 {
		log.Printf("iacSubIac: no subnegotiation char received")
		return
	}

	subBuf := buf.subBufCopy()

	if subBuf[0] == optNaws {
		if subSize != 5 {
			log.Printf("iacSubIac: invalid telnet NAWS size=%d", subSize)
			return
		}

		width := int(subBuf[1])<<8 + int(subBuf[2])
		height := int(subBuf[3])<<8 + int(subBuf[4])

		log.Printf("iacSubIac: window size: width=%d height=%d", width, height)

		c.TermSizeSet(width, height)
	}
}

func iacCmd(buf *telnetBuf, b byte) {

	switch b {
	case cmdSB:
		log.Printf("iacCmd: telnet SUB begin")
		buf.subBufReset()
		buf.iacSet(IAC_SUB)
	case cmdWill, cmdWont, cmdDo, cmdDont:
		log.Printf("iacCmd: telnet OPT begin")
		buf.iacSet(IAC_OPT)
	default:
		log.Printf("iacCmd: telnet IAC end")
		buf.iacSet(IAC_NONE)
	}
}

func iacNone(s *Server, c *Client, buf *telnetBuf, b byte) {

	//log.Printf("iacNone: byte: %d 0x%x", b, b)

	if b != 0 {
		buf.notCtrlM()
	}

	esc := buf.escapeGet()
	if esc != escNone {
		if handleEscape(s, c, buf, b, esc) {
			return
		}
	}

	switch {
	case b == cmdIAC:
		// hit IAC mark?
		log.Printf("iacNone: telnet IAC begin")
		buf.iacSet(IAC_CMD)
	case b == ctrlQuestion, b < 32:
		controlChar(s, c, buf, b)
	case b == '?':
		helpCommandChar(s, c, buf, b)
	default:
		// push non-commands bytes into line buffer

		everyChar := c.SendEveryChar()
		if everyChar {
			s.CommandChannel <- Command{Client: c, Cmd: string(b)}
			return
		}

		buf.insert(c, b)
	}
}

func cursorLeft(c *Client) {
	drawByte(c, byte(keyBackspace))
}

func drawByte(c *Client, b byte) {
	c.echoSend(string(b))
}

func msg(s *Server, c *Client, str string) {
	c.Sendln(str)

	// make main goroutine to send the message queue and command prompt
	s.CommandChannel <- Command{Client: c}
}

func controlChar(s *Server, c *Client, buf *telnetBuf, b byte) {

	// RETURN: CR LF
	// CtrlM: CR NUL
	// CtrlJ: LF

	switch b {
	case '\r': // CR
		buf.hitCR()
	case '\n': // LF
		newlineChar(s, c, buf, b)
	case ctrlQuestion, keyBackspace:
		buf.lineBackspace(c)
	case keyTab:
		helpCommandChar(s, c, buf, b)
	case ctrlA:
		lineBegin(c, buf)
	case ctrlE:
		buf.lineEnd(c)
	case keyEscape:
		buf.escape = escOne
	case ctrlP:
		histPrevious(c, buf)
	case ctrlN:
		histNext(c, buf)
	case ctrlB:
		linePreviousChar(c, buf)
	case ctrlF:
		lineNextChar(c, buf)
	case ctrlD:
		if buf.getLineSize() < 1 {
			// EOF
			msg(s, c, "use 'quit' to exit remote terminal")
			return
		}
		buf.lineDelChar(c)
	case ctrlK:
		buf.lineKillToEnd(c)
	case 0:
		if buf.isExpectingCtrlM() {
			// controlM
			newlineChar(s, c, buf, b)
		}
	default:
		log.Printf("controlChar: unknown control: %d 0x%x", b, b)
	}
}

func newlineChar(s *Server, c *Client, buf *telnetBuf, b byte) {

	if c.SendEveryChar() {
		s.CommandChannel <- Command{Client: c}
		return
	}

	cmdLine := buf.lineExtract() // copy line and reset line buffer
	log.Printf("controlChar: size=%d cmdLine=[%v]", buf.lineSize, cmdLine)

	// string is safe for sharing (immutable)
	s.CommandChannel <- Command{Client: c, Cmd: cmdLine, IsLine: true}

	c.HistoryReset()

	c.SendlnNow("") // echo newline back to client
}

func helpCommandChar(s *Server, c *Client, buf *telnetBuf, b byte) {
	cmdLine := buf.lineCopy() + string(b) // string is safe for sharing (immutable)

	s.CommandChannel <- Command{Client: c, Cmd: cmdLine, IsLine: true, HideFromHistory: true}
}

func handleEscape(s *Server, c *Client, buf *telnetBuf, b byte, esc int) bool {

	switch esc {
	case escOne:
		switch b {
		case '[':
			buf.escapeSet(escTwo)
		default:
			log.Printf("handleEscape: unsupported char=%d for escape stage: %d", b, buf.escape)
			buf.escapeSet(escNone)
		}
	case escTwo:
		switch b {
		case '1':
			lineBegin(c, buf)
			buf.escapeSet(escThree)
		case '3':
			buf.lineDelChar(c)
			buf.escapeSet(escThree)
		case '4':
			buf.lineEnd(c)
			buf.escapeSet(escThree)
		case 'A':
			histPrevious(c, buf)
			buf.escapeSet(escNone)
		case 'B':
			histNext(c, buf)
			buf.escapeSet(escNone)
		case 'C':
			lineNextChar(c, buf)
			buf.escapeSet(escNone)
		case 'D':
			linePreviousChar(c, buf)
			buf.escapeSet(escNone)
		default:
			log.Printf("handleEscape: unsupported char=%d for escape stage: %d", b, esc)
			buf.escapeSet(escNone)
		}
	case escThree:
		switch b {
		case '~':
		default:
			log.Printf("handleEscape: unexpected char=%d for escape: %d", b, esc)
		}
		buf.escapeSet(escNone)
	default:
		log.Printf("handleEscape: bad escape status: %d", esc)
		return false
	}

	return true
}

func histPrevious(c *Client, buf *telnetBuf) {
	histMove(c, buf, c.HistoryPrevious())
}

func histNext(c *Client, buf *telnetBuf) {
	histMove(c, buf, c.HistoryNext())
}

func histMove(c *Client, buf *telnetBuf, hist string) {
	if hist == "" {
		return
	}
	clearLine(c, buf)

	for i, b := range hist {
		buf.lineBuf[i] = byte(b)
	}
	buf.lineSize = len(hist)

	drawLine(c, buf)
}

func clearLine(c *Client, buf *telnetBuf) {
	goToLineEnd(c, buf)
	for buf.linePos > 0 {
		backspaceChar(c, buf)
	}
}

func drawLine(c *Client, buf *telnetBuf) {
	for buf.linePos < buf.lineSize {
		cRight(c, buf)
	}
}

func linePreviousChar(c *Client, buf *telnetBuf) {
	if buf.linePos < 1 {
		return
	}

	buf.linePos--
	cursorLeft(c)
}

func lineNextChar(c *Client, buf *telnetBuf) {
	if buf.linePos >= buf.lineSize {
		return
	}

	cRight(c, buf)
}
