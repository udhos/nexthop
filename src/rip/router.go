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

type RipRouter struct {
	done        chan int // write into this channel (do not close) to request end of rip router
	input       chan udpInfo
	nets        []*net.IPNet // locally generated networks
	ports       []*port      // rip interfaces
	group       net.IP       // 224.0.0.9
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
	dst     net.IP
	ifIndex int
	ifName  string
}

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
					len(u.info), &u.src, u.dst, u.ifName, u.ifIndex)
			}
		}

		log.Printf("rip router: goroutine finished")
	}()

	return r
}

func (r *RipRouter) NetAdd(s string) error {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("RipRouter.NetAdd: parse error: addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("RipRouter.NetAdd: bad mask: addr=[%s]: %v", s, err1)
	}
	for _, a := range r.nets {
		if addr.NetEqual(ipnet, a) {
			// found
			return nil
		}
	}
	// not found
	r.nets = append(r.nets, ipnet) // add
	return nil
}

func (r *RipRouter) NetDel(s string) error {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("RipRouter.NetAdd: parse error: addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("RipRouter.NetDel: bad mask: addr=[%s]: %v", s, err1)
	}
	for i, a := range r.nets {
		if addr.NetEqual(ipnet, a) {
			// found

			last := len(r.nets) - 1
			r.nets[i] = r.nets[last] // overwrite position with last pointer
			r.nets[last] = nil       // free last pointer for garbage collection
			r.nets = r.nets[:last]   // shrink

			return nil
		}
	}
	// not found
	return nil
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

	m, err1 := sock.MulticastListener(RIP_PORT, ifi.Name)
	if err1 != nil {
		return fmt.Errorf("RipRouter.Join: open: %v", err1)
	}

	if err := sock.Join(m, r.group, ifi.Name); err != nil {
		sock.Close(m)
		return fmt.Errorf("RipRouter.Join: join: %v", err)
	}

	newPort := &port{iface: ifi, msock: m}

	r.ports = append(r.ports, newPort)

	go udpReader(m.P, r.input, ifi.Name, r.readerDone)

	r.readerCount++

	return nil
}

func delInterfaces(r *RipRouter) {
	for i := range r.ports {
		r.ifClose(i)
	}
	r.ports = nil // cleanup
}

func udpReader(c *ipv4.PacketConn, input chan<- udpInfo, ifname string, readerDone chan<- int) {

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

		//log.Printf("udpReader: recv %d bytes from %v to %s on %s ifIndex=%d", n, udpSrc, cm.Dst, name, cm.IfIndex)

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		// deliver udp packet to main rip goroutine
		input <- udpInfo{info: b, src: *udpSrc, dst: cm.Dst, ifIndex: cm.IfIndex, ifName: name}
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
