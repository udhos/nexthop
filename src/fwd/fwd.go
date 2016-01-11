package fwd

type Dataplane interface {
	InterfaceVrf(ifname, vrfname string) error
	InterfaceAddressAdd(ifname, addr string) error
	InterfaceAddressDel(ifname, addr string) error
	InterfaceAddressGet(ifname string) ([]string, error)
	Interfaces() ([]string, []string, error)
}
