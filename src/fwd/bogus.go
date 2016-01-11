package fwd

import (
	"fmt"
	"log"
)

func NewDataplaneBogus() *bogusDataplane {
	d := &bogusDataplane{interfaceTable: map[string]*bogusIface{}}
	d.interfaceAdd("eth0", "")
	d.interfaceAdd("eth1", "")
	d.interfaceAdd("eth2", "")
	d.interfaceAdd("eth3", "VRF1")
	d.interfaceAdd("eth4", "VRF1")
	d.interfaceAdd("eth5", "VRF2")
	return d
}

type bogusIface struct {
	name      string
	addresses []string
	vrf       string
}

type bogusDataplane struct {
	interfaceTable map[string]*bogusIface
}

func (d *bogusDataplane) InterfaceVrf(ifname, vrfname string) error {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		return fmt.Errorf("InterfaceVrf: interface not found")
	}
	i.vrf = vrfname
	return nil
}

func (d *bogusDataplane) interfaceAdd(ifname, vrfname string) {
	log.Printf("bogusDataplane.interfaceAdd: ifname=%s on vrf=[%s]", ifname, vrfname)
	i, ok := d.interfaceTable[ifname]
	if !ok {
		i = &bogusIface{name: ifname}
		d.interfaceTable[ifname] = i
	}
	i.vrf = vrfname
}

func (d *bogusDataplane) InterfaceAddressAdd(ifname, addr string) error {
	i, ok := d.interfaceTable[ifname]
	if !ok {
		return fmt.Errorf("InterfaceAddressAdd: interface not found")
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
func (d *bogusDataplane) Interfaces() ([]string, []string, error) {
	var ifnames, vrfnames []string
	for ifname, i := range d.interfaceTable {
		ifnames = append(ifnames, ifname)
		vrfnames = append(vrfnames, i.vrf)
	}
	return ifnames, vrfnames, nil
}
