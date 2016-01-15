package cli

import (
	"log"
	"sync"
)

type bufByteArray [100]byte // affects max input line length

type telnetBuf struct {
	mutex          *sync.RWMutex // mutex: cli.Client is shared between 2 goroutines: cli.InputLoop and main
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
		mutex:          &sync.RWMutex{},
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

func (buf *telnetBuf) linePosInc() {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.linePos++
}

func (buf *telnetBuf) getByteCurrent() byte {
	defer buf.mutex.RUnlock()
	buf.mutex.RLock()
	return buf.lineBuf[buf.linePos]
}

func (buf *telnetBuf) insert(c *Client, b byte) {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()

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

	log.Printf("telnetBuf.insert: pos=%d size=%d line=[%v]", buf.linePos, buf.lineSize, string(buf.lineBuf[:buf.lineSize]))
}

func (buf *telnetBuf) escapeSet(esc int) {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.escape = esc
}

func (buf *telnetBuf) escapeGet() int {
	defer buf.mutex.RUnlock()
	buf.mutex.RLock()
	return buf.escape
}

func (buf *telnetBuf) hitCR() {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.expectingCtrlM = true
}

func (buf *telnetBuf) notCtrlM() {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.expectingCtrlM = false
}

func (buf *telnetBuf) iacGet() int {
	defer buf.mutex.RUnlock()
	buf.mutex.RLock()
	return buf.iac
}

func (buf *telnetBuf) iacSet(i int) {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.iac = i
}

func (buf *telnetBuf) subBufReset() {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.subSize = 0
}

func (buf *telnetBuf) subBufCopy() []byte {
	defer buf.mutex.RUnlock()
	buf.mutex.RLock()
	return append([]byte{}, buf.subBuf[:buf.subSize]...) // clone
}

func (buf *telnetBuf) subSizeGet() int {
	defer buf.mutex.RUnlock()
	buf.mutex.RLock()
	return buf.subSize
}

func (buf *telnetBuf) pushSub(b byte) {
	defer buf.mutex.Unlock()
	buf.mutex.Lock()
	buf.subSize = pushSub(buf.subBuf[:], buf.subSize, b)
}

func pushSub(buf []byte, size int, b byte) int {
	max := len(buf)

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
