package fwd

import (
	"fmt"
	"log"

	"github.com/udhos/netlink"
)

func NewDataplaneNative() *linuxDataplane {

	log.Printf("NewDataplaneNative: Linux dataplane")

	linkUpdateCh := make(chan netlink.LinkUpdate)
	linkDone := make(chan struct{})

	if err := netlink.LinkSubscribe(linkUpdateCh, linkDone); err != nil {
		panic(fmt.Sprintf("Linux NewDataplaneNative: netlink.LinkSubscribe: error: %v", err))
	}

	addrUpdateCh := make(chan netlink.AddrUpdate)
	addrDone := make(chan struct{})

	if err := netlink.AddrSubscribe(addrUpdateCh, addrDone); err != nil {
		panic(fmt.Sprintf("Linux NewDataplaneNative: netlink.AddrSubscribe: error: %v", err))
	}

	go func() {
		log.Printf("NewDataplaneNative: reading netlink updates")

		for {
			select {
			case linkUpdate := <-linkUpdateCh:
				log.Printf("linux dataplane: link update: %s", linkUpdate.Link.Attrs().Name)
			case addrUpdate := <-addrUpdateCh:
				linkName := index2name(addrUpdate.LinkIndex)
				log.Printf("linux dataplane: addr update: new=%v link=[%s] index=%d addr=[%s]",
					addrUpdate.NewAddr, linkName, addrUpdate.LinkIndex, addrUpdate.LinkAddress)
			}
		}
	}()

	return &linuxDataplane{}
}

func index2name(index int) string {
	link, err := netlink.LinkByIndex(index)
	if err != nil {
		log.Printf("linux dataplane: index2name: netlink.LinkByIndex(%d) error: %v", index, err)
	}
	if link == nil {
		return "<ifname?>"
	}
	return link.Attrs().Name
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
