package main

import (
	"fmt"
	"log"
	"strings"

	//"cli"
	"command"
)

func installRibCommands(root *command.CmdNode) {

	command.InstallCommonHelpers(root)

	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdConf, "interface {IFNAME} description {ANY}", command.CONF, cmdDescr, command.ApplyBogus, "Set interface description")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IFADDR}", command.CONF, cmdIfaceAddr, applyIfaceAddr, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IFADDR6}", command.CONF, cmdIfaceAddrIPv6, command.ApplyBogus, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} shutdown", command.CONF, cmdIfaceShutdown, command.ApplyBogus, "Disable interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} vrf {VRFNAME}", command.CONF, cmdIfaceVrf, command.ApplyBogus, "Assign VRF to interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdIPRouting, command.ApplyBogus, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, cmdHostname, command.ApplyBogus, "Assign hostname")
	command.CmdInstall(root, cmdNone, "show interface", command.EXEC, cmdShowInt, nil, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show", command.EXEC, cmdShowInt, nil, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "show ip address", command.EXEC, cmdShowIPAddr, nil, "Show addresses")
	command.CmdInstall(root, cmdNone, "show ip interface", command.EXEC, cmdShowIPInt, nil, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show ip interface detail", command.EXEC, cmdShowIPInt, nil, "Show interface detail")
	command.CmdInstall(root, cmdNone, "show ip route", command.EXEC, cmdShowIPRoute, nil, "Show routing table")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdConf, "vrf {VRFNAME} ipv4 import route-target {RT}", command.CONF, cmdVrfImportRT, command.ApplyBogus, "Route-target for import")
	command.CmdInstall(root, cmdConf, "vrf {VRFNAME} ipv4 export route-target {RT}", command.CONF, cmdVrfExportRT, command.ApplyBogus, "Route-target for export")
}

func cmdDescr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperDescription(ctx, node, line, c)
}

func cmdIfaceAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperIfaceAddr(ctx, node, line, c)
}

func applyIfaceAddr(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	app := ctx.(*RibApp)
	hw := app.hardware

	fields := strings.Fields(action.Cmd)
	ifname := fields[1]
	ifaddr := fields[4]

	if action.Enable {

		if err := hw.InterfaceAddressAdd(ifname, ifaddr); err != nil {
			return fmt.Errorf("applyIfaceAddr: add addr error: %v", err)
		}

		addrs, err1 := hw.InterfaceAddressGet(ifname)
		if err1 != nil {
			return fmt.Errorf("applyIfaceAddr: get addr error: %v", err1)
		}

		for _, a := range addrs {
			if a == ifaddr {
				return nil // success
			}
		}

		return fmt.Errorf("applyIfaceAddr: added address not found")
	}

	if err := hw.InterfaceAddressDel(ifname, ifaddr); err != nil {
		return fmt.Errorf("applyIfaceAddr: del addr error: %v", err)
	}

	addrs, err1 := hw.InterfaceAddressGet(ifname)
	if err1 != nil {
		return fmt.Errorf("applyIfaceAddr: get addr error: %v", err1)
	}

	for _, a := range addrs {
		if a == ifaddr {
			return fmt.Errorf("applyIfaceAddr: deleted address found")
		}
	}

	return nil // success
}

func cmdIfaceAddrIPv6(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	linePath, addr := command.StripLastToken(line)

	path, _ := command.StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, linePath)
	if err != nil {
		log.Printf("iface addr6: error: %v", err)
		return
	}

	confNode.ValueAdd(addr)
}

func cmdIfaceShutdown(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	confCand := ctx.ConfRootCandidate()
	_, err, _ := confCand.Set(node.Path, line)
	if err != nil {
		log.Printf("iface shutdown: error: %v", err)
		return
	}
}

func cmdIfaceVrf(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdIPRouting(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdHostname(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperHostname(ctx, node, line, c)
}

func cmdShowInt(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPInt(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPRoute(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	c.Sendln(command.NexthopVersion)
	ribApp := ctx.(*RibApp)
	c.Sendln(fmt.Sprintf("daemon: %v", ribApp.daemonName))
}

func cmdVrfImportRT(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdVrfExportRT(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}
