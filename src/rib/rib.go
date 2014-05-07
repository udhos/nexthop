package main

import (
	"fmt"
	"log"
	"net"

	"code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net
)

func localAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
		return
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Print(fmt.Errorf("localAddresses: %v\n", err.Error()))
			continue
		}
		for _, a := range addrs {
			log.Printf("%v %v\n", i.Name, a)
		}
	}
}

func main() {
	log.Printf("IP version: %v", ipv4.Version)

	localAddresses()
}
