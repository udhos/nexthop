package main

import (
	"flag"
	"fwd"
	"log"
	"time"

	"cli"
	"command"
)

type Rip struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode

	daemonName       string
	configPathPrefix string
	maxConfigFiles   int

	hardware fwd.Dataplane
}

func (r Rip) CmdRoot() *command.CmdNode {
	return r.cmdRoot
}
func (r Rip) ConfRootCandidate() *command.ConfNode {
	return r.confRootCandidate
}
func (r Rip) ConfRootActive() *command.ConfNode {
	return r.confRootActive
}
func (r *Rip) SetActive(newActive *command.ConfNode) {
	r.confRootActive = newActive
}
func (r *Rip) SetCandidate(newCand *command.ConfNode) {
	r.confRootCandidate = newCand
}
func (r Rip) ConfigPathPrefix() string {
	return r.configPathPrefix
}
func (r Rip) MaxConfigFiles() int {
	return r.maxConfigFiles
}

func main() {

	daemonName := "rip"

	log.Printf("%s daemon starting", daemonName)

	rip := &Rip{
		cmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
		daemonName:        daemonName,
		hardware:          fwd.NewDataplaneBogus(),
	}

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := rip.hardware.Interfaces()
		if err != nil {
			log.Printf("%s main: Interfaces(): error: %v", rip.daemonName, err)
		}
		return ifaces, vrfs
	}
	command.LoadKeywordTable(listInterfaces)

	installCommands(rip.CmdRoot())

	flag.StringVar(&rip.configPathPrefix, "configPathPrefix", "/tmp/devel/nexthop/etc/rip.conf.", "configuration path prefix")
	flag.IntVar(&rip.maxConfigFiles, "maxConfigFiles", 10, "limit number of configuration files (negative value means unlimited)")

	loadConf(rip)

	cliServer := cli.NewServer()

	go cli.ListenTelnet(":2002", cliServer)

	for {
		select {
		case <-time.After(time.Second * 5):
			log.Printf("%s main: tick", rip.daemonName)
		case comm := <-cliServer.CommandChannel:
			log.Printf("%s main: command: isLine=%v len=%d [%s]", rip.daemonName, comm.IsLine, len(comm.Cmd), comm.Cmd)
			cli.Execute(rip, comm.Cmd, comm.IsLine, !comm.HideFromHistory, comm.Client)
		case c := <-cliServer.InputClosed:
			// inputLoop hit closed connection. it's finished.
			// we should discard pending output (if any).
			log.Printf("%s main: inputLoop hit closed connection", rip.daemonName)
			c.DiscardOutputQueue()
		}
	}
}

func loadConf(rip *Rip) {
	lastConfig, err := command.FindLastConfig(rip.configPathPrefix)
	if err != nil {
		log.Printf("%s main: error reading config: [%s]: %v", rip.daemonName, rip.configPathPrefix, err)
	}

	log.Printf("last config file: %s", lastConfig)

	bogusClient := command.NewBogusClient()

	abortOnError := false

	if _, err := command.LoadConfig(rip, lastConfig, bogusClient, abortOnError); err != nil {
		log.Printf("%s main: error loading config: [%s]: %v", rip.daemonName, lastConfig, err)
	}

	if err := command.Commit(rip, bogusClient, false); err != nil {
		log.Printf("%s main: config commit failed: [%s]: %v", rip.daemonName, lastConfig, err)
	}

	command.ConfActiveFromCandidate(rip)
}

func installCommands(root *command.CmdNode) {

	command.InstallCommonHelpers(root)

	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, cmdHostname, command.ApplyBogus, "Hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")

	command.DescInstall(root, "hostname", "Assign hostname")

	command.MissingDescription(root)
}

func cmdHostname(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperHostname(ctx, node, line, c)
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	rip := ctx.(*Rip)
	command.HelperShowVersion(rip.daemonName, c)
}
