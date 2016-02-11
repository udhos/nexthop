package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"golang.org/x/net/ipv4"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("usage:   %s interface protocol address:port\n", os.Args[0])
		fmt.Printf("example: %s eth2      udp      1.0.0.2:520\n", os.Args[0])
		return
	}

	ifname := os.Args[1]
	proto := os.Args[2]
	addrPort := os.Args[3]

	if err := multicastRead(ifname, proto, addrPort); err != nil {
		log.Printf("main: error: %v", err)
	}
}

func multicastRead(ifname, proto, addrPort string) error {

	iface, err1 := net.InterfaceByName(ifname)
	if err1 != nil {
		return err1
	}

	// open/bind socket
	conn, err3 := net.ListenPacket(proto, addrPort)
	if err3 != nil {
		return fmt.Errorf("join: %s/%s listen error: %v", proto, addrPort, err3)
	}

	// join multicast address
	p := ipv4.NewPacketConn(conn)
	if err := p.JoinGroup(iface, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 9)}); err != nil {
		conn.Close()
		return fmt.Errorf("join: join error: %v", err)
	}

	// is this needed for receive?
	if err := p.SetMulticastInterface(iface); err != nil {
		log.Printf("join: %s SetMulticastInterface(%s) error: %v", addrPort, iface.Name, err)
	}

	{
		ifi, err := p.MulticastInterface()
		if err != nil {
			log.Printf("join: %s %s multicastInterface error: %v", iface.Name, addrPort, err)
		} else {
			if ifi == nil {
				log.Printf("join: %s %s multicastInterface=nil", iface.Name, addrPort)
			} else {
				log.Printf("join: %s %s multicastInterface=%s", iface.Name, addrPort, ifi.Name)
			}
		}
	}

	// request control messages
	if err := p.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		// warning only
		log.Printf("join: control message flags error: %v", err)
	}

	udpReader(p, iface.Name, addrPort)

	return nil
}

func udpReader(c *ipv4.PacketConn, ifname, hostPort string) {

	log.Printf("udpReader: reading multicast from '%s' (hostPort: '%s')", ifname, hostPort)

	defer c.Close()

	buf := make([]byte, 10000)

	for {
		n, cm, _, err := c.ReadFrom(buf)
		if err != nil {
			log.Printf("udpReader: ReadFrom: error %v", err)
			break
		}

		log.Printf("udpReader: recv %d bytes from %s to %s on %s", n, cm.Src, cm.Dst, ifname)
	}

	log.Printf("udpReader: exiting '%s'", ifname)
}
