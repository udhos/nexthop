package main

import (
	"fmt"
	"log"
	//"strings"

	//"cli"
	"command"
)

func installRibCommands(root *command.CmdNode) {

	command.InstallCommonHelpers(root)

	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdConf, "interface {IFNAME} description {ANY}", command.CONF, cmdDescr, "Set interface description")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IPADDR}", command.CONF, cmdIfaceAddr, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IPADDR6}", command.CONF, cmdIfaceAddrIPv6, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} shutdown", command.CONF, cmdIfaceShutdown, "Disable interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} vrf {VRFNAME}", command.CONF, cmdIfaceVrf, "Assign VRF to interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdIPRouting, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, cmdHostname, "Assign hostname")
	command.CmdInstall(root, cmdNone, "show interface", command.EXEC, cmdShowInt, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show", command.EXEC, cmdShowInt, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "show ip address", command.EXEC, cmdShowIPAddr, "Show addresses")
	command.CmdInstall(root, cmdNone, "show ip interface", command.EXEC, cmdShowIPInt, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show ip interface detail", command.EXEC, cmdShowIPInt, "Show interface detail")
	command.CmdInstall(root, cmdNone, "show ip route", command.EXEC, cmdShowIPRoute, "Show routing table")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, "Show version")
	command.CmdInstall(root, cmdConf, "vrf {VRFNAME} ipv4 import route-target {RT}", command.CONF, cmdVrfImportRT, "Route-target for import")
	command.CmdInstall(root, cmdConf, "vrf {VRFNAME} ipv4 export route-target {RT}", command.CONF, cmdVrfExportRT, "Route-target for export")
}

func cmdDescr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperDescription(ctx, node, line, c)
}

func cmdIfaceAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperIfaceAddr(ctx, node, line, c)
}

func cmdIfaceAddrIPv6(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	linePath, addr := command.StripLastToken(line)
	log.Printf("cmdIfaceAddr: FIXME check IPv6/plen syntax: ipv6=%s", addr)

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
