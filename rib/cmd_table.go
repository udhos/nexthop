package main

import (
	"fmt"
	//"log"
	"strings"

	//"cli"
	"github.com/udhos/nexthop/command"
)

func installRibCommands(root *command.CmdNode) {

	command.InstallCommonHelpers(root)

	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF
	//cmdConH := command.CMD_CONF | command.CMD_HELP
	cmdConH := cmdConf

	command.CmdInstall(root, cmdConH, "interface {IFNAME} description {ANY}", command.CONF, cmdDescr, command.ApplyBogus, "Interface description")
	command.CmdInstall(root, cmdConH, "interface {IFNAME} ipv4 address {IFADDR}", command.CONF, cmdIfaceAddr, applyIfaceAddr, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConH, "interface {IFNAME} ipv6 address {IFADDR6}", command.CONF, cmdIfaceAddrIPv6, command.ApplyBogus, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConH, "interface {IFNAME} shutdown", command.CONF, cmdIfaceShutdown, command.ApplyBogus, "Disable interface")
	command.CmdInstall(root, cmdConH, "interface {IFNAME} vrf {VRFNAME}", command.CONF, cmdIfaceVrf, applyIfaceVrf, "Interface VRF")
	command.CmdInstall(root, cmdConH, "ip routing", command.CONF, cmdIPRouting, command.ApplyBogus, "Enable IP routing")
	command.CmdInstall(root, cmdConH, "hostname (HOSTNAME)", command.CONF, cmdHostname, command.ApplyBogus, "Assign hostname")
	command.CmdInstall(root, cmdNone, "show interface", command.EXEC, cmdShowInt, nil, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show", command.EXEC, cmdShowInt, nil, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "show ip address", command.EXEC, cmdShowIPAddr, nil, "Show addresses")
	command.CmdInstall(root, cmdNone, "show ip interface", command.EXEC, cmdShowIPInt, nil, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show ip interface detail", command.EXEC, cmdShowIPInt, nil, "Show interface detail")
	command.CmdInstall(root, cmdNone, "show ip route", command.EXEC, cmdShowIPRoute, nil, "Show routing table")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, nil, "Show version")
	command.CmdInstall(root, cmdConf, "vrf {VRFNAME} ipv4 import route-target {RT}", command.CONF, cmdVrfImportRT, command.ApplyBogus, "Route-target for import")
	command.CmdInstall(root, cmdConf, "vrf {VRFNAME} ipv4 export route-target {RT}", command.CONF, cmdVrfExportRT, command.ApplyBogus, "Route-target for export")

	// Node description is used for pretty display in command help.
	// It is not strictly required, but its lack is reported by the command command.MissingDescription().
	command.DescInstall(root, "hostname", "Assign hostname")
	command.DescInstall(root, "interface", "Configure interface")
	command.DescInstall(root, "interface {IFNAME}", "Configure interface parameter")
	command.DescInstall(root, "interface {IFNAME} description", "Configure interface description")
	command.DescInstall(root, "interface {IFNAME} ipv4", "Configure interface IPv4 parameter")
	command.DescInstall(root, "interface {IFNAME} ipv6", "Configure interface IPv6 parameter")
	command.DescInstall(root, "interface {IFNAME} ipv4 address", "Configure interface IPv4 address")
	command.DescInstall(root, "interface {IFNAME} ipv6 address", "Configure interface IPv6 address")
	command.DescInstall(root, "interface {IFNAME} vrf", "Assign VRF to interface")
	command.DescInstall(root, "ip", "Configure IP parameter")
	command.DescInstall(root, "show ip", "Show IP information")
	command.DescInstall(root, "vrf", "Configure VRF")
	command.DescInstall(root, "vrf {VRFNAME}", "Configure VRF parameter")
	command.DescInstall(root, "vrf {VRFNAME} ipv4", "Configure VRF IPv4 parameter")
	command.DescInstall(root, "vrf {VRFNAME} ipv4 import", "Configure VRF import")
	command.DescInstall(root, "vrf {VRFNAME} ipv4 export", "Configure VRF export")
	command.DescInstall(root, "vrf {VRFNAME} ipv4 import route-target", "Import route target")
	command.DescInstall(root, "vrf {VRFNAME} ipv4 export route-target", "Export route target")

	command.MissingDescription(root)
}

func cmdDescr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.HelperIfaceDescr(ctx, node, line, c)
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
			if a.String() == ifaddr {
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
		if a.String() == ifaddr {
			return fmt.Errorf("applyIfaceAddr: deleted address found")
		}
	}

	return nil // success
}

func applyIfaceVrf(ctx command.ConfContext, node *command.CmdNode, action command.CommitAction, c command.CmdClient) error {

	app := ctx.(*RibApp)
	hw := app.hardware

	fields := strings.Fields(action.Cmd)
	ifname := fields[1]
	vrfName := fields[3]

	if action.Enable {

		if err := hw.InterfaceVrf(ifname, vrfName); err != nil {
			return fmt.Errorf("applyIfaceVrf: error: %v", err)
		}

		ifnames, vrfnames, err := hw.Interfaces()
		if err != nil {
			return fmt.Errorf("applyIfaceVrf: error querying interfaces: %v", err)
		}
		for i, ifn := range ifnames {
			if ifn == ifname && vrfnames[i] == vrfName {
				return nil // success
			}
		}

		return fmt.Errorf("applyIfaceVrf: iface/vrf not found")
	}

	if err := hw.InterfaceVrf(ifname, ""); err != nil {
		return fmt.Errorf("applyIfaceVrf: del vrf error: %v", err)
	}

	ifnames, vrfnames, err := hw.Interfaces()
	if err != nil {
		return fmt.Errorf("applyIfaceVrf: error querying interfaces: %v", err)
	}
	for i, ifn := range ifnames {
		if ifn == ifname && vrfnames[i] == vrfName {
			return fmt.Errorf("applyIfaceVrf: deleted iface/vrf found")
		}
	}

	return nil // success
}

func cmdIfaceAddrIPv6(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}

func cmdIfaceShutdown(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}

func cmdIfaceVrf(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}

func cmdIPRouting(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
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
	ribApp := ctx.(*RibApp)
	command.HelperShowVersion(ribApp.daemonName, c)
}

func cmdVrfImportRT(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}

func cmdVrfExportRT(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.SetSimple(ctx, c, node.Path, line)
}
