package main

import (
	"flag"
	"fmt"
	"fwd"
	"log"
	"strconv"
	"strings"
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

	router *RipRouter
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

	var dataplaneName string
	flag.StringVar(&rip.configPathPrefix, "configPathPrefix", command.ConfigPathRoot+"/rip.conf.", "configuration path prefix")
	flag.IntVar(&rip.maxConfigFiles, "maxConfigFiles", command.DefaultMaxConfigFiles, "limit number of configuration files (negative value means unlimited)")
	flag.StringVar(&dataplaneName, "dataplane", "native", "select forwarding engine")
	flag.Parse()

	rip.hardware = fwd.NewDataplane(dataplaneName)

	listInterfaces := func() ([]string, []string) {
		ifaces, vrfs, err := rip.hardware.Interfaces()
		if err != nil {
			log.Printf("%s main: Interfaces(): error: %v", rip.daemonName, err)
		}
		return ifaces, vrfs
	}
	listCommitId := func() []string {
		_, matches, err := command.ListConfig(rip.ConfigPathPrefix(), true)
		if err != nil {
			log.Printf("%s main: error listing commit id's: %v", rip.daemonName, err)
		}
		var idList []string
		for _, m := range matches {
			id, err1 := command.ExtractCommitIdFromFilename(m)
			if err1 != nil {
				log.Printf("%s main: error extracting commit id's: %v", rip.daemonName, err1)
				continue
			}
			idList = append(idList, strconv.Itoa(id))
		}
		return idList
	}
	command.LoadKeywordTable(listInterfaces, listCommitId)

	installCommands(rip.CmdRoot())

	loadConf(rip)

	cliServer := cli.NewServer()

	go cli.ListenTelnet(":2002", cliServer)

	tick := time.Duration(10)
	ticker := time.NewTicker(time.Second * tick)

	for {
		select {
		case <-ticker.C:
			log.Printf("%s main: %ds tick", rip.daemonName, tick)
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
	//cmdConH := command.CMD_CONF | command.CMD_HELP
	cmdConH := command.CMD_CONF

	command.CmdInstall(root, cmdConH, "hostname {HOSTNAME}", command.CONF, cmdHostname, command.ApplyBogus, "Hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdConH, "router rip", command.CONF, cmdRip, applyRip, "Enable RIP protocol")
	command.CmdInstall(root, cmdConH, "router rip network {NETWORK}", command.CONF, cmdRipNetwork, applyRipNet, "Insert network into RIP protocol")
	command.CmdInstall(root, cmdConH, "router rip vrf {VRFNAME} network {NETWORK}", command.CONF, cmdRipNetwork, applyRipNet, "Insert network into RIP protocol")

	command.DescInstall(root, "hostname", "Assign hostname")
	command.DescInstall(root, "router", "Configure routing")
	command.DescInstall(root, "router rip network", "Insert network into RIP protocol")

	command.MissingDescription(root)
}

func cmdHostname(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperHostname(ctx, node, line, c)
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	rip := ctx.(*Rip)
	command.HelperShowVersion(rip.daemonName, c)
}

func cmdRip(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	confCand := ctx.ConfRootCandidate()
	_, err, _ := confCand.Set(node.Path, line)
	if err != nil {
		c.Sendln(fmt.Sprintf("cmdRip: error: %v", err))
		return
	}
}

func cmdRipNetwork(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	linePath, netAddr := command.StripLastToken(line)

	path, _ := command.StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err1, _ := confCand.Set(path, linePath)
	if err1 != nil {
		c.Sendln(fmt.Sprintf("cmdRipNetwork: error: %v", err1))
		return
	}

	/*
		ripRouterNode, err2 := confCand.Get("router rip")
		c.Sendln(fmt.Sprintf("rip router: node=%v error=%v", ripRouterNode, err2))
	*/

	confNode.ValueAdd(netAddr)
}

func applyRip(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {
	return enableRip(ctx, node, action, c, false)
}

func applyRipNet(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {
	return enableRip(ctx, node, action, c, true)
}

func enableRip(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient, isNetCmd bool) error {
	rip := ctx.(*Rip)

	cand, _ := ctx.ConfRootCandidate().Get("router rip")

	if action.Enable {
		// enable RIP

		if rip.router == nil {
			rip.router = NewRipRouter()
		}

		if isNetCmd {
			// add network into rip
			f := strings.Fields(action.Cmd)
			if strings.HasPrefix("network", f[2]) {
				return rip.router.NetAdd("", f[3])
			}
			if strings.HasPrefix("vrf", f[2]) {
				return rip.router.NetAdd(f[3], f[5])
			}
			return fmt.Errorf("enableRip: bad network command: cmd=[%s] conf=[%s]", node.Path, action.Cmd)
		}

		return nil
	}

	// disable RIP

	if rip.router == nil {
		return nil // rip not running
	}

	if isNetCmd {
		// remove network from rip
		f := strings.Fields(action.Cmd)

		if strings.HasPrefix("network", f[2]) {
			if err := rip.router.NetDel("", f[3]); err != nil {
				return err
			}
		} else if strings.HasPrefix("vrf", f[2]) {
			if err := rip.router.NetDel(f[3], f[5]); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("enableRip: bad network command: cmd=[%s] conf=[%s]", node.Path, action.Cmd)
		}

		// router rip removed?
		if cand != nil {
			return nil // router rip still in place
		}

		// router rip removed, fully disable rip
	}

	// fully disable RIP

	rip.router.done <- 1 // request end of rip goroutine
	rip.router = nil

	return nil
}
