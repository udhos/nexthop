package addr

import (
	"fmt"
	//"log"
	"net"
)

func CheckMask(n *net.IPNet) error {
	if ones, bits := n.Mask.Size(); ones == 0 && bits == 0 {
		return fmt.Errorf("bad netmask: addr=[%v]", n)
	}
	return nil
}

func NetEqual(n1, n2 *net.IPNet) bool {
	ones1, bits1 := n1.Mask.Size()
	ones2, bits2 := n2.Mask.Size()
	return ones1 == ones2 && bits1 == bits2 && n1.IP.Equal(n2.IP)
}

func NetIntersect(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}

func ReadIPv4(buf []byte, offset int) net.IP {
	return net.IPv4(buf[offset], buf[offset+1], buf[offset+2], buf[offset+3])
}

func ReadIPv4Mask(buf []byte, offset int) net.IPMask {
	return net.IPv4Mask(buf[offset], buf[offset+1], buf[offset+2], buf[offset+3])
}

func WriteIPv4(buf []byte, offset int, ipaddr net.IP) {
	buf[offset] = ipaddr[0]
	buf[offset+1] = ipaddr[1]
	buf[offset+2] = ipaddr[2]
	buf[offset+3] = ipaddr[3]
}

func WriteIPv4Mask(buf []byte, offset int, ipaddr net.IPMask) {
	buf[offset] = ipaddr[0]
	buf[offset+1] = ipaddr[1]
	buf[offset+2] = ipaddr[2]
	buf[offset+3] = ipaddr[3]
}
