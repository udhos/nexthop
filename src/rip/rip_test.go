package main

import (
	"fmt"
	"log"
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
	return "BOGUS:ripTestApp.configPathPrefix"
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
func (c ripTestClient) InputQuit()                                          {}
func (c ripTestClient) HistoryAdd(cmd string)                               {}
func (c ripTestClient) HistoryShow()                                        {}
func (c ripTestClient) LineBufferComplete(autoComplete string, attach bool) {}

func (c ripTestClient) Newline()              { c.Send("\n") }
func (c ripTestClient) SendNow(msg string)    { c.Output() <- msg }
func (c ripTestClient) Sendln(msg string) int { return c.Send(fmt.Sprintf("%s\n", msg)) }
func (c ripTestClient) SendlnNow(msg string)  { c.SendNow(fmt.Sprintf("%s\n", msg)) }
func (c ripTestClient) Output() chan<- string { return c.outputChannel }
func (c ripTestClient) Send(msg string) int {
	c.SendNow(msg)
	return len(msg)
}

func NewRipTestClient(outputSinkHandler func(string)) *ripTestClient {
	c := &ripTestClient{outputChannel: make(chan string)}

	if outputSinkHandler != nil {
		go func() {
			log.Printf("OutputSink: starting")
			for {
				log.Printf("OutputSink: waiting")

				m, ok := <-c.outputChannel
				if !ok {
					log.Printf("OutputSink: closed channel")
					return
				}
				log.Printf("OutputSink: [%s]", m)
				outputSinkHandler(m)
			}
		}()
		return c
	}

	close(c.outputChannel) // closed channel will break writers
	return c
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

	c := NewRipTestClient(func(string) {})

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
		if len(node.Children) != 2 {
			t.Errorf("bad number of values want=2 got=%d", len(node.Children))
		}
		if v := command.LastToken(node.Children[0].Path); v != n1 {
			t.Errorf("unexpected 1st network want=%s got=%s", n1, v)
		}
		if v := command.LastToken(node.Children[1].Path); v != n2 {
			t.Errorf("unexpected 2nd network want=%s got=%s", n2, v)
		}
	}

	nonet1 := fmt.Sprintf("no %s", net1)
	nonet2 := fmt.Sprintf("no %s", net2)

	/*
		noCmd, err := command.CmdFind(root, "no X", command.CONF, true)
		if err != nil {
			t.Errorf("could not find 'no' command: %v", err)
			return
		}
	*/

	dumpConf(app.confRootCandidate, "conf1:")

	fmt.Printf("trying: [%s]\n", nonet1)
	if err := command.CmdNo(app, nil, nonet1, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", nonet1, err)
		return
	}

	dumpConf(app.confRootCandidate, "conf2:")
	fmt.Printf("trying: [%s]\n", nonet2)
	if err := command.CmdNo(app, nil, nonet2, c); err != nil {
		t.Errorf("cmd failed: [%s] error=[%v]", nonet2, err)
		return
	}

	dumpConf(app.confRootCandidate, "conf3:")

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

func dumpConf(root *command.ConfNode, label string) {
	fmt.Println(label)
	command.WriteConfig(root, &outputWriter{}, false)
}

type outputWriter struct {
}

func (w *outputWriter) WriteLine(s string) (int, error) {
	fmt.Println(s)
	return len(s), nil
}

func Example_diff1() {

	app, c := setup_diff()

	f := func(s string) {
		if err := command.Dispatch(app, s, c, command.CONF, false); err != nil {
			log.Printf("dispatch: [%s]: %v", s, err)
		}
	}

	f("hostname rip")
	f("router rip network 1.1.1.0/24")
	f("router rip network 1.1.2.0/24")
	f("router rip network 1.1.3.0/24 cost 2")
	f("router rip network 1.1.4.0/24 cost 15")
	f("router rip vrf X network 1.1.1.0/24")
	f("router rip vrf X network 1.1.2.0/24")
	f("router rip vrf X network 1.1.3.0/24 cost 3")
	f("router rip vrf x network 1.1.1.1/32")

	command.WriteConfig(app.confRootCandidate, &outputWriter{}, false)
	// Output:
	// hostname rip
	// router rip network 1.1.1.0/24
	// router rip network 1.1.2.0/24
	// router rip network 1.1.3.0/24 cost 2
	// router rip network 1.1.4.0/24 cost 15
	// router rip vrf X network 1.1.1.0/24
	// router rip vrf X network 1.1.2.0/24
	// router rip vrf X network 1.1.3.0/24 cost 3
	// router rip vrf x network 1.1.1.1/32
}

func Example_diff2() {

	app, c := setup_diff()

	f := func(s string) {
		if err := command.Dispatch(app, s, c, command.CONF, false); err != nil {
			log.Printf("dispatch: [%s]: %v", s, err)
		}
	}

	f("hostname rip")
	f("router rip network 1.1.1.0/24")
	f("router rip network 1.1.2.0/24")
	f("router rip network 1.1.3.0/24 cost 2")
	f("router rip network 1.1.4.0/24 cost 15")
	f("router rip vrf X network 1.1.1.0/24")
	f("router rip vrf X network 1.1.2.0/24")
	f("router rip vrf X network 1.1.3.0/24 cost 3")
	f("router rip vrf x network 1.1.1.1/32")

	if err := command.Dispatch(app, "commit", c, command.CONF, false); err != nil {
		log.Printf("dispatch: [commit]: %v", err)
	}

	/*
		noCmd, err := command.CmdFind(app.cmdRoot, "no X", command.CONF, true)
		if err != nil {
			log.Printf("could not find 'no' command: %v", err)
			return
		}
	*/

	nonet := "no router rip network"
	if err := command.CmdNo(app, nil, nonet, c); err != nil {
		log.Printf("cmd failed: [%s] error=[%v]", nonet, err)
		return
	}

	command.WriteConfig(app.confRootCandidate, &outputWriter{}, false)
	// Output:
	// hostname rip
	// router rip vrf X network 1.1.1.0/24
	// router rip vrf X network 1.1.2.0/24
	// router rip vrf X network 1.1.3.0/24 cost 3
	// router rip vrf x network 1.1.1.1/32
}

func setup_diff() (*ripTestApp, *ripTestClient) {
	app := &ripTestApp{
		cmdRoot:           &command.CmdNode{MinLevel: command.EXEC},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
	}

	hardware := fwd.NewDataplaneBogus()

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := hardware.Interfaces()
		if err != nil {
			log.Printf("Example_diff: hardware.Interfaces(): error: %v", err)
		}
		return ifaces, vrfs
	}
	listCommitId := func() []string {
		return []string{"BOGUS:rip.Example_diff:listCommitId"}
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	root := app.cmdRoot
	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdNone, "commit", command.CONF, command.HelperCommit, nil, "Apply current candidate configuration")
	command.CmdInstall(root, cmdNone, "show configuration", command.EXEC, command.HelperShowConf, nil, "Show candidate configuration")
	command.CmdInstall(root, cmdNone, "show configuration compare", command.EXEC, command.HelperShowCompare, nil, "Show differences between active and candidate configurations")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.CONF, command.HelperNo, nil, "Remove a configuration item")

	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, command.HelperHostname, command.ApplyBogus, "Hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdConf, "router rip", command.CONF, cmdRip, applyRip, "Enable RIP protocol")
	command.CmdInstall(root, cmdConf, "router rip network {NETWORK}", command.CONF, cmdRipNetwork, applyRipNet, "Insert network into RIP protocol")
	command.CmdInstall(root, cmdConf, "router rip network {NETWORK} cost {RIPMETRIC}", command.CONF, cmdRipNetCost, applyRipNetCost, "RIP network metric")
	command.CmdInstall(root, cmdConf, "router rip vrf {VRFNAME} network {NETWORK}", command.CONF, cmdRipNetwork, applyRipNet, "Insert network into RIP protocol")
	command.CmdInstall(root, cmdConf, "router rip vrf {VRFNAME} network {NETWORK} cost {RIPMETRIC}", command.CONF, cmdRipNetCost, applyRipNetCost, "RIP network metric")

	outputSinkFunc := func(m string) {
	}
	c := NewRipTestClient(outputSinkFunc)

	return app, c
}
