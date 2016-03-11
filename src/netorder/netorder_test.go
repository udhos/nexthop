package netorder

import (
	//"encoding/binary"
	"testing"
)

/*
func TestEndian(t *testing.T) {
	if findEndian() != binary.BigEndian {
		t.Errorf("only big endian is currently supported")
	}
}
*/

func TestUint32(t *testing.T) {
	buf := make([]byte, 4, 4)
	x := uint32(0x01020304)
	WriteUint32(buf, 0, x)

	if buf[0] != 0x01 || buf[1] != 0x02 || buf[2] != 0x03 || buf[3] != 0x04 {
		t.Errorf("bad netorder: uint32=%d buf=%v", x, buf)
	}
}
