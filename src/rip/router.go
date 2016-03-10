package main

import (
	"fmt"
	"log"
	"net"
	//"sync"
	"time"

	"golang.org/x/net/ipv4"

	"addr"
	"sock"
)

type ripNet struct {
	addr   net.IPNet
	metric int
}

type ripRoute struct {
	family  uint16
	tag     uint16
	addr    net.IPNet
	nexthop net.IP
	metric  int
}

type ripVrf struct {
	name   string
	nets   []*ripNet   // locally configured networks
	routes []*ripRoute // learnt networks
}

// Empty: VRF does not contain any data
func (v *ripVrf) Empty() bool {
	return len(v.nets) < 1
}

func (v *ripVrf) NetAdd(s string, cost int) error {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("ripVrf.NetAdd: parse error: addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetAdd: bad mask: addr=[%s]: %v", s, err1)
	}
	for _, n := range v.nets {
		if addr.NetEqual(ipnet, &n.addr) {
			// found
			n.metric = cost
			return nil
		}
	}
	// not found
	v.nets = append(v.nets, &ripNet{addr: *ipnet, metric: cost}) // add
	return nil
}

func (v *ripVrf) NetDel(s string) error {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("ripVrf.NetDel: parse error: addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("ripVrf.NetDel: bad mask: addr=[%s]: %v", s, err1)
	}
	for i, n := range v.nets {
		if addr.NetEqual(ipnet, &n.addr) {
			// found

			last := len(v.nets) - 1
			v.nets[i] = v.nets[last] // overwrite position with last pointer
			v.nets[last] = nil       // free last pointer for garbage collection
			v.nets = v.nets[:last]   // shrink

			return nil
		}
	}
	// not found
	return nil
}

type RipRouter struct {
	done        chan int // write into this channel (do not close) to request end of rip router
	input       chan *udpInfo
	vrfs        []*ripVrf
	ports       []*port // rip interfaces
	group       net.IP  // 224.0.0.9
	readerDone  chan int
	readerCount int
}

const (
	RIP_PORT            = 520
	RIP_METRIC_INFINITY = 16
)

// rip interface
type port struct {
	iface *net.Interface
	msock *sock.MulticastSock
}

type udpInfo struct {
	info    []byte
	src     net.UDPAddr
	dst     net.UDPAddr
	ifIndex int
	ifName  string
}

// NewRipRouter(): Spawn new rip router.
// Write on RipRouter.done channel (do not close it) to request termination of rip router.
func NewRipRouter() *RipRouter {

	RIP_GROUP := net.IPv4(224, 0, 0, 9)

	r := &RipRouter{done: make(chan int), input: make(chan *udpInfo), group: RIP_GROUP, readerDone: make(chan int)}

	addInterfaces(r)

	go func() {
		log.Printf("rip router: goroutine started")

		tick := time.Duration(10)
		ticker := time.NewTicker(time.Second * tick)

	LOOP:
		for {
			select {
			case <-ticker.C:
				log.Printf("rip router: %ds tick", tick)
			case <-r.done:
				// finish requested
				log.Printf("rip router: finish request received")
				delInterfaces(r) // break udpReader goroutines
			case <-r.readerDone:
				// one udpReader goroutine finished
				r.readerCount--
				if r.readerCount < 1 {
					// all udpReader goroutines finished
					break LOOP
				}
			case u, ok := <-r.input:
				if !ok {
					log.Printf("rip router: udpReader channel closed")
					break LOOP
				}
				parseRipPacket(r, u)
			}
		}

		log.Printf("rip router: goroutine finished")
	}()

	return r
}

func parseRipPacket(r *RipRouter, u *udpInfo) {
	/*
		log.Printf("parseRipPacket: recv %d bytes from %v to %v on %s ifIndex=%d",
			len(u.info), &u.src, &u.dst, u.ifName, u.ifIndex)
	*/

	size := len(u.info)
	entries := (size - 4) / 20
	if entries < 1 {
		log.Printf("parseRipPacket: short packet size=%d bytes from %v to %v on %s ifIndex=%d",
			size, &u.src, &u.dst, u.ifName, u.ifIndex)
		return
	}
	if entries > 25 {
		log.Printf("parseRipPacket: long packet size=%d bytes from %v to %v on %s ifIndex=%d",
			size, &u.src, &u.dst, u.ifName, u.ifIndex)
		return
	}

	cmd := u.info[0]
	version := int(u.info[1])

	/*
		log.Printf("parseRipPacket: entries=%d cmd=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
			entries, cmd, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	*/

	vrf := getInterfaceVrf(u.ifName)

	switch cmd {
	case 1:
		ripRequest(r, u, size, version, entries, vrf)
	case 2:
		ripResponse(r, u, size, version, entries)
	default:
		log.Printf("parseRipPacket: unknown command %d version=%d size=%d from %v to %v on %s ifIndex=%d",
			cmd, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	}

}

func ripRequest(r *RipRouter, u *udpInfo, size, version, entries int, vrf string) {
	log.Printf("ripRequest: entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
		entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)

	if entries == 1 {
		/*
			RFC2453 3.9.1 Request Messages

			There is one special case.
			If there is exactly one entry in the request,
			and it has an address family identifier of
			zero and a metric of infinity (i.e., 16),
			then this is a request to send the entire
			routing table.
		*/
		family, _, _, _, metric := parseEntry(u.info, 0)

		if family == 0 && metric == RIP_METRIC_INFINITY {
			log.Printf("ripRequest: FIXME WRITEME reply with full routing table")
			return
		}
	}

	/*
		RFC2453 3.9.1 Request Messages

		Examine the list of RTEs in the Request one by one.  For
		each entry, look up the destination in the router's routing database
		and, if there is a route, put that route's metric in the metric field
		of the RTE.  If there is no explicit route to the specified
		destination, put infinity in the metric field.  Once all the entries
		have been filled in, change the command from Request to Response and
		send the datagram back to the requestor.
	*/
	for i := 0; i < entries; i++ {
		_, _, addr, _, _ := parseEntry(u.info, 0)
		route := r.lookupAddress(vrf, addr)
		var metric int
		if route == nil {
			metric = RIP_METRIC_INFINITY
		} else {
			metric = route.metric
		}

		setEntryMetric(u.info, i, metric)
	}

	log.Printf("ripRequest: FIXME WRITEME change request to response, echo dgram back")
}

func setEntryMetric(buf []byte, entry, metric int) {
	offset := 4 + 20*entry
	writeUint32(buf, offset+16, uint32(metric))
}

func parseEntry(buf []byte, entry int) (family int, tag int, addr net.IPNet, nexthop net.IP, metric int) {
	offset := 4 + 20*entry

	family = int(readUint16(buf, offset))
	tag = int(readUint16(buf, offset+2))
	addr = net.IPNet{IP: readIPv4(buf, offset+4), Mask: readIPv4Mask(buf, offset+8)}
	nexthop = readIPv4(buf, offset+12)
	metric = int(readUint32(buf, offset+16))

	log.Printf("parseEntry: entry=%d family=%d tag=%d addr=%v nexthop=%v metric=%v", entry, family, tag, &addr, nexthop, metric)

	return
}

func readIPv4(buf []byte, offset int) net.IP {
	return net.IPv4(buf[offset], buf[offset+1], buf[offset+2], buf[offset+3])
}

func readIPv4Mask(buf []byte, offset int) net.IPMask {
	return net.IPv4Mask(buf[offset], buf[offset+1], buf[offset+2], buf[offset+3])
}

func readUint16(buf []byte, offset int) uint16 {
	return uint16(buf[offset])<<8 + uint16(buf[offset+1])
}

func readUint32(buf []byte, offset int) uint32 {
	a := uint32(buf[offset]) << 24
	b := uint32(buf[offset+1]) << 16
	c := uint32(buf[offset+2]) << 8
	d := uint32(buf[offset+3])
	return a + b + c + d
}

func writeUint32(buf []byte, offset int, value uint32) {
	buf[offset] = byte((value >> 24) & 0xFF)
	buf[offset+1] = byte((value >> 16) & 0xFF)
	buf[offset+2] = byte((value >> 8) & 0xFF)
	buf[offset+3] = byte(value & 0xFF)
}

func ripResponse(r *RipRouter, u *udpInfo, size, version, entries int) {
	log.Printf("ripResponse: entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
		entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
}

func (r *RipRouter) lookupAddress(vrf string, addr net.IPNet) *ripRoute {
	log.Printf("RipRouter.lookupAddress(vrf=%s,addr=%s): FIXME WRITEME", vrf, &addr)
	return nil
}

func (r *RipRouter) NetAdd(vrf, s string, cost int) error {
	for _, v := range r.vrfs {
		if v.name == vrf {
			// vrf found
			return v.NetAdd(s, cost)
		}
	}
	// vrf not found
	v := &ripVrf{name: vrf}
	r.vrfs = append(r.vrfs, v) // add vrf
	return v.NetAdd(s, cost)
}

func (r *RipRouter) NetDel(vrf, s string) error {
	for i, v := range r.vrfs {
		if v.name == vrf {
			// vrf found

			err := v.NetDel(s) // remove net from VRF

			if v.Empty() {
				// delete vrf
				last := len(r.vrfs) - 1
				r.vrfs[i] = r.vrfs[last] // overwrite position with last pointer
				r.vrfs[last] = nil       // free last pointer for garbage collection
				r.vrfs = r.vrfs[:last]   // shrink
			}

			return err
		}
	}
	// vrf not found
	return fmt.Errorf("RipRouter.NetDel: vrf not found: vrf=[%s] addr=[%s]", vrf, s)
}

func addInterfaces(r *RipRouter) {
	ifList, err1 := net.Interfaces()
	if err1 != nil {
		log.Printf("NewRipRouter: could not find local interfaces: %v", err1)
	}
	for _, i := range ifList {
		if err := r.InterfaceAdd(i.Name); err != nil {
			log.Printf("NewRipRouter: error adding interface '%s': %v", i.Name, err)
		}
	}
}

func (r *RipRouter) InterfaceAdd(s string) error {

	for _, p := range r.ports {
		if s == p.iface.Name {
			return fmt.Errorf("RipRouter.InterfaceAdd: interface '%s' exists", s)
		}
	}

	ifi, err1 := net.InterfaceByName(s)
	if err1 != nil {
		return err1
	}

	return r.Join(ifi)
}

func (r *RipRouter) Join(ifi *net.Interface) error {

	ripPort := RIP_PORT

	m, err1 := sock.MulticastListener(ripPort, ifi.Name)
	if err1 != nil {
		return fmt.Errorf("RipRouter.Join: open: %v", err1)
	}

	if err := sock.Join(m, r.group, ifi.Name); err != nil {
		sock.Close(m)
		return fmt.Errorf("RipRouter.Join: join: %v", err)
	}

	newPort := &port{iface: ifi, msock: m}

	r.ports = append(r.ports, newPort)

	go udpReader(m.P, r.input, ifi.Name, r.readerDone, ripPort)

	r.readerCount++

	return nil
}

func getInterfaceVrf(ifname string) string {
	log.Printf("getInterfaceVrf(ifname=%s): FIXME WRITEME", ifname)
	return ""
}

func delInterfaces(r *RipRouter) {
	for i := range r.ports {
		r.ifClose(i)
	}
	r.ports = nil // cleanup
}

func udpReader(c *ipv4.PacketConn, input chan<- *udpInfo, ifname string, readerDone chan<- int, listenPort int) {

	log.Printf("udpReader: reading from '%s'", ifname)

	defer c.Close()

	buf := make([]byte, 10000)

LOOP:
	for {
		n, cm, srcAddr, err1 := c.ReadFrom(buf)
		if err1 != nil {
			log.Printf("udpReader: ReadFrom: error %v", err1)
			break LOOP
		}

		var udpSrc *net.UDPAddr

		switch srcAddr.(type) {
		case *net.UDPAddr:
			udpSrc = srcAddr.(*net.UDPAddr)
		}

		var name string

		var ifi *net.Interface
		var err2 error

		if cm != nil {
			ifi, err2 = net.InterfaceByIndex(cm.IfIndex)
			if err2 != nil {
				log.Printf("udpReader: unable to solve ifIndex=%d: error: %v", cm.IfIndex, err2)
			}
		}

		if ifi == nil {
			name = "ifname?"
		} else {
			name = ifi.Name
		}

		udpDst := net.UDPAddr{IP: cm.Dst, Port: listenPort}

		//log.Printf("udpReader: recv %d bytes from %v to %v on %s ifIndex=%d", n, udpSrc, &udpDst, name, cm.IfIndex)

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		// deliver udp packet to main rip goroutine
		input <- &udpInfo{info: b, src: *udpSrc, dst: udpDst, ifIndex: cm.IfIndex, ifName: name}
	}

	log.Printf("udpReader: exiting '%s' -- trying", ifname)
	readerDone <- 1 // tell rip router goroutine
	log.Printf("udpReader: exiting '%s'", ifname)
}

func (r *RipRouter) InterfaceDel(s string) error {
	log.Printf("RipRouter.InterfaceDel: %s", s)

	for i, p := range r.ports {
		if s == p.iface.Name {
			// found interface
			r.ifClose(i)
			r.ifDel(i)
			return nil
		}
	}

	return fmt.Errorf("RipRouter.InterfaceDel: interface '%s' not found", s)
}

func (r *RipRouter) ifClose(i int) {
	p := r.ports[i]

	log.Printf("RipRouter.ifDel: %s", p.iface.Name)

	if err := sock.Leave(p.msock, r.group, p.iface); err != nil {
		// warning only
		log.Printf("RipRouter.InterfaceDel: leave group error: %v", err)
	}

	sock.Close(p.msock) // break reader goroutine
}

func (r *RipRouter) ifDel(i int) {
	size := len(r.ports)
	r.ports[i] = r.ports[size-1]
	r.ports = r.ports[:size-1]
}
