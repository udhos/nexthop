package fwd

type Addr string

type Dataplane interface {
	InterfaceAddressAdd(ifname string, addr Addr) error
	InterfaceAddressDel(ifname string, addr Addr) error
	InterfaceAddressGet(ifname string) ([]Addr, error)
}
