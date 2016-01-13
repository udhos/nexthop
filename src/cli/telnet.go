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

type bufByteArray [100]byte // affects max input line length

type telnetBuf struct {
	escape         int
	iac            int
	lineBuf        bufByteArray
	lineSize       int
	linePos        int
	subBuf         [5]byte
	subSize        int
	expectingCtrlM bool
}

func newTelnetBuf() *telnetBuf {
	return &telnetBuf{
		escape:         escNone,
		iac:            IAC_NONE,
		lineBuf:        bufByteArray{},
		lineSize:       0,
		linePos:        0,
		subBuf:         [5]byte{},
		subSize:        0,
		expectingCtrlM: false,
	}
}

func telnetHandleByte(s *Server, c *Client, buf *telnetBuf, b byte) bool {

	//log.Printf("telnetHandleByte: byte: %d 0x%x", b, b)

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

	//log.Printf("iacNone: byte: %d 0x%x", b, b)

	if b != 0 {
		buf.expectingCtrlM = false
	}

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
	case b == ctrlQuestion, b < 32:
		controlChar(s, c, buf, b)
	case b == '?':
		msg(s, c, "? key: command context help - FIXME WRITEME")
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

		// insert
		for i := buf.lineSize; i > buf.linePos; i-- {
			buf.lineBuf[i] = buf.lineBuf[i-1]
		}

		buf.lineBuf[buf.linePos] = b
		buf.lineSize++
		buf.linePos++

		// redraw
		for i := buf.linePos - 1; i < buf.lineSize; i++ {
			drawByte(c, buf.lineBuf[i])
		}

		// reposition cursor
		for i := buf.linePos; i < buf.lineSize; i++ {
			cursorLeft(c)
		}

		log.Printf("iacNone: pos=%d size=%d line=[%v]", buf.linePos, buf.lineSize, string(buf.lineBuf[:buf.lineSize]))
	}
}

func cursorLeft(c *Client) {
	drawByte(c, byte(keyBackspace))
}

func cursorRight(c *Client, buf *telnetBuf) {
	drawCurrent(c, buf)
	buf.linePos++
}

func drawCurrent(c *Client, buf *telnetBuf) {
	drawByte(c, buf.lineBuf[buf.linePos])
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
		buf.expectingCtrlM = true
	case '\n': // LF
		newlineChar(s, c, buf, b)
	case ctrlQuestion, keyBackspace:
		lineBackspace(c, buf)
	case keyTab:
		msg(s, c, "TAB key: command completion - FIXME WRITEME")
	case ctrlA:
		lineBegin(c, buf)
	case ctrlE:
		lineEnd(c, buf)
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
		if buf.lineSize < 1 {
			// EOF
			msg(s, c, "use 'quit' to exit remote terminal")
			return
		}
		lineDelChar(c, buf)
	case ctrlK:
		lineKillToEnd(c, buf)
	case 0:
		if buf.expectingCtrlM {
			// controlM
			newlineChar(s, c, buf, b)
		}
	default:
		log.Printf("controlChar: unknown control: %d 0x%x", b, b)
	}
}

func newlineChar(s *Server, c *Client, buf *telnetBuf, b byte) {
	//log.Printf("newlineChar()")

	sendEveryChar := c.SendEveryChar()
	if sendEveryChar {
		s.CommandChannel <- Command{Client: c, Cmd: "", IsLine: false}
		return
	}

	cmdLine := string(buf.lineBuf[:buf.lineSize]) // string is safe for sharing (immutable)
	log.Printf("controlChar: size=%d cmdLine=[%v]", buf.lineSize, cmdLine)
	s.CommandChannel <- Command{Client: c, Cmd: cmdLine, IsLine: true}

	// reset reading buffer position
	buf.lineSize = 0
	buf.linePos = 0
	c.HistoryReset()

	c.SendlnNow("") // echo newline back to client
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
			lineBegin(c, buf)
			buf.escape = escThree
		case '3':
			lineDelChar(c, buf)
			buf.escape = escThree
		case '4':
			lineEnd(c, buf)
			buf.escape = escThree
		case 'A':
			histPrevious(c, buf)
			buf.escape = escNone
		case 'B':
			histNext(c, buf)
			buf.escape = escNone
		case 'C':
			lineNextChar(c, buf)
			buf.escape = escNone
		case 'D':
			linePreviousChar(c, buf)
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

func lineKillToEnd(c *Client, buf *telnetBuf) {
	killCount := buf.lineSize - buf.linePos

	// erase chars
	for i := 0; i < killCount; i++ {
		drawByte(c, ' ')
	}

	// return cursor
	for i := 0; i < killCount; i++ {
		cursorLeft(c)
	}

	buf.lineSize = buf.linePos // drop chars from buffer
}

func lineBackspace(c *Client, buf *telnetBuf) {
	if buf.linePos < 1 {
		return
	}

	cursorLeft(c)
	buf.linePos--

	lineDelChar(c, buf)
}

func lineBegin(c *Client, buf *telnetBuf) {
	for ; buf.linePos > 0; buf.linePos-- {
		cursorLeft(c)
	}
}

func lineEnd(c *Client, buf *telnetBuf) {
	for buf.linePos < buf.lineSize {
		cursorRight(c, buf)
	}
}

func lineDelChar(c *Client, buf *telnetBuf) {
	if buf.lineSize < 1 || buf.linePos >= buf.lineSize {
		return
	}

	buf.lineSize--

	// redraw
	for i := buf.linePos; i < buf.lineSize; i++ {
		buf.lineBuf[i] = buf.lineBuf[i+1] // shift
		drawByte(c, buf.lineBuf[i])
	}
	drawByte(c, ' ') // erase last char

	// reposition cursor
	for i := buf.linePos; i < buf.lineSize+1; i++ {
		cursorLeft(c)
	}
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
	lineEnd(c, buf)
	for buf.linePos > 0 {
		lineBackspace(c, buf)
	}
}

func drawLine(c *Client, buf *telnetBuf) {
	for buf.linePos < buf.lineSize {
		cursorRight(c, buf)
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

	cursorRight(c, buf)
}
