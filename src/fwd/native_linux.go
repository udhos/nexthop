package fwd

import (
	"fmt"
	"log"

	"github.com/udhos/netlink"
)

func NewDataplaneNative() *linuxDataplane {

	log.Printf("NewDataplaneNative: Linux dataplane")

	update := make(chan netlink.LinkUpdate)
	done := make(chan struct{})

	if err := netlink.LinkSubscribe(update, done); err != nil {
		panic(fmt.Sprintf("Linux NewDataplaneNative: netlink.LinkSubscribe: error: %v", err))
	}

	go func() {
		log.Printf("NewDataplaneNative: reading netlink updates")

		for {
			select {
			case linkUpdate := <-update:
				log.Printf("linux dataplane: link update: %s", linkUpdate.Link.Attrs().Name)
			}
		}
	}()

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
