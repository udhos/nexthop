package main

import (
	"fmt"
	"log"
	"strings"

	"command"
)

func installRibCommands(root *command.CmdNode) {
	command.CmdInstall(root, "configure", command.ENAB, cmdConfig, "Enter configuration mode")
	command.CmdInstall(root, "enable", command.EXEC, cmdEnable, "Enter privileged mode")
	command.CmdInstall(root, "interface {IFNAME} ip address {IPADDR}", command.CONF, cmdIfaceAddr, "Assign IPv4 address to interface")
	command.CmdInstall(root, "interface {IFNAME} ipv6 address {IPADDR6}", command.CONF, cmdIfaceAddrIPv6, "Assign IPv6 address to interface")
	command.CmdInstall(root, "ip routing", command.CONF, cmdIPRouting, "Enable IP routing")
	command.CmdInstall(root, "hostname {HOSTNAME}", command.CONF, cmdHostname, "Assign hostname")
	command.CmdInstall(root, "list", command.EXEC, cmdList, "List command tree")
	command.CmdInstall(root, "quit", command.EXEC, cmdQuit, "Quit session")
	command.CmdInstall(root, "reload", command.ENAB, cmdReload, "Reload")
	command.CmdInstall(root, "reload", command.ENAB, cmdReload, "Ugh") // duplicated command
	command.CmdInstall(root, "show interface", command.EXEC, cmdShowInt, "Show interfaces")
	command.CmdInstall(root, "show", command.EXEC, cmdShowInt, "Ugh") // duplicated command
	command.CmdInstall(root, "show configuration", command.EXEC, cmdShowConf, "Show candidate configuration")
	command.CmdInstall(root, "show ip address", command.EXEC, cmdShowIPAddr, "Show addresses")
	command.CmdInstall(root, "show ip interface", command.EXEC, cmdShowIPInt, "Show interfaces")
	command.CmdInstall(root, "show ip interface detail", command.EXEC, cmdShowIPInt, "Show interface detail")
	command.CmdInstall(root, "show ip route", command.EXEC, cmdShowIPRoute, "Show routing table")
	command.CmdInstall(root, "show running-configuration", command.EXEC, cmdShowRun, "Show active configuration")
}

func cmdConfig(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdEnable(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdIfaceAddr(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdIfaceAddrIPv6(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdIPRouting(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdHostname(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdQuit(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func list(node *command.CmdNode, depth int, c command.CmdClient) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}
	ident := strings.Repeat(" ", 4*depth)
	//c.userOut <- fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	//sendln(c, fmt.Sprintf("%s %d %s[%s] desc=[%s]", handler, node.MinLevel, ident, node.Path, node.Desc))
	output := fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	log.Printf(output)
	for _, n := range node.Children {
		list(n, depth+1, c)
	}
}

func cmdList(ctx *command.ConfContext, line string, c command.CmdClient) {
	list(ctx.CmdRoot, 0, c)
}

func cmdReload(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdShowInt(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdShowConf(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdShowIPAddr(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdShowIPInt(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdShowIPRoute(ctx *command.ConfContext, line string, c command.CmdClient) {
}

func cmdShowRun(ctx *command.ConfContext, line string, c command.CmdClient) {
}
