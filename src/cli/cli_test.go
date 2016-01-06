package cli

import (
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
func (a testApp) SetActive(newActive *command.ConfNode) {
	a.confRootActive = newActive
}
func (a testApp) ConfigPathPrefix() string {
	return "testApp.configPathPrefix"
}

type testClient struct {
	outputChannel chan string
}

func (c testClient) ConfigPath() string {
	return ""
}
func (c testClient) Status() int {
	return command.CONF
}
func (c testClient) StatusConf()               {}
func (c testClient) StatusEnable()             {}
func (c testClient) StatusExit()               {}
func (c testClient) ConfigPathSet(path string) {}
func (c testClient) Send(msg string)           {}
func (c testClient) SendNow(msg string)        {}
func (c testClient) Sendln(msg string)         {}
func (c testClient) SendlnNow(msg string)      {}
func (c testClient) InputQuit()                {}
func (c testClient) HistoryAdd(cmd string)     {}
func (c testClient) HistoryShow()              {}
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
	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	cmdBogus := func(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	}

	command.CmdInstall(root, cmdConf, "interface {IFNAME} description {ANY}", command.CONF, command.HelperDescription, command.ApplyBogus, "Set interface description")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IPADDR}", command.CONF, command.HelperIfaceAddr, command.ApplyBogus, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IPADDR6}", command.CONF, cmdBogus, command.ApplyBogus, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} shutdown", command.CONF, cmdBogus, command.ApplyBogus, "Disable interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdBogus, command.ApplyBogus, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, command.HelperHostname, command.ApplyBogus, "Assign hostname")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.CONF, command.HelperNo, nil, "Remove a configuration item")

	c := &testClient{outputChannel: make(chan string)}

	go func() {
		t.Logf("TestConf: starting output channel goroutine")
		for {
			line, ok := <-c.outputChannel
			if !ok {
				break
			}
			t.Logf("TestConf: read bogus client output channel: [%s]", line)
		}
		t.Logf("TestConf: exiting output channel goroutine")
	}()

	if err := dispatchCommand(app, "", c, command.CONF); err != nil {
		t.Errorf("empty command rejected: %v", err)
	}

	if err := dispatchCommand(app, "      ", c, command.CONF); err != nil {
		t.Errorf("blank command rejected: %v", err)
	}

	if err := dispatchCommand(app, "xxxxxx", c, command.CONF); err == nil {
		t.Errorf("bad command accepted")
	}

	dispatchCommand(app, "hostname nexthop-router", c, command.CONF)
	if host := hostname(app); host != "nexthop-router" {
		t.Errorf("bad hostname: %s", host)
	}

	dispatchCommand(app, "int eth0 desc  aa  bb   ccc", c, command.CONF)
	node, err := app.confRootCandidate.Get("interface eth0 description")
	if err != nil {
		t.Errorf("bad description: %v", err)
	}
	if node.Path != "interface eth0 description" {
		t.Errorf("bad description path: [%s]", node.Path)
	}
	if len(node.Value) != 1 {
		t.Errorf("bad description value count: %d", len(node.Value))
	}
	if node.Value[0] != " aa  bb   ccc" {
		t.Errorf("bad description value: [%s]", node.Value[0])
	}

	dispatchCommand(app, "no int eth0 desc xxxxxxx", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth0 description")
	if node != nil || err == nil {
		t.Errorf("eth0 description should not be present: node=[%v] error=[%v]", node, err)
	}

	dispatchCommand(app, "int eth1 desc ddd   eee   fff ", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth1 description")
	if err != nil {
		t.Errorf("bad description: %v", err)
	}
	if node.Path != "interface eth1 description" {
		t.Errorf("bad description path: [%s]", node.Path)
	}
	if len(node.Value) != 1 {
		t.Errorf("bad description value count: %d", len(node.Value))
	}
	if node.Value[0] != "ddd   eee   fff " {
		t.Errorf("bad description value: [%s]", node.Value[0])
	}

	dispatchCommand(app, "no int eth1 desc", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth1 description")
	if node != nil || err == nil {
		t.Errorf("eth1 description should not be present: node=[%v] error=[%v]", node, err)
	}

	dispatchCommand(app, "int eth2 ipv4 addr 1", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth2 ipv4 address")
	if err != nil {
		t.Errorf("bad eth2 address 1: %v", err)
		return
	}
	if len(node.Value) != 1 {
		t.Errorf("wrong number of eth2 addresses (expected=1): %d", len(node.Value))
	}
	dispatchCommand(app, "int eth2 ipv4 addr 2", c, command.CONF)
	if err != nil {
		t.Errorf("bad eth2 address 2: %v", err)
		return
	}
	if len(node.Value) != 2 {
		t.Errorf("wrong number of eth2 addresses (expected=2): %d", len(node.Value))
	}
	dispatchCommand(app, "int eth2 ipv4 addr 3", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth2 ipv4 address")
	if err != nil {
		t.Errorf("bad eth2 address 3: %v", err)
		return
	}
	if len(node.Value) != 3 {
		t.Errorf("wrong number of eth2 addresses (expected=3): %d", len(node.Value))
	}
	dispatchCommand(app, "no int eth2 ipv4 addr 3", c, command.CONF)
	if len(node.Value) != 2 {
		t.Errorf("wrong number of eth2 addresses (expected=2): %d", len(node.Value))
	}
	dispatchCommand(app, "no int eth2 ipv4 addr", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth2 ipv4 address")
	if node != nil || err == nil {
		t.Errorf("eth2 should not have address: node=[%v] error=[%v]", node, err)
	}

	dispatchCommand(app, "int eth3 ipv4 addr 1", c, command.CONF)
	dispatchCommand(app, "int eth3 ipv4 addr 2", c, command.CONF)
	dispatchCommand(app, "int eth3 ipv4 addr 3", c, command.CONF)
	dispatchCommand(app, "int eth3 ipv4 addr 4", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth3 ipv4 address")
	if err != nil {
		t.Errorf("bad eth3 address: %v", err)
		return
	}
	if len(node.Value) != 4 {
		t.Errorf("wrong number of eth3 addresses (expected=4): %d", len(node.Value))
	}
	node, err = app.confRootCandidate.Get("interface eth3 ipv4")
	if err != nil {
		t.Errorf("eth3 should have ipv4: node=[%v] error=[%v]", node, err)
	}
	dispatchCommand(app, "no int eth3 ipv4", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth3")
	if node != nil || err == nil {
		t.Errorf("eth3 should not exist: node=[%v] error=[%v]", node, err)
	}

	dispatchCommand(app, "int eth4 ipv4 addr 1", c, command.CONF)
	dispatchCommand(app, "int eth4 desc abc", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth4 ipv4 address")
	if err != nil {
		t.Errorf("bad eth4 address: %v", err)
		return
	}
	if len(node.Value) != 1 {
		t.Errorf("wrong number of eth4 addresses (expected=1): %d", len(node.Value))
	}
	node, err = app.confRootCandidate.Get("interface eth4 description")
	if err != nil {
		t.Errorf("bad eth4 description: %v", err)
		return
	}
	dispatchCommand(app, "no int eth4 ipv4 addr", c, command.CONF)
	node, err = app.confRootCandidate.Get("interface eth4 ipv4")
	if node != nil || err == nil {
		t.Errorf("eth4 should not have ipv4: node=[%v] error=[%v]", node, err)
		return
	}
	node, err = app.confRootCandidate.Get("interface eth4")
	if node == nil || err != nil {
		t.Errorf("eth4 should exist: node=[%v] error=[%v]", node, err)
		return
	}

	close(c.outputChannel)
}
