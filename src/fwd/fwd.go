package fwd

import (
	"net"
)

type Dataplane interface {
	InterfaceVrf(ifname, vrfname string) error
	InterfaceAddressAdd(ifname, addr string) error
	InterfaceAddressDel(ifname, addr string) error
	InterfaceAddressGet(ifname string) ([]string, error)
	Interfaces() ([]string, []string, error)
}

func intersect(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}
