package iface

import (
	"log"
	"net"
)

type Interface struct {
	Name string
	Addr net.IPNet
}

// getMask: get net.IPNet from net.IPAddr
func getMask(i net.Interface, addr net.IPAddr) net.IPNet {
	// see https://code.google.com/p/go/source/browse/src/pkg/net/interface_linux.go
	return net.IPNet{IP: addr.IP, Mask: net.CIDRMask(16, 8*net.IPv4len)}
}

func GetInterfaceAddrs(i net.Interface) ([]net.Addr, error) {

	addrs, err := i.Addrs()
	if err != nil {
		return addrs, err
	}

	result := []net.Addr{}

	for _, a := range addrs {
		switch ad := a.(type) {
		case *net.IPNet:
			// linux, bsd, darwin, etc...
			result = append(result, a)
		case *net.IPAddr:
			// windows: missing netmask
			log.Printf("GetInterfaceAddrs: net.IPAddr: %v: does not provide netmask", ad)
			ipNet := getMask(i, *ad)
			result = append(result, &ipNet)
		default:
			// does this happen?
			log.Printf("GetInterfaceAddrs: unknown type: %v: does not provide netmask", ad)
			result = append(result, a)
		}
	}

	return result, nil
}
