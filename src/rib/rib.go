package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"cli"
	"command"

	"golang.org/x/net/ipv4" // "code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net
)

type RibApp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode

	daemonName string
}

func (r RibApp) CmdRoot() *command.CmdNode {
	return r.cmdRoot
}

func (r RibApp) ConfRootCandidate() *command.ConfNode {
	return r.confRootCandidate
}

func (r RibApp) ConfRootActive() *command.ConfNode {
	return r.confRootActive
}

func main() {
	log.Printf("rib starting")
	log.Printf("runtime operating system: [%v]", runtime.GOOS)
	log.Printf("CPUs: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	log.Printf("IP version: %v", ipv4.Version)

	ribConf := &RibApp{
		cmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},

		daemonName: "rib",
	}

	installRibCommands(ribConf.CmdRoot())

	cliServer := cli.NewServer()

	go listenTelnet(":2001", cliServer)

	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("rib main: tick")
		case comm := <-cliServer.CommandChannel:
			log.Printf("rib main: command: isLine=%v len=%d [%s]", comm.IsLine, len(comm.Cmd), comm.Cmd)
			execute(ribConf, comm.Cmd, comm.IsLine, comm.Client)
		case outputQuitChannel := <-cliServer.InputClosed:
			// inputLoop hit closed connection. it's finished.
			// we should discard pending output (if any) and request
			// termination of output loop.
			log.Printf("rib main: inputLoop hit closed connection, requesting outputLoop to quit")
			outputQuitChannel <- 1 // request outputLoop to quit
		}
	}
}

func execute(ctx command.ConfContext, line string, isLine bool, c *cli.Client) {
	log.Printf("rib main: execute: isLine=%v cmd=[%s]", isLine, line)

	if isLine {
		// full-line command
		executeLine(ctx, line, c)
		return
	}

	// single-char command
	log.Printf("rib main: execute: isLine=%v cmd=[%s] single-char command", isLine, line)
}

func executeLine(ctx command.ConfContext, line string, c *cli.Client) {
	log.Printf("executeLine: [%v]", line)

	status := c.Status()

	switch status {
	case command.MOTD:
		c.Sendln("")
		c.Sendln("rib server ready")
		c.StatusSet(command.USER)
	case command.USER:
		c.EchoDisable()
		c.StatusSet(command.PASS)
	case command.PASS:
		c.EchoEnable()
		c.StatusSet(command.EXEC)
	case command.EXEC, command.ENAB, command.CONF:
		dispatchCommand(ctx, line, c)
	default:
		msg := fmt.Sprintf("unknown state for command: [%s]", line)
		log.Print(msg)
		c.Sendln(msg)
	}

	paging := c.SendQueue()
	//c.SetSendEveryChar(!paging)
	c.SendPrompt(paging)
	c.Flush()

	log.Printf("executeLine: flushed [%v]", line)
}

func dispatchCommand(ctx command.ConfContext, line string, c *cli.Client) {

	/*
		if line == "" {
			return
		}
	*/

	prependConfigPath := true

	status := c.Status()

	n, e := command.CmdFind(ctx.CmdRoot(), line, status)
	if e == nil {
		// found at root
		if n.Options&command.CMD_CONF == 0 {
			// not a config cmd
			prependConfigPath = false
		}
	}

	lookupPath := line
	configPath := c.ConfigPath()
	if prependConfigPath && configPath != "" {
		lookupPath = fmt.Sprintf("%s %s", c.ConfigPath(), line)
	}

	//log.Printf("dispatchCommand: prepend=%v path=[%s] line=[%s] full=[%s]", prependConfigPath, configPath, line, lookupPath)

	node, err := command.CmdFind(ctx.CmdRoot(), lookupPath, status)
	if err != nil {
		log.Printf("dispatchCommand: command not found: %s", err)
		return
	}

	if node.Handler == nil {
		log.Printf("dispatchCommand: command missing handler: [%s]", lookupPath)
		if node.Options&command.CMD_CONF != 0 {
			c.ConfigPathSet(lookupPath)
		}
		return
	}

	if node.MinLevel > status {
		log.Printf("dispatchCommand: command level prohibited: [%s]", lookupPath)
		return
	}

	node.Handler(ctx, node, lookupPath, c)
}
