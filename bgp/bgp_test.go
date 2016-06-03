package main

import (
	"fmt"
	"log"
	//"testing"

	"github.com/udhos/nexthop/command"
	"github.com/udhos/nexthop/fwd"
)

type bgpTestApp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode
}

func (a bgpTestApp) CmdRoot() *command.CmdNode {
	return a.cmdRoot
}
func (a bgpTestApp) ConfRootCandidate() *command.ConfNode {
	return a.confRootCandidate
}
func (a bgpTestApp) ConfRootActive() *command.ConfNode {
	return a.confRootActive
}
func (a bgpTestApp) SetActive(newActive *command.ConfNode) {
	a.confRootActive = newActive
}
func (a bgpTestApp) SetCandidate(newCand *command.ConfNode) {
	a.confRootCandidate = newCand
}
func (a bgpTestApp) ConfigPathPrefix() string {
	return "BOGUS:bgpTestApp.configPathPrefix"
}
func (a bgpTestApp) MaxConfigFiles() int {
	return 3
}

type bgpTestClient struct {
	outputChannel chan string
}

func (c bgpTestClient) ConfigPath() string {
	return ""
}
func (c bgpTestClient) Status() int {
	return command.CONF
}
func (c bgpTestClient) StatusConf()                                         {}
func (c bgpTestClient) StatusEnable()                                       {}
func (c bgpTestClient) StatusExit()                                         {}
func (c bgpTestClient) ConfigPathSet(path string)                           {}
func (c bgpTestClient) InputQuit()                                          {}
func (c bgpTestClient) HistoryAdd(cmd string)                               {}
func (c bgpTestClient) HistoryShow()                                        {}
func (c bgpTestClient) LineBufferComplete(autoComplete string, attach bool) {}

func (c bgpTestClient) Newline()              { c.Send("\n") }
func (c bgpTestClient) SendNow(msg string)    { c.Output() <- msg }
func (c bgpTestClient) Sendln(msg string) int { return c.Send(fmt.Sprintf("%s\n", msg)) }
func (c bgpTestClient) SendlnNow(msg string)  { c.SendNow(fmt.Sprintf("%s\n", msg)) }
func (c bgpTestClient) Output() chan<- string { return c.outputChannel }
func (c bgpTestClient) Send(msg string) int {
	c.SendNow(msg)
	return len(msg)
}

func NewBgpTestClient(outputSinkHandler func(string)) *bgpTestClient {
	c := &bgpTestClient{outputChannel: make(chan string)}

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

	f("hostname bgp1")
	f("hostname bgp2")
	f("router bgp 1 neighbor 1.1.1.1 description  A  BB   C")
	f("router bgp 1 neighbor 1.1.1.1 description  AA  BB   C")
	f("router bgp 1 neighbor 1.1.1.1 remote-as 1")
	f("router bgp 1 neighbor 2.2.2.2 remote-as 1")
	f("router bgp 1 neighbor 3.3.3.3 remote-as 2")
	f("router bgp 2 neighbor 3.3.3.3 remote-as 2")
	f("router bgp 2 neighbor 4.4.4.4 remote-as 2")
	f("router bgp 2 neighbor 4.4.4.4 remote-as 3")

	command.WriteConfig(app.confRootCandidate, &outputWriter{})
	// Output:
	// hostname bgp2
	// router bgp 1 neighbor 1.1.1.1 description  AA  BB   C
	// router bgp 1 neighbor 1.1.1.1 remote-as 1
	// router bgp 1 neighbor 2.2.2.2 remote-as 1
	// router bgp 1 neighbor 3.3.3.3 remote-as 2
	// router bgp 2 neighbor 3.3.3.3 remote-as 2
	// router bgp 2 neighbor 4.4.4.4 remote-as 3
}

func Example_diff2() {

	app, c := setup_diff()

	f := func(s string) {
		if err := command.Dispatch(app, s, c, command.CONF, false); err != nil {
			log.Printf("dispatch: [%s]: %v", s, err)
		}
	}

	f("hostname bgp1")
	f("hostname bgp2")
	f("router bgp 1 neighbor 1.1.1.1 remote-as 1")
	f("router bgp 1 neighbor 2.2.2.2 remote-as 1")
	f("router bgp 1 neighbor 3.3.3.3 remote-as 2")
	f("router bgp 2 neighbor 3.3.3.3 remote-as 2")
	f("router bgp 2 neighbor 4.4.4.4 remote-as 2")
	f("router bgp 2 neighbor 4.4.4.4 remote-as 3")

	if err := command.Dispatch(app, "commit", c, command.CONF, false); err != nil {
		log.Printf("dispatch: [commit]: %v", err)
	}

	nonet := "no router bgp 1"
	if err := command.CmdNo(app, nil, nonet, c); err != nil {
		log.Printf("cmd failed: [%s] error=[%v]", nonet, err)
		return
	}

	command.WriteConfig(app.confRootCandidate, &outputWriter{})
	// Output:
	// hostname bgp2
	// router bgp 2 neighbor 3.3.3.3 remote-as 2
	// router bgp 2 neighbor 4.4.4.4 remote-as 3
}

func setup_diff() (*bgpTestApp, *bgpTestClient) {
	app := &bgpTestApp{
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
		return []string{"BOGUS:bgp.Example_diff:listCommitId"}
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	root := app.cmdRoot
	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdNone, "commit", command.CONF, command.HelperCommit, nil, "Apply current candidate configuration")
	command.CmdInstall(root, cmdNone, "show configuration", command.EXEC, command.HelperShowConf, nil, "Show candidate configuration")
	command.CmdInstall(root, cmdNone, "show configuration compare", command.EXEC, command.HelperShowCompare, nil, "Show differences between active and candidate configurations")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.CONF, command.HelperNo, nil, "Remove a configuration item")

	command.CmdInstall(root, cmdConf, "hostname (HOSTNAME)", command.CONF, command.HelperHostname, command.ApplyBogus, "Hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdConf, "router bgp {ASN} neighbor {IPADDR} description {ANY}", command.CONF, cmdNeighDesc, command.ApplyBogus, "BGP neighbor description")
	command.CmdInstall(root, cmdConf, "router bgp {ASN} neighbor {IPADDR} remote-as (ASN)", command.CONF, cmdNeighAsn, applyNeighAsn, "BGP neighbor ASN")

	outputSinkFunc := func(m string) {
	}
	c := NewBgpTestClient(outputSinkFunc)

	return app, c
}
