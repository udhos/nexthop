package fwd

import (
	"fmt"
	"log"
	"net"
)

type Dataplane interface {
	InterfaceVrf(ifname, vrfname string) error
	InterfaceAddressAdd(ifname, addr string) error
	InterfaceAddressDel(ifname, addr string) error
	InterfaceAddressGet(ifname string) ([]string, error)
	Interfaces() ([]string, []string, error)
}

func NewDataplane(dataplaneName string) Dataplane {

	log.Printf("NewDataplane: forwarding engine: %s", dataplaneName)

	var engine Dataplane

	switch dataplaneName {
	case "native":
		engine = NewDataplaneNative()
	case "bogus":
		engine = NewDataplaneBogus()
	case "interactive":
		engine = nil
		panic(fmt.Sprintf("NewDataplane: FIXME WRITEME dataplane: %s", dataplaneName))
	case "simulator":
		engine = nil
		panic(fmt.Sprintf("NewDataplane: FIXME WRITEME dataplane: %s", dataplaneName))
	default:
		panic(fmt.Sprintf("NewDataplane: unsupported dataplane: %s", dataplaneName))
	}

	return engine
}

func intersect(n1, n2 *net.IPNet) bool {
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}
