package main

import (
	"fmt"
	"testing"

	"command"
	"fwd"
)

type ripTestApp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode
}

func (a ripTestApp) CmdRoot() *command.CmdNode {
	return a.cmdRoot
}
func (a ripTestApp) ConfRootCandidate() *command.ConfNode {
	return a.confRootCandidate
}
func (a ripTestApp) ConfRootActive() *command.ConfNode {
	return a.confRootActive
}
func (a ripTestApp) SetActive(newActive *command.ConfNode) {
	a.confRootActive = newActive
}
func (a ripTestApp) SetCandidate(newCand *command.ConfNode) {
	a.confRootCandidate = newCand
}
func (a ripTestApp) ConfigPathPrefix() string {
	return "ripTestApp.configPathPrefix"
}
func (a ripTestApp) MaxConfigFiles() int {
	return 3
}

type ripTestClient struct {
	outputChannel chan string
}

func (c ripTestClient) ConfigPath() string {
	return ""
}
func (c ripTestClient) Status() int {
	return command.CONF
}
func (c ripTestClient) StatusConf()                                         {}
func (c ripTestClient) StatusEnable()                                       {}
func (c ripTestClient) StatusExit()                                         {}
func (c ripTestClient) ConfigPathSet(path string)                           {}
func (c ripTestClient) Newline()                                            {}
func (c ripTestClient) Send(msg string)                                     {}
func (c ripTestClient) SendNow(msg string)                                  {}
func (c ripTestClient) Sendln(msg string)                                   {}
func (c ripTestClient) SendlnNow(msg string)                                {}
func (c ripTestClient) InputQuit()                                          {}
func (c ripTestClient) HistoryAdd(cmd string)                               {}
func (c ripTestClient) HistoryShow()                                        {}
func (c ripTestClient) LineBufferComplete(autoComplete string, attach bool) {}
func (c ripTestClient) Output() chan<- string {
	return c.outputChannel
}

func TestConf(t *testing.T) {

	app := &ripTestApp{
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
		return []string{"BOGUS:rip.TestConf:listCommitId"}
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	root := app.cmdRoot
	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdConf, "router rip", command.CONF, cmdRip, applyRip, "Enable RIP protocol")
	command.CmdInstall(root, cmdConf, "router rip network {NETWORK}", command.CONF, cmdRipNetwork, applyRipNet, "Insert network into RIP protocol")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.CONF, command.HelperNo, nil, "Remove a configuration item")

	c := &ripTestClient{outputChannel: make(chan string)}
	close(c.outputChannel) // closed channel will break writers

	r := "router rip"
	net := fmt.Sprintf("%s network", r)
	n1 := "1.1.1.0/24"
	n2 := "2.2.2.0/24"
	net1 := fmt.Sprintf("%s %s", net, n1)
	net2 := fmt.Sprintf("%s %s", net, n2)
	command.Dispatch(app, net1, c, command.CONF, false)
	command.Dispatch(app, net2, c, command.CONF, false)

	{
		node, err := app.confRootCandidate.Get(net)
		if node == nil || err != nil {
			t.Errorf("missing config node=[%s] error: %v", net, err)
			return
		}
		if node.Path != net {
			t.Errorf("config node mismatch want=[%s] got=[%s]", net, node.Path)
		}
		if len(node.Value) != 2 {
			t.Errorf("bad number of values want=2 got=%d", len(node.Value))
		}
		if node.Value[0] != n1 {
			t.Errorf("unexpected 1st network want=%s got=%s", n1, node.Value[0])
		}
		if node.Value[1] != n2 {
			t.Errorf("unexpected 2nd network want=%s got=%s", n2, node.Value[1])
		}
	}

	nonet1 := fmt.Sprintf("no %s", net1)
	nonet2 := fmt.Sprintf("no %s", net2)

	noCmd, err := command.CmdFind(root, "no X", command.CONF, true)
	if err != nil {
		t.Errorf("could not find 'no' command: %v", err)
		return
	}

	if err := command.CmdNo(app, noCmd, nonet1, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", nonet1, err)
		return
	}
	if err := command.CmdNo(app, noCmd, nonet2, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", nonet2, err)
		return
	}

	{
		node, err := app.confRootCandidate.Get(net)
		if node != nil || err == nil {
			t.Errorf("unexpected config node=[%s] error: %v", net, err)
		}
	}

	{
		node, err := app.confRootCandidate.Get(r)
		if node == nil || err != nil {
			t.Errorf("missing config node=[%s] error: %v", r, err)
		}
	}

}
