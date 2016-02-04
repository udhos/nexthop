package main

import (
	"fmt"
	"log"
	"net"

	"addr"
)

type RipRouter struct {
	done chan int     // close this channel to request end of rip router
	nets []*net.IPNet // locally generated networks
}

type udpInfo struct {
	info []byte
	addr net.UDPAddr
}

func NewRipRouter() *RipRouter {

	proto := "udp"
	hostPort := ":520"

	addr, err1 := net.ResolveUDPAddr(proto, hostPort)
	if err1 != nil {
		log.Printf("NewRipRouter: bad UDP addr=%s/%s: %v", proto, hostPort, err1)
		return nil
	}

	log.Printf("NewRipRouter: reading from: %v", addr)

	conn, err2 := net.ListenUDP(proto, addr)
	if err2 != nil {
		log.Printf("NewRipRouter: listen error addr=%s/%s: %v", proto, hostPort, err2)
		return nil
	}

	input := make(chan udpInfo)

	go udpReader(conn, input)

	r := &RipRouter{done: make(chan int)}

	go func() {
		log.Printf("rip router goroutine started")

		defer conn.Close()

	LOOP:
		for {
			select {
			case <-r.done:
				// finish requested
				break LOOP
			case u, ok := <-input:
				if !ok {
					log.Printf("rip router: udpReader channel closed")
					break LOOP
				}
				log.Printf("rip router: recv %d bytes from %v", len(u.info), u.addr)
			}
		}

		log.Printf("rip router goroutine finished")
	}()

	return r
}

func udpReader(conn *net.UDPConn, input chan<- udpInfo) {
	buf := make([]byte, 10000)

	defer close(input)

	for {
		n, from, err := conn.ReadFromUDP(buf)
		log.Printf("udpReader: %d bytes from %v: error: %v", n, from, err)
		if err != nil {
			log.Printf("udpReader: error: %v", err)
			break
		}

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		input <- udpInfo{info: b, addr: *from}
	}
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
