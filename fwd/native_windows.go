package fwd

func NewDataplaneNative() *windowsDataplane {
	return &windowsDataplane{}
}

type windowsDataplane struct {
}

func (d *windowsDataplane) InterfaceVrf(ifname, vrfname string) error {
	return nil
}

func (d *windowsDataplane) InterfaceAddressAdd(ifname, addr string) error {
	return nil
}

func (d *windowsDataplane) InterfaceAddressDel(ifname, addr string) error {
	return nil
}

func (d *windowsDataplane) InterfaceAddressGet(ifname string) ([]string, error) {
	return nil, nil
}

func (d *windowsDataplane) Interfaces() ([]string, []string, error) {
	return nil, nil, nil
}
