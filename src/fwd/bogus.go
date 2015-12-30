package fwd

import (
	"fmt"
)

func NewDataplaneBogus() *bogusDataplane {
	return &bogusDataplane{interfaceTable: map[string]bogusIface{}}
}

type bogusIface struct {
	name      string
	addresses []Addr
}

type bogusDataplane struct {
	interfaceTable map[string]bogusIface
}

func (d *bogusDataplane) InterfaceAddressAdd(ifname string, addr Addr) error {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		i = bogusIface{name: ifname}
		d.interfaceTable[ifname] = i
	}
	for _, a := range i.addresses {
		if a == addr {
			return fmt.Errorf("address exists")
		}
	}
	i.addresses = append(i.addresses, addr)
	return nil
}
func (d *bogusDataplane) InterfaceAddressDel(ifname string, addr Addr) error {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		i = bogusIface{name: ifname}
		d.interfaceTable[ifname] = i
	}
	for j, a := range i.addresses {
		if a == addr {
			last := len(i.addresses) - 1
			i.addresses[j] = i.addresses[last]
			i.addresses = i.addresses[:last] // pop
			return nil
		}
	}
	return fmt.Errorf("address not found")
}
func (d *bogusDataplane) InterfaceAddressGet(ifname string) ([]Addr, error) {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		return []Addr{}, fmt.Errorf("interface not found")
	}
	a := append([]Addr{}, i.addresses...) // clone
	return a, nil
}
