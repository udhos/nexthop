package fwd

func NewDataplaneNative() *linuxDataplane {
	return &linuxDataplane{}
}

type linuxDataplane struct {
}

func (d *linuxDataplane) InterfaceVrf(ifname, vrfname string) error {
	return nil
}

func (d *linuxDataplane) InterfaceAddressAdd(ifname, addr string) error {
	return nil
}

func (d *linuxDataplane) InterfaceAddressDel(ifname, addr string) error {
	return nil
}

func (d *linuxDataplane) InterfaceAddressGet(ifname string) ([]string, error) {
	return nil, nil
}

func (d *linuxDataplane) Interfaces() ([]string, []string, error) {
	return nil, nil, nil
}
