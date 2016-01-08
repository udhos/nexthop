package fwd

import (
	"fmt"
	//"log"
)

func NewDataplaneBogus() *bogusDataplane {
	return &bogusDataplane{interfaceTable: map[string]*bogusIface{}}
}

type bogusIface struct {
	name      string
	addresses []string
}

type bogusDataplane struct {
	interfaceTable map[string]*bogusIface
}

func (d *bogusDataplane) InterfaceAddressAdd(ifname, addr string) error {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		i = &bogusIface{name: ifname}
		d.interfaceTable[ifname] = i
	}
	for _, a := range i.addresses {
		if a == addr {
			return fmt.Errorf("address exists")
		}
	}
	i.addresses = append(i.addresses, addr)
	//log.Printf("InterfaceAddressAdd: %v", i.addresses)
	return nil
}
func (d *bogusDataplane) InterfaceAddressDel(ifname, addr string) error {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		return fmt.Errorf("InterfaceAddressDel: interface not found")
	}
	for j, a := range i.addresses {
		if a == addr {
			last := len(i.addresses) - 1
			i.addresses[j] = i.addresses[last]
			i.addresses = i.addresses[:last] // pop
			//log.Printf("InterfaceAddressDel: %v", i.addresses)
			return nil
		}
	}
	return fmt.Errorf("address not found")
}
func (d *bogusDataplane) InterfaceAddressGet(ifname string) ([]string, error) {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		return []string{}, fmt.Errorf("InterfaceAddressGet: interface not found")
	}
	a := append([]string{}, i.addresses...) // clone
	//log.Printf("InterfaceAddressGet: %v", a)
	return a, nil
}
