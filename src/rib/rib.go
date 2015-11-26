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
}

func main() {
	log.Printf("rib starting")
	log.Printf("runtime operating system: [%v]", runtime.GOOS)
	log.Printf("CPUs: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	log.Printf("IP version: %v", ipv4.Version)

	ribConf := &command.ConfContext{
		CmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		ConfRootCandidate: &command.ConfNode{},
		ConfRootActive:    &command.ConfNode{},
	}

	//cmdRoot := &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil}
	installRibCommands(ribConf.CmdRoot)

	cliServer := cli.NewServer()

	go listenTelnet(":2001", cliServer)

	for {
		select {
		case <-time.After(time.Second * 3):
			log.Printf("rib main: tick")
		case comm := <-cliServer.CommandChannel:
			log.Printf("rib main: command: isLine=%v len=%d [%s]", comm.IsLine, len(comm.Cmd), comm.Cmd)
			execute(ribConf, comm.Cmd, comm.IsLine, comm.Client)
		}
	}
}

func execute(ctx *command.ConfContext, line string, isLine bool, c *cli.Client) {
	log.Printf("rib main: execute: isLine=%v cmd=[%s]", isLine, line)

	if isLine {
		// full-line command
		executeLine(ctx, line, c)
		return
	}

	// single-char command
	log.Printf("rib main: execute: isLine=%v cmd=[%s] single-char command", isLine, line)
}

func executeLine(ctx *command.ConfContext, line string, c *cli.Client) {

	/*
		if line == "" {
			return
		}
	*/

	status := c.Status()

	node, err := command.CmdFind(ctx.CmdRoot, line, status)
	if err != nil {
		//c.userOut <- fmt.Sprintf("command not found: %s\r\n", err)
		//sendln(c, fmt.Sprintf("command not found: %s", err))
		msg := fmt.Sprintf("command not found: %s", err)
		log.Printf("executeLine: %v", msg)
		return
	}

	if node.Handler == nil {
		//sendln(c, fmt.Sprintf("command missing handler: [%s]", line))
		//c.userOut <- fmt.Sprintf("command missing handler: [%s]\r\n", line)
		msg := fmt.Sprintf("command missing handler: [%s]", line)
		log.Printf("executeLine: %v", msg)
		return
	}

	if node.MinLevel > status {
		//c.userOut <- fmt.Sprintf("command level prohibited: [%s]\r\n", line)
		//sendln(c, fmt.Sprintf("command level prohibited: [%s]", line))
		msg := fmt.Sprintf("command level prohibited: [%s]", line)
		log.Printf("executeLine: %v", msg)
		return
	}

	node.Handler(ctx, line, c)
}
