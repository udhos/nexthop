package sock

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
)

type multicastSock struct {
	P *ipv4.PacketConn
	U *net.UDPConn
}

func MulticastListener(port int, ifname string) (*multicastSock, error) {
	s, err1 := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err1 != nil {
		return nil, fmt.Errorf("MulticastListener: could not create socket(port=%d,ifname=%s): %v", port, ifname, err1)
	}
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could not set reuse addr socket(port=%d,ifname=%s): %v", port, ifname, err)
	}
	if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could bind to device socket(port=%d,ifname=%s): %v", port, ifname, err)
	}

	bindAddr := net.IP(net.IPv4(0, 0, 0, 0))
	lsa := syscall.SockaddrInet4{Port: port}
	copy(lsa.Addr[:], bindAddr.To4())

	if err := syscall.Bind(s, &lsa); err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could bind socket to address %v,%d: %v", bindAddr, port, err)
	}
	f := os.NewFile(uintptr(s), "")
	c, err2 := net.FilePacketConn(f)
	f.Close()
	if err2 != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could get packet connection for socket(port=%d,ifname=%s): %v", port, ifname, err2)
	}
	u := c.(*net.UDPConn)
	p := ipv4.NewPacketConn(c)

	return &multicastSock{P: p, U: u}, nil
}

func Join(sock *multicastSock, group net.IP, ifname string) error {
	ifi, err1 := net.InterfaceByName(ifname)
	if err1 != nil {
		return fmt.Errorf("Join: could get find interface %s: %v", ifname, err1)
	}

	if err := sock.P.JoinGroup(ifi, &net.UDPAddr{IP: group}); err != nil {
		return fmt.Errorf("Join: could get join group %v on interface %s: %v", group, ifname, err)
	}

	return nil
}

func Close(sock *multicastSock) {
	sock.P.Close()
	sock.U.Close()
	sock.P = nil
	sock.U = nil
}
