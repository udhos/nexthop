package main

import (
	"flag"
	"fmt"
	"fwd"
	"log"
	"net"
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

	command.CmdInstall(root, cmdConH, "hostname {HOSTNAME}", command.CONF, command.HelperHostname, command.ApplyBogus, "Hostname")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdNone, "show rip routes", command.EXEC, cmdShowRipRoutes, nil, "Show RIP routes")
	command.CmdInstall(root, cmdConH, "router rip", command.CONF, cmdRip, applyRip, "Enable RIP protocol")
	command.CmdInstall(root, cmdConH, "router rip network {NETWORK}", command.CONF, cmdRipNetwork, applyRipNet, "Insert network into RIP protocol")
	command.CmdInstall(root, cmdConH, "router rip network {NETWORK} cost {RIPMETRIC}", command.CONF, cmdRipNetCost, applyRipNetCost, "RIP network metric")
	command.CmdInstall(root, cmdConH, "router rip network {NETWORK} nexthop {IPADDR}", command.CONF, cmdRipNetNexthop, applyRipNetNexthop, "RIP network nexthop")
	command.CmdInstall(root, cmdConH, "router rip network {NETWORK} nexthop {IPADDR} cost {RIPMETRIC}", command.CONF, cmdRipNetNexthopCost, applyRipNetNexthopCost, "RIP network metric")
	command.CmdInstall(root, cmdConH, "router rip vrf {VRFNAME} network {NETWORK}", command.CONF, cmdRipNetwork, applyRipVrfNet, "Insert network into RIP protocol")
	command.CmdInstall(root, cmdConH, "router rip vrf {VRFNAME} network {NETWORK} cost {RIPMETRIC}", command.CONF, cmdRipNetCost, applyRipVrfNetCost, "RIP network metric")
	command.CmdInstall(root, cmdConH, "router rip vrf {VRFNAME} network {NETWORK} nexthop {IPADDR}", command.CONF, cmdRipNetNexthop, applyRipVrfNetNexthop, "RIP network nexthop")
	command.CmdInstall(root, cmdConH, "router rip vrf {VRFNAME} network {NETWORK} nexthop {IPADDR} cost {RIPMETRIC}", command.CONF, cmdRipNetNexthopCost, applyRipVrfNetNexthopCost, "RIP network metric")

	// Node description is used for pretty display in command help.
	// It is not strictly required, but its lack is reported by the command command.MissingDescription().
	command.DescInstall(root, "hostname", "Assign hostname")
	command.DescInstall(root, "router", "Configure routing")
	command.DescInstall(root, "router rip network", "Insert network into RIP protocol")
	command.DescInstall(root, "router rip network {NETWORK} cost", "RIP network cost")
	command.DescInstall(root, "router rip vrf", "Insert network into RIP protocol for specific VRF")
	command.DescInstall(root, "router rip vrf {VRFNAME}", "Insert network into RIP protocol for specific VRF")
	command.DescInstall(root, "router rip vrf {VRFNAME} network", "Insert network into RIP protocol for specific VRF")
	command.DescInstall(root, "router rip vrf {VRFNAME} network {NETWORK} cost", "RIP network cost")

	command.MissingDescription(root)
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	rip := ctx.(*Rip)
	command.HelperShowVersion(rip.daemonName, c)
}

func cmdShowRipRoutes(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	rip := ctx.(*Rip)
	if rip.router == nil {
		c.Sendln("RIP not running")
		return
	}
	rip.router.ShowRoutes(c)
}

func cmdRip(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	/*
		confCand := ctx.ConfRootCandidate()
		_, err, _ := confCand.Set(node.Path, line)
		if err != nil {
			c.Sendln(fmt.Sprintf("cmdRip: error: %v", err))
			return
		}
	*/

	command.SetSimple(ctx, c, node.Path, line)
}

func cmdRipNetwork(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	/*
		linePath, netAddr := command.StripLastToken(line)

		path, _ := command.StripLastToken(node.Path)

		confCand := ctx.ConfRootCandidate()
		confNode, err1, _ := confCand.Set(path, linePath)
		if err1 != nil {
			c.Sendln(fmt.Sprintf("cmdRipNetwork: error: %v", err1))
			return
		}

		confNode.ValueAdd(netAddr)
	*/

	command.MultiValueAdd(ctx, c, node.Path, line)
}

func cmdRipNetCost(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SingleValueSetSimple(ctx, c, node.Path, line)
}

func cmdRipNetNexthop(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.MultiValueAdd(ctx, c, node.Path, line)
}

func cmdRipNetNexthopCost(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SingleValueSetSimple(ctx, c, node.Path, line)
}

func ripCtx(ctx command.ConfContext, c command.CmdClient) *Rip {
	if rip, ok := ctx.(*Rip); ok {
		return rip
	}
	// non-rip context is a bogus state used for unit testing
	err := fmt.Errorf("ripCtx: not a true Rip context: %v", ctx)
	log.Printf("%v", err)
	c.Sendln(fmt.Sprintf("%v", err))
	return nil
}

func applyRip(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	enableRip(rip, action.Enable)

	return nil
}

func applyRipNet(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := ""
	netAddr := f[3]

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetAdd(vrf, netAddr)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipNet: rip router disabled")
	}

	if err := rip.router.NetDel(vrf, netAddr); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipVrfNet(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := f[3]
	netAddr := f[5]

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetAdd(vrf, netAddr)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipVrfNet: rip router disabled")
	}

	if err := rip.router.NetDel(vrf, netAddr); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipNetCost(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	vrf := ""
	f := strings.Fields(action.Cmd)
	netAddr := f[3]
	metricStr := f[5]

	metric, err := strconv.Atoi(metricStr)
	if err != nil {
		return fmt.Errorf("applyRipNetCost: bad metric: '%s': %v", metricStr, err)
	}

	nexthop := net.IPv4zero

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetMetricAdd(vrf, netAddr, nexthop, metric)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipNetCost: rip router disabled")
	}

	if err := rip.router.NetMetricDel(vrf, netAddr, nexthop, metric); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipVrfNetCost(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := f[3]
	netAddr := f[5]
	metricStr := f[7]

	metric, err := strconv.Atoi(metricStr)
	if err != nil {
		return fmt.Errorf("applyRipNetCost: bad metric: '%s': %v", metricStr, err)
	}

	nexthop := net.IPv4zero

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetMetricAdd(vrf, netAddr, nexthop, metric)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipVrfNetCost: rip router disabled")
	}

	if err := rip.router.NetMetricDel(vrf, netAddr, nexthop, metric); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipNetNexthop(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := ""
	netAddr := f[3]
	nexthopStr := f[5]

	nexthop := net.ParseIP(nexthopStr)
	if nexthop == nil {
		return fmt.Errorf("applyRipNetNexthop: bad address: '%s'", nexthopStr)
	}

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetNexthopAdd(vrf, netAddr, nexthop)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipNetNexthop: rip router disabled")
	}

	if err := rip.router.NetNexthopDel(vrf, netAddr, nexthop); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipVrfNetNexthop(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := f[3]
	netAddr := f[5]
	nexthopStr := f[7]

	nexthop := net.ParseIP(nexthopStr)
	if nexthop == nil {
		return fmt.Errorf("applyRipVrfNetNexthop: bad address: '%s'", nexthopStr)
	}

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetNexthopAdd(vrf, netAddr, nexthop)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipVrfNetNexthop: rip router disabled")
	}

	if err := rip.router.NetNexthopDel(vrf, netAddr, nexthop); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipNetNexthopCost(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := ""
	netAddr := f[3]
	nexthopStr := f[5]
	metricStr := f[7]

	nexthop := net.ParseIP(nexthopStr)
	if nexthop == nil {
		return fmt.Errorf("applyRipNetNexthopCost: bad address: '%s'", nexthopStr)
	}

	metric, err1 := strconv.Atoi(metricStr)
	if err1 != nil {
		return fmt.Errorf("applyRipNetNexthopCost: bad metric: '%s': %v", metricStr, err1)
	}

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetMetricAdd(vrf, netAddr, nexthop, metric)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipNetNexthopCost: rip router disabled")
	}

	if err := rip.router.NetMetricDel(vrf, netAddr, nexthop, metric); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func applyRipVrfNetNexthopCost(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	rip := ripCtx(ctx, c)
	if rip == nil {
		return nil
	}

	f := strings.Fields(action.Cmd)
	vrf := f[3]
	netAddr := f[5]
	nexthopStr := f[7]
	metricStr := f[9]

	nexthop := net.ParseIP(nexthopStr)
	if nexthop == nil {
		return fmt.Errorf("applyRipVrfNetNexthopCost: bad address: '%s'", nexthopStr)
	}

	metric, err1 := strconv.Atoi(metricStr)
	if err1 != nil {
		return fmt.Errorf("applyRipVrfNetNexthopCost: bad metric: '%s': %v", metricStr, err1)
	}

	if action.Enable {
		enableRip(rip, true) // try to enable rip
		return rip.router.NetMetricAdd(vrf, netAddr, nexthop, metric)
	}

	if rip.router == nil {
		return fmt.Errorf("applyRipVrfNetNexthopCost: rip router disabled")
	}

	if err := rip.router.NetMetricDel(vrf, netAddr, nexthop, metric); err != nil {
		return err
	}

	enableRip(rip, false) // disable rip if needed

	return nil
}

func enableRip(rip *Rip, enable bool) {

	if enable {
		// enable RIP

		if rip.router == nil {
			rip.router = NewRipRouter(rip.hardware, rip)
		}

		return
	}

	// disable RIP

	if cand, _ := rip.ConfRootCandidate().Get("router rip"); cand != nil {
		return // router rip still in place
	}

	//log.Printf("enableRip: disabling RIP")

	if rip.router == nil {
		return // rip not running
	}

	// fully disable RIP

	rip.router.done <- 1 // request end of rip goroutine
	rip.router = nil
}

/*
func enableRip(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient, isNetCmd bool, cost int, nexthop net.IP) error {
	var rip *Rip
	var ok bool
	if rip, ok = ctx.(*Rip); !ok {
		// non-rip context is a bogus state used for unit testing
		err := fmt.Errorf("enableRip: not a true Rip context: %v", ctx)
		log.Printf("%v", err)
		c.Sendln(fmt.Sprintf("%v", err))
		return nil // fake success
	}

	cand, _ := ctx.ConfRootCandidate().Get("router rip")

	if action.Enable {
		// enable RIP

		if rip.router == nil {
			rip.router = NewRipRouter(rip.hardware, ctx)
		}

		if isNetCmd {
			// add network into rip

			f := strings.Fields(action.Cmd)
			if strings.HasPrefix("network", f[2]) {
				return rip.router.NetAdd("", f[3], cost, nexthop)
			}
			if strings.HasPrefix("vrf", f[2]) {
				return rip.router.NetAdd(f[3], f[5], cost, nexthop)
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
			if err := rip.router.NetDel("", f[3], nexthop); err != nil {
				return err
			}
		} else if strings.HasPrefix("vrf", f[2]) {
			if err := rip.router.NetDel(f[3], f[5], nexthop); err != nil {
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
*/
