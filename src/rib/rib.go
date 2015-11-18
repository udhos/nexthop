package main

import (
	//"fmt"
	"log"
	"runtime"
	"time"

	"cli"

	"golang.org/x/net/ipv4" // "code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net
)

func main() {
	log.Printf("rib starting")
	log.Printf("runtime operating system: [%v]", runtime.GOOS)
	log.Printf("CPUs: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	log.Printf("IP version: %v", ipv4.Version)

	cliServer := cli.NewServer()

	go listenTelnet(":2001", cliServer)

	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("rib main: tick")
		case cmd := <-cliServer.CommandChannel:
			//log.Printf("rib main: command: isLine=%v len=%d [%s]", cmd.IsLine, len(cmd.Cmd), cmd.Cmd)
			execute(cmd.Cmd, cmd.IsLine, cmd.Client)
		}
	}
}

func execute(cmd string, isLine bool, c *cli.Client) {
	log.Printf("rib main: execute: isLine=%v cmd=[%s]", isLine, cmd)

	if isLine {
		// single-char command
		return
	}

	// full-line command
}
