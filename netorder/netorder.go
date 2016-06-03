package netorder

import (
//"encoding/binary"
//"unsafe"
)

/*
var NativeEndian binary.ByteOrder

func init() {
	NativeEndian = findEndian()
}

func findEndian() binary.ByteOrder {
	var x uint32 = 0x01020304
	switch *(*byte)(unsafe.Pointer(&x)) {
	case 0x01:
		return binary.BigEndian
	case 0x04:
		return binary.LittleEndian
	}
	panic("endian.findEndian: unknown byte order")
}
*/

func ReadUint16(buf []byte, offset int) uint16 {
	return uint16(buf[offset])<<8 + uint16(buf[offset+1])
}

func ReadUint32(buf []byte, offset int) uint32 {
	a := uint32(buf[offset]) << 24
	b := uint32(buf[offset+1]) << 16
	c := uint32(buf[offset+2]) << 8
	d := uint32(buf[offset+3])
	return a + b + c + d
}

func WriteUint16(buf []byte, offset int, value uint16) {
	buf[offset] = byte((value >> 8) & 0xFF)
	buf[offset+1] = byte(value & 0xFF)
}

func WriteUint32(buf []byte, offset int, value uint32) {
	buf[offset] = byte((value >> 24) & 0xFF)
	buf[offset+1] = byte((value >> 16) & 0xFF)
	buf[offset+2] = byte((value >> 8) & 0xFF)
	buf[offset+3] = byte(value & 0xFF)
}
