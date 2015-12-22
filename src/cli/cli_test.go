package cli

import (
	"fmt"
	"testing"

	"command"
)

type testApp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode
}

func (a testApp) CmdRoot() *command.CmdNode {
	return a.cmdRoot
}
func (a testApp) ConfRootCandidate() *command.ConfNode {
	return a.confRootCandidate
}
func (a testApp) ConfRootActive() *command.ConfNode {
	return a.confRootActive
}

type testClient struct {
	outputChannel chan string
}

func (c testClient) ConfigPath() string {
	return ""
}
func (c testClient) ConfigPathSet(path string) {}
func (c testClient) Send(msg string)           {}
func (c testClient) Sendln(msg string)         {}
func (c testClient) SendlnNow(msg string)      {}
func (c testClient) InputQuit()                {}
func (c testClient) Output() chan<- string {
	return c.outputChannel
}

func TestConf(t *testing.T) {

	app := &testApp{
		cmdRoot:           &command.CmdNode{MinLevel: command.EXEC},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
	}

	root := app.cmdRoot
	//cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	cmdBogus := func(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	}

	command.CmdInstall(root, cmdConf, "interface {IFNAME} description {ANY}", command.CONF, cmdBogus, "Set interface description")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IPADDR}", command.CONF, cmdBogus, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IPADDR6}", command.CONF, cmdBogus, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} shutdown", command.CONF, cmdBogus, "Disable interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdBogus, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, command.HelperHostname, "Assign hostname")

	c := &testClient{outputChannel: make(chan string)}

	go func() {
		for {
			line, ok := <-c.outputChannel
			if !ok {
				break
			}
			fmt.Printf("TestConf: read bogus client output channel: [%s]", line)
		}
		fmt.Printf("TestConf: exiting")
	}()

	dispatchCommand(app, "hostname nexthop-router", c, command.CONF)

	if host := hostname(app); host != "nexthop-router" {
		t.Errorf("bad hostname: %s", host)
	}

	close(c.outputChannel)
}
