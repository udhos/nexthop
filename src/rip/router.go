package main

import (
	"fmt"
	"log"
	"net"
	//"sync"
	"time"

	"golang.org/x/net/ipv4"

	"addr"
	"command"
	"fwd"
	"netorder"
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

	srcIfIndex int
	srcIfName  string
	srcRouter  net.IP
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

	v.RouteLocalAdd(ipnet, cost)

	return nil
}

func (v *ripVrf) RouteLocalAdd(ipnet *net.IPNet, metric int) {
	log.Printf("ripVrf.RouteAdd: %v/%d", ipnet, metric)
}

func (v *ripVrf) RouteLocalDel(ipnet *net.IPNet, metric int) {
	log.Printf("ripVrf.RouteDel: %v/%d", ipnet, metric)
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

			metric := n.metric // save

			last := len(v.nets) - 1
			v.nets[i] = v.nets[last] // overwrite position with last pointer
			v.nets[last] = nil       // free last pointer for garbage collection
			v.nets = v.nets[:last]   // shrink

			v.RouteLocalDel(ipnet, metric)

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
	hardware    fwd.Dataplane
	conf        command.ConfContext
}

const (
	RIP_PORT            = 520
	RIP_METRIC_INFINITY = 16
	RIP_REQUEST         = 1
	RIP_RESPONSE        = 2
	RIP_FAMILY_INET     = 2 // AF_INET
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
func NewRipRouter(hw fwd.Dataplane, ctx command.ConfContext) *RipRouter {

	RIP_GROUP := net.IPv4(224, 0, 0, 9)

	r := &RipRouter{done: make(chan int), input: make(chan *udpInfo), group: RIP_GROUP, readerDone: make(chan int), hardware: hw, conf: ctx}

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

	vrf, err := r.hardware.InterfaceVrfGet(u.ifName)
	if err != nil {
		log.Printf("parseRipPacket: unable to find VRF for interface '%s': %v", u.ifName, err)
		return
	}

	port := r.getInterfaceByIndex(u.ifIndex)
	if port == nil {
		log.Printf("ripRequest: unable to find RIP interface for incoming %v to %v on %s ifIndex=%d",
			&u.src, &u.dst, u.ifName, u.ifIndex)
		return
	}

	switch cmd {
	case RIP_REQUEST:
		ripRequest(r, u, port, size, version, entries, vrf)
	case RIP_RESPONSE:
		ripResponse(r, u, port, size, version, entries, vrf)
	default:
		log.Printf("parseRipPacket: unknown command %d version=%d size=%d from %v to %v on %s ifIndex=%d",
			cmd, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)
	}

}

func ripRequest(r *RipRouter, u *udpInfo, p *port, size, version, entries int, vrf string) {
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

	u.info[0] = RIP_RESPONSE // change command to rip response

	// Update metric for every network in the request

	for i := 0; i < entries; i++ {
		_, _, addr, _, _ := parseEntry(u.info, i)
		route := r.lookupAddress(vrf, addr)
		var metric int
		if route == nil {
			metric = RIP_METRIC_INFINITY
		} else {
			metric = route.metric
		}

		setEntryMetric(u.info, i, metric)
	}

	// Echo request back to source
	ripSend(p.msock.U, &u.src, u.info, u.ifName, u.ifIndex)
}

func ripSend(conn *net.UDPConn, dst *net.UDPAddr, buf []byte, ifname string, ifindex int) {

	// Set 500 ms timeout
	timeout := time.Duration(500) * time.Millisecond
	deadline := time.Now().Add(timeout)
	conn.SetWriteDeadline(deadline)

	size := len(buf)

	n, err := conn.WriteToUDP(buf, dst)
	if err != nil {
		log.Printf("ripSend: error writing back to %v on %s ifIndex=%d: %v", dst, ifname, ifindex, err)
	}
	if n != size {
		log.Printf("ripSend: partial %d/%d write back to %v on %s ifIndex=%d: %v", n, size, dst, ifname, ifindex, err)
	}

	log.Printf("ripSend: wrote back to %v on %s ifIndex=%d", dst, ifname, ifindex)
}

func setEntryMetric(buf []byte, entry, metric int) {
	offset := 4 + 20*entry
	netorder.WriteUint32(buf, offset+16, uint32(metric))
}

func parseEntry(buf []byte, entry int) (family int, tag int, netaddr net.IPNet, nexthop net.IP, metric int) {
	offset := 4 + 20*entry

	family = int(netorder.ReadUint16(buf, offset))
	tag = int(netorder.ReadUint16(buf, offset+2))
	netaddr = net.IPNet{IP: addr.ReadIPv4(buf, offset+4), Mask: addr.ReadIPv4Mask(buf, offset+8)}
	nexthop = addr.ReadIPv4(buf, offset+12)
	metric = int(netorder.ReadUint32(buf, offset+16))

	// log.Printf("parseEntry: entry=%d family=%d tag=%d addr=%v nexthop=%v metric=%v", entry, family, tag, &netaddr, nexthop, metric)

	return
}

func ripResponse(r *RipRouter, u *udpInfo, p *port, size, version, entries int, vrf string) {
	log.Printf("ripResponse: entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
		entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)

	/*
		RFC2453 3.9.2 Response Messages
		The Response must be ignored if it is not from the RIP port.
	*/
	if u.src.Port != RIP_PORT {
		return
	}

	/*
		RFC2453 3.9.2 Response Messages
		the source of the datagram must be on a directly-connected network.
	*/
	ifaceAddrs, err1 := r.hardware.InterfaceAddressGet(u.ifName)
	if err1 != nil {
		log.Printf("ripResponse: unable to find addresses for interface %s: %v",
			u.ifName, err1)
		return
	}

	found := false
	for _, a := range ifaceAddrs {
		if a.Contains(u.src.IP) {
			found = true
			break
		}
	}
	if !found {
		return // ignore response from non-directly-connected address
	}

	/*
		RFC2453 3.9.2 Response Messages
		Ignore packets from our addresses.
		(But only on the interface's VRF, because it is ok to
	*/
	vrfAddresses, err2 := r.hardware.VrfAddresses(vrf)
	if err2 != nil {
		log.Printf("ripResponse: unable to find addresses for VRF %s: %v",
			vrf, err1)
		return
	}
	for _, a := range vrfAddresses {
		if a.IP.Equal(u.src.IP) {
			return // ignore packets from our addresses
		}
	}

	log.Printf("ripResponse: VALID RESPONSE entries=%d version=%d size=%d from %v to %v on %s ifIndex=%d",
		entries, version, size, &u.src, &u.dst, u.ifName, u.ifIndex)

	for i := 0; i < entries; i++ {
		family, tag, netaddr, nexthop, metric := parseEntry(u.info, i)

		/*
			log.Printf("ripResponse: entry=%d/%d family=%d tag=%d net=%v nexthop=%v metric=%d",
				i, entries, family, tag, &netaddr, nexthop, metric)
		*/

		if metric < 1 || metric > RIP_METRIC_INFINITY {
			log.Printf("ripResponse: bad metric entry=%d/%d family=%d tag=%d net=%v nexthop=%v metric=%d from %v to %v on %s ifIndex=%d",
				i, entries, family, tag, &netaddr, nexthop, metric, &u.src, &u.dst, u.ifName, u.ifIndex)
			continue // ignore entry with bad metric
		}

		newMetric := metric + getInterfaceRipCost(r.conf, u.ifName)
		if newMetric > RIP_METRIC_INFINITY {
			newMetric = RIP_METRIC_INFINITY
		}

		if newMetric == RIP_METRIC_INFINITY {
			continue // ignore entry with infinity metric
		}
	}

}

func getInterfaceRipCost(ctx command.ConfContext, ifname string) int {
	//
	// CAUTION: Concurrent access to command.ConfContext
	//
	// RIP main goroutine has full (unprotected) access to command.ConfContext
	// RIP router goroutine should synchronize its access to command.ConfContext
	//

	log.Printf("getInterfaceRipCost(%s): FIXME WRITE", ifname)

	return 1
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

func (r *RipRouter) getInterfaceByIndex(ifIndex int) *port {

	for _, p := range r.ports {
		if ifIndex == p.iface.Index {
			return p
		}
	}

	return nil
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
