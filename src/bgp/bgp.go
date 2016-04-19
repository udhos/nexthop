package main

import (
	"flag"
	"fmt"
	"fwd"
	"log"
	//"net"
	"strconv"
	//"strings"
	"time"

	"cli"
	"command"
)

type Bgp struct {
	cmdRoot           *command.CmdNode
	confRootCandidate *command.ConfNode
	confRootActive    *command.ConfNode

	daemonName       string
	configPathPrefix string
	maxConfigFiles   int

	hardware fwd.Dataplane

	router *BgpRouter
}

func (r Bgp) CmdRoot() *command.CmdNode {
	return r.cmdRoot
}
func (r Bgp) ConfRootCandidate() *command.ConfNode {
	return r.confRootCandidate
}
func (r Bgp) ConfRootActive() *command.ConfNode {
	return r.confRootActive
}
func (r *Bgp) SetActive(newActive *command.ConfNode) {
	r.confRootActive = newActive
}
func (r *Bgp) SetCandidate(newCand *command.ConfNode) {
	r.confRootCandidate = newCand
}
func (r Bgp) ConfigPathPrefix() string {
	return r.configPathPrefix
}
func (r Bgp) MaxConfigFiles() int {
	return r.maxConfigFiles
}

func main() {

	daemonName := "bgp"

	log.Printf("%s daemon starting", daemonName)

	bgp := &Bgp{
		cmdRoot:           &command.CmdNode{Path: "", MinLevel: command.EXEC, Handler: nil},
		confRootCandidate: &command.ConfNode{},
		confRootActive:    &command.ConfNode{},
		daemonName:        daemonName,
		hardware:          fwd.NewDataplaneBogus(),
	}

	var dataplaneName string
	configPrefix := command.ConfigPathRoot + "/" + daemonName + ".conf."
	flag.StringVar(&bgp.configPathPrefix, "configPathPrefix", configPrefix, "configuration path prefix")
	flag.IntVar(&bgp.maxConfigFiles, "maxConfigFiles", command.DefaultMaxConfigFiles, "limit number of configuration files (negative value means unlimited)")
	flag.StringVar(&dataplaneName, "dataplane", "native", "select forwarding engine")
	flag.Parse()

	bgp.hardware = fwd.NewDataplane(dataplaneName)

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := bgp.hardware.Interfaces()
		if err != nil {
			log.Printf("%s main: Interfaces(): error: %v", bgp.daemonName, err)
		}
		return ifaces, vrfs
	}
	listCommitId := func() []string {
		_, matches, err := command.ListConfig(bgp.ConfigPathPrefix(), true)
		if err != nil {
			log.Printf("%s main: error listing commit id's: %v", bgp.daemonName, err)
		}
		var idList []string
		for _, m := range matches {
			id, err1 := command.ExtractCommitIdFromFilename(m)
			if err1 != nil {
				log.Printf("%s main: error extracting commit id's: %v", bgp.daemonName, err1)
				continue
			}
			idList = append(idList, strconv.Itoa(id))
		}
		return idList
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	installCommands(bgp.CmdRoot())

	loadConf(bgp)

	cliServer := cli.NewServer()

	go cli.ListenTelnet(":2003", cliServer)

	tick := time.Duration(10)
	ticker := time.NewTicker(time.Second * tick)

	for {
		select {
		case <-ticker.C:
			log.Printf("%s main: %ds tick", bgp.daemonName, tick)
		case comm := <-cliServer.CommandChannel:
			log.Printf("%s main: command: isLine=%v len=%d [%s]", bgp.daemonName, comm.IsLine, len(comm.Cmd), comm.Cmd)
			cli.Execute(bgp, comm.Cmd, comm.IsLine, !comm.HideFromHistory, comm.Client)
		case c := <-cliServer.InputClosed:
			// inputLoop hit closed connection. it's finished.
			// we should discard pending output (if any).
			log.Printf("%s main: inputLoop hit closed connection", bgp.daemonName)
			c.DiscardOutputQueue()
		}
	}
}

func loadConf(bgp *Bgp) {
	lastConfig, err := command.FindLastConfig(bgp.configPathPrefix)
	if err != nil {
		log.Printf("%s main: error reading config: [%s]: %v", bgp.daemonName, bgp.configPathPrefix, err)
	}

	log.Printf("last config file: %s", lastConfig)

	bogusClient := command.NewBogusClient()

	abortOnError := false

	if _, err := command.LoadConfig(bgp, lastConfig, bogusClient, abortOnError); err != nil {
		log.Printf("%s main: error loading config: [%s]: %v", bgp.daemonName, lastConfig, err)
	}

	if err := command.Commit(bgp, bogusClient, false); err != nil {
		log.Printf("%s main: config commit failed: [%s]: %v", bgp.daemonName, lastConfig, err)
	}

	command.ConfActiveFromCandidate(bgp)
}

func installCommands(root *command.CmdNode) {

	command.InstallCommonHelpers(root)

	cmdNone := command.CMD_NONE
	cmdConH := command.CMD_CONF

	command.CmdInstall(root, cmdConH, "hostname {HOSTNAME}", command.CONF, command.HelperHostname, command.ApplyBogus, "Hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdConH, "router bgp {ASN}", command.CONF, cmdBgp, applyBgp, "Enable BGP protocol")
	command.CmdInstall(root, cmdConH, "router bgp {ASN} neighbor {IPADDR} remote-as {ASN}", command.CONF, cmdNeighAsn, applyNeighAsn, "Neighbor ASN")

	// Node description is used for pretty display in command help.
	// It is not strictly required, but its lack is reported by the command command.MissingDescription().
	command.DescInstall(root, "hostname", "Assign hostname")
	command.DescInstall(root, "router", "Configure routing")

	command.MissingDescription(root)
}

func bgpCtx(ctx command.ConfContext, c command.CmdClient) *Bgp {
	if bgp, ok := ctx.(*Bgp); ok {
		return bgp
	}
	// non-bgp context is a bogus state used for unit testing
	err := fmt.Errorf("bgpCtx: not a true Bgp context: %v", ctx)
	log.Printf("%v", err)
	c.Sendln(fmt.Sprintf("%v", err))
	return nil
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	bgp := ctx.(*Bgp)
	command.HelperShowVersion(bgp.daemonName, c)
}

func cmdBgp(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}

func applyBgp(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	bgp := bgpCtx(ctx, c)
	if bgp == nil {
		return nil
	}

	//enableBgp(bgp, action.Enable)

	return nil
}

func cmdNeighAsn(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}

func applyNeighAsn(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	bgp := bgpCtx(ctx, c)
	if bgp == nil {
		return nil
	}

	return nil
}
