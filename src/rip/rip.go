package main

import (
	"flag"
	"fmt"
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

func main() {

	rip := &Rip{
		cmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
		daemonName:        "rip",
	}

	log.Printf("%s daemon starting", rip.daemonName)

	installCommands(rip.CmdRoot())

	flag.StringVar(&rip.configPathPrefix, "configPathPrefix", "/tmp/devel/nexthop/etc/rip.conf.", "configuration path prefix")

	lastConfig, err := command.FindLastConfig(rip.configPathPrefix)
	if err != nil {
		log.Printf("main: error reading config: '%s': %v", rip.configPathPrefix, err)
	}

	log.Printf("last config file: %s", lastConfig)

	cliServer := cli.NewServer()

	go cli.ListenTelnet(":2002", cliServer)

	for {
		select {
		case <-time.After(time.Second * 5):
			log.Printf("%s main: tick", rip.daemonName)
		case comm := <-cliServer.CommandChannel:
			log.Printf("%s main: command: isLine=%v len=%d [%s]", rip.daemonName, comm.IsLine, len(comm.Cmd), comm.Cmd)
			cli.Execute(rip, comm.Cmd, comm.IsLine, comm.Client)
		case c := <-cliServer.InputClosed:
			// inputLoop hit closed connection. it's finished.
			// we should discard pending output (if any).
			log.Printf("%s main: inputLoop hit closed connection", rip.daemonName)
			c.DiscardOutputQueue()
		}
	}
}

func installCommands(root *command.CmdNode) {

	command.InstallCommonHelpers(root)

	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, cmdHostname, "Assign hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, "Show version")
}

func cmdHostname(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperHostname(ctx, node, line, c)
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	c.Sendln(command.NexthopVersion)
	rip := ctx.(*Rip)
	c.Sendln(fmt.Sprintf("daemon: %v", rip.daemonName))
}
