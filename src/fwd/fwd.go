package fwd

type Dataplane interface {
	InterfaceAddressAdd(ifname, addr string) error
	InterfaceAddressDel(ifname, addr string) error
	InterfaceAddressGet(ifname string) ([]string, error)
}
