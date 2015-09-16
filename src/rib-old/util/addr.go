package util

import (
	"net"
)

func IpIsIPv4(ip net.IP) bool {
	p4 := ip.To4()
	return len(p4) == net.IPv4len
}
