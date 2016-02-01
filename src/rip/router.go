package main

import (
	"log"
	"net"
)

type RipRouter struct {
	done chan int // close this channel to request end of rip router
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
		input <- udpInfo{info: buf[:n], addr: *from}
	}
}

func (r *RipRouter) NetAdd(net string) {
}

func (r *RipRouter) NetDel(net string) {
}
