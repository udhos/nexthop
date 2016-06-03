package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/net/ipv4"
)

func main() {
	prog := "mcast_listener"

	if len(os.Args) != 5 {
		fmt.Printf("usage:   %s interface group     laddress:lport raddress:rport\n", prog)
		fmt.Printf("example: %s eth2      224.0.0.9 0.0.0.0:2000   1.0.0.1:3000\n", prog)
		return
	}

	ifname := os.Args[1]
	group := os.Args[2]
	locAddrPort := os.Args[3]
	remAddrPort := os.Args[4]

	mcast(ifname, group, locAddrPort, remAddrPort)
}

func mcast(ifname, group, locAddrPort, remAddrPort string) {
	locAddr, locPort := splitHostPort(locAddrPort)
	p, err1 := strconv.Atoi(locPort)
	if err1 != nil {
		log.Fatal(err1)
	}

	remAddr, err2 := net.ResolveUDPAddr("udp", remAddrPort)
	if err2 != nil {
		log.Fatal(err2)
	}

	la := net.ParseIP(locAddr)
	if la == nil {
		log.Fatal(fmt.Errorf("bad address: '%s'", locAddr))
	}

	g := net.ParseIP(group)
	if g == nil {
		log.Fatal(fmt.Errorf("bad group: '%s'", group))
	}

	ifi, err3 := net.InterfaceByName(ifname)
	if err3 != nil {
		log.Fatal(err2)
	}

	c, u, err3 := mcastOpen(la, p, ifname)
	if err3 != nil {
		log.Fatal(err3)
	}

	if err := c.JoinGroup(ifi, &net.UDPAddr{IP: g}); err != nil {
		log.Fatal(err)
	}

	if err := c.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		log.Fatal(err)
	}

	log.Printf("writing one udp unicast packet to: %v", remAddr)

	n, err4 := u.WriteToUDP([]byte{0}, remAddr)
	if err4 != nil {
		log.Fatal(err4)
	}

	log.Printf("wrote one udp unicast packet (%d bytes) to %v", n, remAddr)

	readLoop(c)

	c.Close()
}

func splitHostPort(hostPort string) (string, string) {
	s := strings.Split(hostPort, ":")
	host := s[0]
	if host == "" {
		host = "0.0.0.0"
	}
	if len(s) == 1 {
		return host, ""
	}
	return host, s[1]
}

func mcastOpen(bindAddr net.IP, port int, ifname string) (*ipv4.PacketConn, *net.UDPConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		log.Fatal(err)
	}
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		log.Fatal(err)
	}
	//syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
	if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
		log.Fatal(err)
	}

	lsa := syscall.SockaddrInet4{Port: port}
	copy(lsa.Addr[:], bindAddr.To4())

	if err := syscall.Bind(s, &lsa); err != nil {
		syscall.Close(s)
		log.Fatal(err)
	}
	f := os.NewFile(uintptr(s), "")
	c, err := net.FilePacketConn(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	u := c.(*net.UDPConn)
	p := ipv4.NewPacketConn(c)

	return p, u, nil
}

func readLoop(c *ipv4.PacketConn) {

	log.Printf("readLoop: reading")

	buf := make([]byte, 10000)

	for {
		n, cm, _, err1 := c.ReadFrom(buf)
		if err1 != nil {
			log.Printf("readLoop: ReadFrom: error %v", err1)
			break
		}

		var name string

		ifi, err2 := net.InterfaceByIndex(cm.IfIndex)
		if err2 != nil {
			log.Printf("readLoop: unable to solve ifIndex=%d: error: %v", cm.IfIndex, err2)
		}

		if ifi == nil {
			name = "ifname?"
		} else {
			name = ifi.Name
		}

		log.Printf("readLoop: recv %d bytes from %s to %s on %s", n, cm.Src, cm.Dst, name)
	}

	log.Printf("readLoop: exiting")
}
