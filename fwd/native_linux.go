package fwd

import (
	"fmt"
	"log"
	"net"
	"syscall"

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
				t := "uknown"
				switch linkUpdate.Header.Type {
				case syscall.RTM_NEWLINK:
					t = "new_link"
				case syscall.RTM_DELLINK:
					t = "delete_link"
				}

				isUp := linkUpdate.Flags&syscall.IFF_UP != 0
				isRunning := linkUpdate.Flags&syscall.IFF_RUNNING != 0

				// RTM_DELLINK: interface permanently removed from system
				// RTM_NEWLINK: interface added or changed
				//              IFF_UP: interface administratively enabled
				//              IFF_RUNNING: interface operational (cable attached)

				log.Printf("linux dataplane: link update: type=%s link=[%s] up=%v running=%v",
					t, linkUpdate.Link.Attrs().Name, isUp, isRunning)
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

func (d *linuxDataplane) VrfAddresses(vrfname string) ([]net.IPNet, error) {
	log.Printf("linuxDataplane.VrfAddresses(vrfname=[%s]): FIXME WRITEME", vrfname)
	return nil, nil
}

func (d *linuxDataplane) InterfaceAddressGet(ifname string) ([]net.IPNet, error) {
	link, err1 := netlink.LinkByName(ifname)
	if err1 != nil {
		return nil, fmt.Errorf("linuxDataplane.InterfaceAddressGet: netlink LinkByName error: %v", err1)
	}

	addrs, err2 := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err2 != nil {
		return nil, fmt.Errorf("linuxDataplane.InterfaceAddressGet: netlink AddrList error: %v", err2)
	}

	addrList := []net.IPNet{}

	for _, a := range addrs {
		addrList = append(addrList, *a.IPNet)
	}

	return addrList, nil
}

func (d *linuxDataplane) Interfaces() ([]string, []string, error) {
	links, err1 := netlink.LinkList()
	if err1 != nil {
		return nil, nil, fmt.Errorf("linuxDataplane.Interfaces: netlink.LinkList error: %v", err1)
	}
	ifaces := []string{}
	vrfs := []string{}
	for _, l := range links {
		ifname := l.Attrs().Name
		ifaces = append(ifaces, ifname)
		vrfname, _ := d.InterfaceVrfGet(ifname)
		vrfs = append(vrfs, vrfname)
	}
	return ifaces, vrfs, nil
}

func (d *linuxDataplane) InterfaceVrfGet(ifname string) (string, error) {
	log.Printf("linuxDataplane.InterfaceVrfGet(%s): FIXME WRITEME", ifname)
	return "", nil
}
