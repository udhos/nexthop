package main

import (
	//"fmt"
	"log"
	"runtime"
	"time"

	"golang.org/x/net/ipv4" // "code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net
)

func main() {
	log.Printf("rib starting")
	log.Printf("runtime operating system: [%v]", runtime.GOOS)
	log.Printf("CPUs: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	log.Printf("IP version: %v", ipv4.Version)

	go listenTelnet(":2001")

	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("rib main: tick")
		}
	}
}
