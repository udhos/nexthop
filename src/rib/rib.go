package main

import (
	"fmt"
	"log"
	"net"
	"runtime"

	"code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net

	"rib/iface"
	"rib/route"
)

func localAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
		return
	}
	for _, i := range ifaces {
		addrs, err := iface.GetInterfaceAddrs(i)
		if err != nil {
			log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
			continue
		}
		for _, a := range addrs {
			log.Printf("index=%v iface=%v addr=[%v]\n", i.Index, i.Name, a)
		}
	}
}

func main() {
	log.Printf("runtime operating system: [%v]", runtime.GOOS)

	log.Printf("IP version: %v", ipv4.Version)

	go listenTelnet(":1234")

	localAddresses()

	route.Routes()

	for {
		select {
		case r, ok := <-route.RouteAdd:
			if !ok {
				log.Printf("Routes: quit")
				break
			}
			log.Printf("route add: %v", r)
		case r := <-route.RouteDel:
			log.Printf("route del: %v", r)
		case c := <-cmdInput:
			log.Printf("command: [%v]", c.line)
			c.client.userOut <- fmt.Sprintf("echo: [%v]\r\n", c.line)
		}
	}
}
