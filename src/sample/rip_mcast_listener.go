package main

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/net/ipv4"
)

func main() {
	if err := interfaceAdd("eth2"); err != nil {
		log.Printf("main: error: %v", err)
	}

	log.Printf("main: waiting forever")
	<-make(chan int)
}

func interfaceAdd(s string) error {

	iface, err1 := net.InterfaceByName(s)
	if err1 != nil {
		return err1
	}

	addrList, err2 := iface.Addrs()
	if err2 != nil {
		return err2
	}

	for _, a := range addrList {
		addr, _, err3 := net.ParseCIDR(a.String())
		if err3 != nil {
			log.Printf("interfaceAdd: parse CIDR error for '%s' on '%s': %v", addr, s, err3)
			continue
		}
		if err := join(iface, addr); err != nil {
			log.Printf("interfaceAdd: join error for '%s' on '%s': %v", addr, s, err)
		}
	}

	return nil
}

func join(iface *net.Interface, addr net.IP) error {
	proto := "udp"
	var a string
	if addr.To4() == nil {
		// IPv6
		a = fmt.Sprintf("[%s]", addr.String())
	} else {
		// IPv4
		a = addr.String()
	}

	hostPort := fmt.Sprintf("%s:520", a) // rip multicast port

	// open socket (connection)
	conn, err2 := net.ListenPacket(proto, hostPort)
	if err2 != nil {
		return fmt.Errorf("join: %s/%s listen error: %v", proto, hostPort, err2)
	}

	// join multicast address
	pc := ipv4.NewPacketConn(conn)
	if err := pc.JoinGroup(iface, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 9)}); err != nil {
		conn.Close()
		return fmt.Errorf("join: join error: %v", err)
	}

	if err := pc.SetMulticastInterface(iface); err != nil {
		log.Printf("join: %s SetMulticastInterface(%s) error: %v", a, iface.Name, err)
	}

	{
		ifi, err := pc.MulticastInterface()
		if err != nil {
			log.Printf("join: %s %s multicastInterface error: %v", iface.Name, a, err)
		} else {
			if ifi == nil {
				log.Printf("join: %s %s multicastInterface=nil", iface.Name, a)
			} else {
				log.Printf("join: %s %s multicastInterface=%s", iface.Name, a, ifi.Name)
			}
		}
	}

	// request control messages
	/*
		if err := pc.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
			// warning only
			log.Printf("join: control message flags error: %v", err)
		}
	*/

	go udpReader(pc, iface.Name, addr.String())

	return nil
}

func udpReader(c *ipv4.PacketConn, ifname, ifaddr string) {

	log.Printf("udpReader: reading from '%s' on '%s'", ifaddr, ifname)

	defer c.Close()

	buf := make([]byte, 10000)

	for {
		n, cm, _, err := c.ReadFrom(buf)
		if err != nil {
			log.Printf("udpReader: ReadFrom: error %v", err)
			break
		}

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		log.Printf("udpReader: recv %d bytes from %s to %s on %s", n, cm.Src, cm.Dst, ifname)
	}

	log.Printf("udpReader: exiting '%s'", ifname)
}
