package cli

import (
	"fmt"
	"testing"

	"github.com/udhos/nexthop/command"
	"github.com/udhos/nexthop/fwd"
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
func (a testApp) SetCandidate(newCand *command.ConfNode) {
	a.confRootCandidate = newCand
}
func (a testApp) ConfigPathPrefix() string {
	return "testApp.configPathPrefix"
}
func (a testApp) MaxConfigFiles() int {
	return 3
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
func (c testClient) StatusConf()                                         {}
func (c testClient) StatusEnable()                                       {}
func (c testClient) StatusExit()                                         {}
func (c testClient) ConfigPathSet(path string)                           {}
func (c testClient) Newline()                                            {}
func (c testClient) Send(msg string) int                                 { return len(msg) }
func (c testClient) SendNow(msg string)                                  {}
func (c testClient) Sendln(msg string) int                               { return len(msg) }
func (c testClient) SendlnNow(msg string)                                {}
func (c testClient) InputQuit()                                          {}
func (c testClient) HistoryAdd(cmd string)                               {}
func (c testClient) HistoryShow()                                        {}
func (c testClient) LineBufferComplete(autoComplete string, attach bool) {}
func (c testClient) Output() chan<- string {
	return c.outputChannel
}

func TestConf(t *testing.T) {

	app := &testApp{
		cmdRoot:           &command.CmdNode{MinLevel: command.EXEC},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
	}

	hardware := fwd.NewDataplaneBogus()

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := hardware.Interfaces()
		if err != nil {
			t.Errorf("hardware.Interfaces(): error: %v", err)
		}
		return ifaces, vrfs
	}
	listCommitId := func() []string {
		return []string{"BOGUS:TestConf:listCommitId"}
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	root := app.cmdRoot
	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	cmdBogus := func(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	}

	command.CmdInstall(root, cmdConf, "interface {IFNAME} description {ANY}", command.CONF, command.HelperIfaceDescr, command.ApplyBogus, "Set interface description")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IFADDR}", command.CONF, command.HelperIfaceAddr, command.ApplyBogus, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IFADDR6}", command.CONF, cmdBogus, command.ApplyBogus, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} shutdown", command.CONF, cmdBogus, command.ApplyBogus, "Disable interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdBogus, command.ApplyBogus, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname (HOSTNAME)", command.CONF, command.HelperHostname, command.ApplyBogus, "Assign hostname")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.CONF, command.HelperNo, nil, "Remove a configuration item")

	c := &testClient{outputChannel: make(chan string)}
	close(c.outputChannel) // closed channel will break writers

	if err := command.Dispatch(app, "", c, command.CONF, false); err != nil {
		t.Errorf("empty command rejected: %v", err)
	}

	if err := command.Dispatch(app, "      ", c, command.CONF, false); err != nil {
		t.Errorf("blank command rejected: %v", err)
	}

	if err := command.Dispatch(app, "  !xxx    ", c, command.CONF, false); err != nil {
		t.Errorf("comment command ! rejected: %v", err)
	}

	if err := command.Dispatch(app, "  #xxx    ", c, command.CONF, false); err != nil {
		t.Errorf("comment command # rejected: %v", err)
	}

	if err := command.Dispatch(app, "xxxxxx", c, command.CONF, false); err == nil {
		t.Errorf("bad command accepted")
	}

	command.Dispatch(app, "hostname nexthop-router", c, command.CONF, false)
	if host := getHostname(app.ConfRootCandidate()); host != "nexthop-router" {
		t.Errorf("bad hostname: %s", host)
	}

	command.Dispatch(app, "int eth0 desc  aa  bb   ccc", c, command.CONF, false)
	node, err := app.confRootCandidate.Get("interface eth0 description")
	if err != nil || node == nil {
		t.Errorf("bad description: %v", err)
		return
	}
	if node.Path != "interface eth0 description" {
		t.Errorf("bad description path: [%s]", node.Path)
	}
	if len(node.Children) != 1 {
		t.Errorf("bad description value count: %d", len(node.Children))
	}
	if v := command.DescriptionDecode(command.LastToken(node.Children[0].Path)); v != " aa  bb   ccc" {
		t.Errorf("bad description value: [%s]", v)
	}

	command.Dispatch(app, "no int eth0 desc xxxxxxx", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth0 description")
	if node != nil || err == nil {
		t.Errorf("eth0 description should not be present: node=[%v] error=[%v]", node, err)
	}

	command.Dispatch(app, "int eth1 desc ddd   eee   fff ", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth1 description")
	if err != nil {
		t.Errorf("bad description: %v", err)
	}
	if node.Path != "interface eth1 description" {
		t.Errorf("bad description path: [%s]", node.Path)
	}
	if len(node.Children) != 1 {
		t.Errorf("bad description value count: %d", len(node.Children))
	}
	if v := command.DescriptionDecode(command.LastToken(node.Children[0].Path)); v != "ddd   eee   fff " {
		t.Errorf("bad description value: [%s]", v)
	}

	command.Dispatch(app, "no int eth1 desc", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth1 description")
	if node != nil || err == nil {
		t.Errorf("eth1 description should not be present: node=[%v] error=[%v]", node, err)
	}

	command.Dispatch(app, "int eth2 ipv4 addr 1.1.1.1/1", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth2 ipv4 address")
	if err != nil {
		t.Errorf("bad eth2 address 1: %v", err)
		return
	}
	if len(node.Children) != 1 {
		t.Errorf("wrong number of eth2 addresses (expected=1): %d", len(node.Children))
	}
	command.Dispatch(app, "int eth2 ipv4 addr 2.2.2.2/2", c, command.CONF, false)
	if err != nil {
		t.Errorf("bad eth2 address 2: %v", err)
		return
	}
	if len(node.Children) != 2 {
		t.Errorf("wrong number of eth2 addresses (expected=2): %d", len(node.Children))
	}
	command.Dispatch(app, "int eth2 ipv4 addr 3.3.3.3/3", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth2 ipv4 address")
	if err != nil {
		t.Errorf("bad eth2 address 3: %v", err)
		return
	}
	if len(node.Children) != 3 {
		t.Errorf("wrong number of eth2 addresses (expected=3): %d", len(node.Children))
	}
	command.Dispatch(app, "no int eth2 ipv4 addr 3.3.3.3/3", c, command.CONF, false)
	if len(node.Children) != 2 {
		t.Errorf("wrong number of eth2 addresses (expected=2): %d", len(node.Children))
	}
	command.Dispatch(app, "no int eth2 ipv4 addr", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth2 ipv4 address")
	if node != nil || err == nil {
		t.Errorf("eth2 should not have address: node=[%v] error=[%v]", node, err)
	}

	command.Dispatch(app, "int eth3 ipv4 addr 1.1.1.1/1", c, command.CONF, false)
	command.Dispatch(app, "int eth3 ipv4 addr 2.2.2.2/2", c, command.CONF, false)
	command.Dispatch(app, "int eth3 ipv4 addr 3.3.3.3/3", c, command.CONF, false)
	command.Dispatch(app, "int eth3 ipv4 addr 4.4.4.4/4", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth3 ipv4 address")
	if err != nil {
		t.Errorf("bad eth3 address: %v", err)
		return
	}
	if len(node.Children) != 4 {
		t.Errorf("wrong number of eth3 addresses (expected=4): %d", len(node.Children))
	}
	node, err = app.confRootCandidate.Get("interface eth3 ipv4")
	if err != nil {
		t.Errorf("eth3 should have ipv4: node=[%v] error=[%v]", node, err)
	}
	command.Dispatch(app, "no int eth3 ipv4", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth3")
	if node != nil || err == nil {
		t.Errorf("eth3 should not exist: node=[%v] error=[%v]", node, err)
	}

	command.Dispatch(app, "int eth4 ipv4 addr 1.1.1.1/1", c, command.CONF, false)
	command.Dispatch(app, "int eth4 desc abc", c, command.CONF, false)
	node, err = app.confRootCandidate.Get("interface eth4 ipv4 address")
	if err != nil {
		t.Errorf("bad eth4 address: %v", err)
		return
	}
	if len(node.Children) != 1 {
		t.Errorf("wrong number of eth4 addresses (expected=1): %d", len(node.Children))
	}
	node, err = app.confRootCandidate.Get("interface eth4 description")
	if err != nil {
		t.Errorf("bad eth4 description: %v", err)
		return
	}
	command.Dispatch(app, "no int eth4 ipv4 addr", c, command.CONF, false)
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

	f := func() {
		command.Dispatch(app, "interface eth5 ipv4 address 1.1.1.1/24", c, command.CONF, false)
	}
	noCmd, err := command.CmdFind(root, "no X", command.CONF, true)
	if err != nil {
		t.Errorf("could not find 'no' command: %v", err)
	}
	f()
	cmd := "no interface eth5 ipv4 address 1.1.1.1/24"
	if err := command.CmdNo(app, noCmd, cmd, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", cmd, err)
	}
	f()
	cmd = "no interface eth5 ipv4 address"
	if err := command.CmdNo(app, noCmd, cmd, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", cmd, err)
	}
	f()
	cmd = "no interface eth5 ipv4"
	if err := command.CmdNo(app, noCmd, cmd, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", cmd, err)
	}
	f()
	cmd = "no interface eth5"
	if err := command.CmdNo(app, noCmd, cmd, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", cmd, err)
	}
	f()
	cmd = "no interface"
	if err := command.CmdNo(app, noCmd, cmd, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", cmd, err)
	}
	f()
	cmd = "no"
	if err := command.CmdNo(app, noCmd, cmd, c); err == nil {
		t.Errorf("bad cmd silently accepted: [%s]", cmd)
	}

}

func TestPrune(t *testing.T) {

	app := &testApp{
		cmdRoot:           &command.CmdNode{MinLevel: command.EXEC},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
	}

	hardware := fwd.NewDataplaneBogus()

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := hardware.Interfaces()
		if err != nil {
			t.Errorf("hardware.Interfaces(): error: %v", err)
		}
		return ifaces, vrfs
	}
	listCommitId := func() []string {
		return []string{"BOGUS:TestPrune:listCommitId"}
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	root := app.cmdRoot
	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	/*
		cmdBogus := func(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
		}
	*/

	cmdSimpleSet := func(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
		ctx.ConfRootCandidate().Set(node.Path, line)
	}

	path := "a b c d e f g h i j"
	jPath := "b c d e f g h i j"
	hPath := "b c d e f g h"
	gPath := "b c d e f g"
	fPath := "b c d e f"

	command.CmdInstall(root, cmdConf, path, command.CONF, cmdSimpleSet, command.ApplyBogus, "Teste prune A")
	command.CmdInstall(root, cmdConf, fPath, command.CONF, cmdSimpleSet, command.ApplyBogus, "Teste prune F")
	command.CmdInstall(root, cmdConf, jPath, command.CONF, cmdSimpleSet, command.ApplyBogus, "Teste prune J")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.CONF, command.HelperNo, nil, "Remove a configuration item")

	c := &testClient{outputChannel: make(chan string)}
	close(c.outputChannel) // closed channel will break writers

	noCmd, err := command.CmdFind(root, "no X", command.CONF, true)
	if noCmd == nil || err != nil {
		t.Errorf("could not find 'no' command: %v", err)
		return
	}

	command.Dispatch(app, path, c, command.CONF, false)

	{
		node, err := app.confRootCandidate.Get(path)
		if node == nil || err != nil {
			t.Errorf("missing config node=[%s] error: %v", path, err)
			return
		}
		if node.Path != path {
			t.Errorf("config node mismatch want=[%s] got=[%s]", path, node.Path)
			return
		}
	}

	noPath := fmt.Sprintf("no %s", path)
	if err := command.CmdNo(app, noCmd, noPath, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", noPath, err)
		return
	}

	{
		node, err := app.confRootCandidate.Get(path)
		if node != nil || err == nil {
			t.Errorf("unexpected config node=[%s] error: %v", path, err)
			return
		}
	}

	{
		p := "a"
		node, err := app.confRootCandidate.Get(p)
		if node != nil || err == nil {
			t.Errorf("unexpected config node=[%s] error: %v", p, err)
			return
		}
	}

	command.Dispatch(app, jPath, c, command.CONF, false)

	{
		node, err := app.confRootCandidate.Get(jPath)
		if node == nil || err != nil {
			t.Errorf("missing config node=[%s] error: %v", jPath, err)
			return
		}
		if node.Path != jPath {
			t.Errorf("config node mismatch want=[%s] got=[%s]", jPath, node.Path)
			return
		}
	}

	noH := fmt.Sprintf("no %s", hPath)
	if err := command.CmdNo(app, noCmd, noH, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", noH, err)
		return
	}

	{
		node, err := app.confRootCandidate.Get(hPath)
		if node != nil || err == nil {
			t.Errorf("unexpected config node=[%s] error: %v", hPath, err)
			return
		}
	}

	{
		node, err := app.confRootCandidate.Get(gPath)
		if node != nil || err == nil {
			t.Errorf("unexpected config node=[%s] error: %v", gPath, err)
			return
		}
	}

	{
		node, err := app.confRootCandidate.Get(fPath)
		if node == nil || err != nil {
			t.Errorf("missing config node=[%s] error: %v", fPath, err)
			return
		}
		if node.Path != fPath {
			t.Errorf("config node mismatch want=[%s] got=[%s]", fPath, node.Path)
			return
		}
	}

}
