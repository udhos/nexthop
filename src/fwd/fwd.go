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
	InterfaceAddressGet(ifname string) ([]net.IPNet, error)
	VrfAddresses(vrfname string) ([]net.IPNet, error)
	Interfaces() ([]string, []string, error)
	InterfaceVrfGet(ifname string) (string, error)
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
