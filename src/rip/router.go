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
	addr   *net.IPNet
	metric int
}

type ripVrf struct {
	name string
	nets []*ripNet // locally configured networks
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
		if addr.NetEqual(ipnet, n.addr) {
			// found
			n.metric = cost
			return nil
		}
	}
	// not found
	v.nets = append(v.nets, &ripNet{addr: ipnet, metric: cost}) // add
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
		if addr.NetEqual(ipnet, n.addr) {
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
	input       chan udpInfo
	vrfs        []*ripVrf
	ports       []*port // rip interfaces
	group       net.IP  // 224.0.0.9
	readerDone  chan int
	readerCount int
}

const RIP_PORT = 520

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

	r := &RipRouter{done: make(chan int), input: make(chan udpInfo), group: RIP_GROUP, readerDone: make(chan int)}

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
				log.Printf("rip router: recv %d bytes from %v to %v on %s ifIndex=%d",
					len(u.info), &u.src, &u.dst, u.ifName, u.ifIndex)
			}
		}

		log.Printf("rip router: goroutine finished")
	}()

	return r
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

func delInterfaces(r *RipRouter) {
	for i := range r.ports {
		r.ifClose(i)
	}
	r.ports = nil // cleanup
}

func udpReader(c *ipv4.PacketConn, input chan<- udpInfo, ifname string, readerDone chan<- int, listenPort int) {

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
		input <- udpInfo{info: b, src: *udpSrc, dst: udpDst, ifIndex: cm.IfIndex, ifName: name}
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
