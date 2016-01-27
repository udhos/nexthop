package main

import (
	"flag"
	"log"
	//"os"
	//"path/filepath"
	"runtime"
	//"sort"
	"strconv"
	//"strings"
	"time"

	"cli"
	"command"
	"fwd"
	//"golang.org/x/net/ipv4" // "code.google.com/p/go.net/ipv4" // https://code.google.com/p/go/source/checkout?repo=net
)

type RibApp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode

	daemonName       string
	configPathPrefix string
	maxConfigFiles   int

	hardware fwd.Dataplane
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
func (r *RibApp) SetActive(newActive *command.ConfNode) {
	r.confRootActive = newActive
}
func (r *RibApp) SetCandidate(newCand *command.ConfNode) {
	r.confRootCandidate = newCand
}
func (r RibApp) ConfigPathPrefix() string {
	return r.configPathPrefix
}
func (r RibApp) MaxConfigFiles() int {
	return r.maxConfigFiles
}

func main() {

	daemonName := "rib"

	log.Printf("%s daemon starting", daemonName)
	log.Printf("runtime operating system: [%v]", runtime.GOOS)
	log.Printf("CPUs: NumCPU=%d GOMAXPROCS=%d", runtime.NumCPU(), runtime.GOMAXPROCS(0))
	//log.Printf("IP version: %v", ipv4.Version)

	ribConf := &RibApp{
		cmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
		daemonName:        daemonName,
		hardware:          fwd.NewDataplaneBogus(),
	}

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := ribConf.hardware.Interfaces()
		if err != nil {
			log.Printf("%s main: Interfaces(): error: %v", ribConf.daemonName, err)
		}
		return ifaces, vrfs
	}
	listCommitId := func() []string {
		_, matches, err := command.ListConfig(ribConf.ConfigPathPrefix())
		if err != nil {
			log.Printf("%s main: error listing commit id's: %v", ribConf.daemonName, err)
		}
		var idList []string
		for _, m := range matches {
			id, err1 := command.ExtractCommitIdFromFilename(m)
			if err1 != nil {
				log.Printf("%s main: error extracting commit id's: %v", ribConf.daemonName, err1)
				continue
			}
			idList = append(idList, strconv.Itoa(id))
		}
		return idList
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	installRibCommands(ribConf.CmdRoot())

	flag.StringVar(&ribConf.configPathPrefix, "configPathPrefix", command.ConfigPathRoot+"/rib.conf.", "configuration path prefix")
	flag.IntVar(&ribConf.maxConfigFiles, "maxConfigFiles", 10, "limit number of configuration files (negative value means unlimited)")
	flag.Parse()

	loadConf(ribConf)

	cliServer := cli.NewServer()

	go cli.ListenTelnet(":2001", cliServer)

	for {
		select {
		case <-time.After(time.Second * 5):
			log.Printf("rib main: tick")
		case comm := <-cliServer.CommandChannel:
			log.Printf("rib main: command: isLine=%v len=%d [%s]", comm.IsLine, len(comm.Cmd), comm.Cmd)
			cli.Execute(ribConf, comm.Cmd, comm.IsLine, !comm.HideFromHistory, comm.Client)
		case c := <-cliServer.InputClosed:
			// inputLoop hit closed connection. it's finished.
			// we should discard pending output (if any).
			log.Printf("rib main: inputLoop hit closed connection")
			c.DiscardOutputQueue()
		}
	}
}

func loadConf(rib *RibApp) {
	lastConfig, err := command.FindLastConfig(rib.configPathPrefix)
	if err != nil {
		log.Printf("main: error reading config: '%s': %v", rib.configPathPrefix, err)
	}

	log.Printf("last config file: %s", lastConfig)

	bogusClient := command.NewBogusClient()

	abortOnError := false

	if _, err := command.LoadConfig(rib, lastConfig, bogusClient, abortOnError); err != nil {
		log.Printf("%s main: error loading config: [%s]: %v", rib.daemonName, lastConfig, err)
	}

	if err := command.Commit(rib, bogusClient, false); err != nil {
		log.Printf("%s main: config commit failed: [%s]: %v", rib.daemonName, lastConfig, err)
	}

	command.ConfActiveFromCandidate(rib)
}
