package main

import (
	"flag"
	//"fmt"
	"log"
	//"os"
	//"path/filepath"
	"runtime"
	//"sort"
	//"strconv"
	//"strings"
	"time"

	"cli"
	"command"

	"golang.org/x/net/ipv4" // "code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net
)

type RibApp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode

	daemonName       string
	configPathPrefix string
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

/*
func (r RibApp) Hostname() string {
	root := r.ConfRootCandidate()
	log.Printf("rib RibApp.Hostname(): FIXME: query ACTIVE config")
	node, err := root.Get("hostname")
	if err != nil {
		return "hostname?"
	}

	return node.Value[0]
}
*/

func main() {
	log.Printf("rib starting")
	log.Printf("runtime operating system: [%v]", runtime.GOOS)
	log.Printf("CPUs: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	log.Printf("IP version: %v", ipv4.Version)

	ribConf := &RibApp{
		cmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},

		daemonName:       "rib",
		configPathPrefix: "",
	}

	installRibCommands(ribConf.CmdRoot())

	flag.StringVar(&ribConf.configPathPrefix, "configPathPrefix", "/tmp/devel/nexthop/etc/rib.conf.", "configuration path prefix")

	lastConfig, err := command.FindLastConfig(ribConf.configPathPrefix)
	if err != nil {
		log.Printf("main: error reading config: '%s': %v", ribConf.configPathPrefix, err)
	}

	log.Printf("last config file: %s", lastConfig)

	cliServer := cli.NewServer()

	go listenTelnet(":2001", cliServer)

	for {
		select {
		case <-time.After(time.Second * 5):
			log.Printf("rib main: tick")
		case comm := <-cliServer.CommandChannel:
			log.Printf("rib main: command: isLine=%v len=%d [%s]", comm.IsLine, len(comm.Cmd), comm.Cmd)
			cli.Execute(ribConf, comm.Cmd, comm.IsLine, comm.Client)
		case c := <-cliServer.InputClosed:
			// inputLoop hit closed connection. it's finished.
			// we should discard pending output (if any).
			log.Printf("rib main: inputLoop hit closed connection")
			c.DiscardOutputQueue()
		}
	}
}
