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

/*
TODO: Fetch interfaces names on windows:

C:\>netsh interface ipv4 show interfaces

Idx     Met         MTU          State                Name
---  ----------  ----------  ------------  ---------------------------
  1          50  4294967295  connected     Loopback Pseudo-Interface 1
 14          20        1500  connected     Local Area Connection
 10           5        1400  disconnected  Local Area Connection* 22
 19          20        1500  connected     VirtualBox Host-Only Network
*/

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

func cmdQuit(root *CmdNode, c *TelnetClient, line string) {
	c.userOut <- fmt.Sprintf("bye\r\n")
	c.status = QUIT

	/*
		https://groups.google.com/d/msg/golang-nuts/JB_iiSQkmOk/dJNKSFQXUUQJ

		There is nothing wrong with having arbitrary numbers of senders, but if
		you do then it doesn't work to close the channel.  You need some other
		way to indicate EOF.

		Ian Lance Taylor
	*/
	//close(c.quit)

	c.quitInput <- 1
	c.quitOutput <- 1
}

func list(node *CmdNode, c *TelnetClient, depth int) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}
	ident := strings.Repeat(" ", 4*depth)
	c.userOut <- fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	for _, n := range node.Children {
		list(n, c, depth+1)
	}
}

func cmdList(root *CmdNode, c *TelnetClient, line string) {
	list(root, c, 0)
}

func cmdReload(root *CmdNode, c *TelnetClient, line string) {
}

func cmdShowInt(root *CmdNode, c *TelnetClient, line string) {
}

func cmdShowIPAddr(root *CmdNode, c *TelnetClient, line string) {
}

func cmdShowIPInt(root *CmdNode, c *TelnetClient, line string) {
}

func cmdShowIPRoute(root *CmdNode, c *TelnetClient, line string) {
}

func execute(root *CmdNode, c *TelnetClient, line string) {
	//log.Printf("execute: [%v]", line)
	//c.userOut <- fmt.Sprintf("echo: [%v]\r\n", line)

	if line == "" {
		return
	}

	/*
		if strings.HasPrefix("list", line) {
			cmdList(root, c, line)
			return
		}

		if strings.HasPrefix("quit", line) {
			cmdQuit(root, c, line)
			return
		}
	*/

	node, err := cmdFind(root, line, c.status)
	if err != nil {
		c.userOut <- fmt.Sprintf("command not found: %s\r\n", err)
		return
	}

	if node.Handler == nil {
		c.userOut <- fmt.Sprintf("command missing handler: [%s]\r\n", line)
		return
	}

	if node.MinLevel > c.status {
		c.userOut <- fmt.Sprintf("command level prohibited: [%s]\r\n", line)
		return
	}

	node.Handler(root, c, line)
}

func command(root *CmdNode, c *TelnetClient, line string) {
	//log.Printf("command: [%v]", line)
	//c.userOut <- fmt.Sprintf("echo: [%v]\r\n", line)

	switch c.status {
	case MOTD:
		// hello banner
		c.userOut <- fmt.Sprintf("\r\nrib server ready\r\n")
		c.status = USER
	case USER:
		c.echo <- false
		c.status = PASS
	case PASS:
		c.echo <- true
		c.status = EXEC
	case EXEC, ENAB, CONF:
		execute(root, c, line)
	default:
		log.Printf("unknown state for command: [%s]", line)
		c.userOut <- fmt.Sprintf("unknown state for command: [%s]\r\n", line)
	}

	sendPrompt(c.userOut, c.status)
}

func main() {
	log.Printf("runtime operating system: [%v]", runtime.GOOS)

	log.Printf("IP version: %v", ipv4.Version)

	cmdRoot := CmdNode{Path: "", MinLevel: EXEC, Handler: nil}

	cmdInstall(&cmdRoot, "list", EXEC, cmdList, "List command tree")
	cmdInstall(&cmdRoot, "quit", EXEC, cmdQuit, "Quit session")
	cmdInstall(&cmdRoot, "reload", ENAB, cmdReload, "Reload")
	cmdInstall(&cmdRoot, "reload", ENAB, cmdReload, "Ugh") // duplicated command
	cmdInstall(&cmdRoot, "show interface", EXEC, cmdShowInt, "Show interfaces")
	cmdInstall(&cmdRoot, "show", EXEC, cmdShowInt, "Ugh") // duplicated command
	cmdInstall(&cmdRoot, "show ip address", EXEC, cmdShowIPAddr, "Show addresses")
	cmdInstall(&cmdRoot, "show ip interface", EXEC, cmdShowIPInt, "Show interfaces")
	cmdInstall(&cmdRoot, "show ip interface detail", EXEC, cmdShowIPInt, "Show interface detail")
	cmdInstall(&cmdRoot, "show ip route", EXEC, cmdShowIPRoute, "Show routing table")

	go listenTelnet(":1234")

	localAddresses()

	routeAdd, routeDel := route.Routes()

LOOP:
	for {
		select {
		case r, ok := <-routeAdd:
			if !ok {
				log.Printf("Routes: quit")
				break LOOP
			}
			log.Printf("route add: %v", r)
		case r := <-routeDel:
			log.Printf("route del: %v", r)
		case cmd := <-cmdInput:
			command(&cmdRoot, cmd.client, cmd.line)
		}
	}
}
