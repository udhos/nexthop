package main

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"

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

func sendPrompt(out chan string, status int) {
	host := "hostname"
	var p string
	switch status {
	case USER:
		p = " login:"
	case PASS:
		host = ""
		p = "password:"
	case EXEC:
		p = ">"
	case ENAB:
		p = "#"
	case CONF:
		p = "(conf)#"
	case QUIT:
		return // no prompt
	default:
		p = "?"
	}
	out <- fmt.Sprintf("\r\n%s%s ", host, p)
}

func execute(c *TelnetClient, line string) {
	log.Printf("execute: [%v]", line)
	c.userOut <- fmt.Sprintf("echo: [%v]\r\n", line)

	if line == "" {
		return
	}

	if strings.HasPrefix("quit", line) {
		c.userOut <- fmt.Sprintf("bye\r\n")
		c.status = QUIT
		close(c.quit)
	}
}

func command(c *TelnetClient, line string) {
	//log.Printf("command: [%v]", line)
	//c.userOut <- fmt.Sprintf("echo: [%v]\r\n", line)

	switch c.status {
	case MOTD:
		// hello banner
		c.userOut <- fmt.Sprintf("\r\nrib server ready\r\n")
		c.status = USER
	case USER:
		c.status = PASS
	case PASS:
		c.status = EXEC
	case EXEC, ENAB, CONF:
		execute(c, line)
	default:
		log.Printf("unknown state for command: [%s]", line)
		c.userOut <- fmt.Sprintf("unknown state for command: [%s]\r\n", line)
	}

	sendPrompt(c.userOut, c.status)
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
		case cmd := <-cmdInput:
			command(cmd.client, cmd.line)
		}
	}
}
