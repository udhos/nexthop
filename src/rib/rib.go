package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
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

type sortByCommitId []string

func (s sortByCommitId) Len() int {
	return len(s)
}
func (s sortByCommitId) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortByCommitId) Less(i, j int) bool {
	s1 := s[i]
	lastDot1 := strings.LastIndexByte(s1, '.')
	commitId1 := s1[lastDot1+1:]
	id1, err1 := strconv.Atoi(commitId1)
	if err1 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s1, err1)
	}
	s2 := s[j]
	lastDot2 := strings.LastIndexByte(s2, '.')
	commitId2 := s2[lastDot2+1:]
	id2, err2 := strconv.Atoi(commitId2)
	if err2 != nil {
		log.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s2, err2)
	}
	return id1 < id2
}

func (r *RibApp) LoadLastConfig() (*command.ConfNode, error) {
	log.Printf("LoadLastConfig: configuration path prefix: %s", r.configPathPrefix)

	dirname := filepath.Dir(r.configPathPrefix)

	dir, err := os.Open(dirname)
	if err != nil {
		return nil, fmt.Errorf("LoadLastConfig: error opening dir '%s': %v", dirname, err)
	}

	names, e := dir.Readdirnames(0)
	if e != nil {
		return nil, fmt.Errorf("LoadLastConfig: error reading dir '%s': %v", dirname, e)
	}

	dir.Close()

	//log.Printf("LoadLastConfig: found %d files: %v", len(names), names)

	basename := filepath.Base(r.configPathPrefix)

	// filter prefix
	matches := names[:0]
	for _, x := range names {
		//log.Printf("LoadLastConfig: x=[%s] prefix=[%s]", x, basename)
		if strings.HasPrefix(x, basename) {
			matches = append(matches, x)
		}
	}

	sort.Sort(sortByCommitId(matches))

	m := len(matches)

	log.Printf("LoadLastConfig: found %d matching files: %v", m, matches)

	if m < 1 {
		return nil, fmt.Errorf("LoadLastConfig: no config file found for prefix: %s", r.configPathPrefix)
	}

	lastConfig := names[m-1]

	return nil, fmt.Errorf("LoadLastConfig: found=[%s] FIXME WRITEME", lastConfig)
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

		daemonName:       "rib",
		configPathPrefix: "",
	}

	installRibCommands(ribConf.CmdRoot())

	flag.StringVar(&ribConf.configPathPrefix, "configPathPrefix", "/tmp/devel/nexthop/etc/rib.conf.", "configuration path prefix")

	lastConfig, err := ribConf.LoadLastConfig()
	if err != nil {
		log.Printf("main: error reading config: '%s': %v", ribConf.configPathPrefix, err)
	}

	log.Printf("last config loaded: %v", lastConfig)

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
