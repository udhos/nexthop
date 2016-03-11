package main

import (
	"fmt"
	"net"
	"os"

	"addr"
	"netorder"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Printf("usage:   rip-query host:port     net1 [ net2  ... netN ]\n")
		fmt.Printf("example: rip-query 224.0.0.9:520 1.0.0.0/24       2.0.0.0/24\n")
		return
	}

	query(os.Args[1], os.Args[2:])
}

func query(hostPort string, nets []string) {

	entries := len(nets)
	bufSize := 4 + 20*entries
	buf := make([]byte, bufSize, bufSize)

	buf[0] = 1 // rip request
	buf[1] = 2 // rip version

	for i, n := range nets {
		_, netaddr, err := net.ParseCIDR(n)
		if err != nil {
			fmt.Printf("could not solve network: '%s': %v\n", n, err)
			return
		}

		offset := 4 + 20*i
		netorder.WriteUint16(buf, offset, 2)   // family=AF_INET
		netorder.WriteUint16(buf, offset+2, 0) // route tag
		addr.WriteIPv4(buf, offset+4, netaddr.IP)
		addr.WriteIPv4Mask(buf, offset+8, netaddr.Mask)
		addr.WriteIPv4(buf, offset+12, net.IPv4(0, 0, 0, 0))
		netorder.WriteUint32(buf, offset+16, 0) // metric
	}

	proto := "udp"

	raddr, err := net.ResolveUDPAddr(proto, hostPort)
	if err != nil {
		fmt.Printf("could not solve udp endpoint: '%s': %v\n", hostPort, err)
		return
	}

	conn, err := net.DialUDP(proto, nil, raddr)
	if err != nil {
		fmt.Printf("could not create connection for remote endpoint: %v: %v\n", raddr, err)
		return
	}

	n, err := conn.Write(buf)
	if err != nil {
		fmt.Printf("could not send rip dgram: size=%d to %v: %v\n", len(buf), raddr, err)
		return
	}
	if n != len(buf) {
		fmt.Printf("partil write rip dgram: sent=%d size=%d to %v: %v\n", n, len(buf), raddr, err)
		return
	}

	fmt.Printf("sent rip dgram: size=%d to %v\n", len(buf), raddr)
}
