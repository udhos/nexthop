package main

import (
	"fmt"
	"log"
	"strings"

	"cli"
	"command"
)

func installRibCommands(root *command.CmdNode) {

	cmdNone := command.CMD_NONE
	cmdConf := command.CMD_CONF

	command.CmdInstall(root, cmdNone, "commit", command.CONF, cmdCommit, "Apply current candidate configuration")
	command.CmdInstall(root, cmdNone, "configure", command.ENAB, cmdConfig, "Enter configuration mode")
	command.CmdInstall(root, cmdNone, "enable", command.EXEC, cmdEnable, "Enter privileged mode")
	command.CmdInstall(root, cmdNone, "exit", command.EXEC, cmdExit, "Exit current location")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IPADDR}", command.CONF, cmdIfaceAddr, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IPADDR6}", command.CONF, cmdIfaceAddrIPv6, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdIPRouting, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, cmdHostname, "Assign hostname")
	command.CmdInstall(root, cmdNone, "list", command.EXEC, cmdList, "List command tree")
	command.CmdInstall(root, cmdNone, "quit", command.EXEC, cmdQuit, "Quit session")
	command.CmdInstall(root, cmdNone, "reload", command.ENAB, cmdReload, "Reload")
	command.CmdInstall(root, cmdNone, "reload", command.ENAB, cmdReload, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "rollback", command.CONF, cmdRollback, "Reset candidate configuration from active configuration")
	command.CmdInstall(root, cmdNone, "rollback {ID}", command.CONF, cmdRollback, "Reset candidate configuration from rollback configuration")
	command.CmdInstall(root, cmdNone, "show interface", command.EXEC, cmdShowInt, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show", command.EXEC, cmdShowInt, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "show configuration", command.EXEC, cmdShowConf, "Show candidate configuration")
	command.CmdInstall(root, cmdNone, "show ip address", command.EXEC, cmdShowIPAddr, "Show addresses")
	command.CmdInstall(root, cmdNone, "show ip interface", command.EXEC, cmdShowIPInt, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show ip interface detail", command.EXEC, cmdShowIPInt, "Show interface detail")
	command.CmdInstall(root, cmdNone, "show ip route", command.EXEC, cmdShowIPRoute, "Show routing table")
	command.CmdInstall(root, cmdNone, "show running-configuration", command.EXEC, cmdShowRun, "Show active configuration")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, "Show version")
}

func cmdCommit(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdConfig(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	cc := c.(*cli.Client)
	status := cc.Status()
	if status < command.CONF {
		cc.StatusConf()
	}
	output := fmt.Sprintf("configure: new status=%d", cc.Status())
	log.Printf(output)
}

func cmdEnable(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	cc := c.(*cli.Client)
	status := cc.Status()
	if status < command.ENAB {
		cc.StatusEnable()
	}
	output := fmt.Sprintf("enable: new status=%d", cc.Status())
	log.Printf(output)
}

func cmdExit(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	cc := c.(*cli.Client)
	cc.StatusExit()
	log.Printf("exit: new status=%d", cc.Status())
}

func cmdIfaceAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {

	line, addr := command.StripLastToken(line)
	log.Printf("cmdIfaceAddr: FIXME check IPv4/plen syntax: ipv4=%s", addr)

	path, _ := command.StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		log.Printf("iface addr: error: %v", err)
		return
	}

	confNode.ValueAdd(addr)
}

func cmdIfaceAddrIPv6(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdIPRouting(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdHostname(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdQuit(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func list(node *command.CmdNode, depth int, c command.CmdClient) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}
	ident := strings.Repeat(" ", 4*depth)
	output := fmt.Sprintf("%s %d %s[%s] desc=[%s]", handler, node.MinLevel, ident, node.Path, node.Desc)
	log.Printf(output)
	c.Sendln(output)
	for _, n := range node.Children {
		list(n, depth+1, c)
	}
}

func cmdList(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	list(ctx.CmdRoot(), 0, c)
}

func cmdReload(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdRollback(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	fields := strings.Fields(line)
	if len(fields) > 1 {
		id := fields[1]
		log.Printf("cmdRollback: reset candidate config from rollback: %s", id)
	} else {
		log.Printf("cmdRollback: reset candidate config from active configuration")
	}
}

func cmdShowInt(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowConf(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	confCand := ctx.ConfRootCandidate()
	showConf(confCand, 0, c)
}

func showConf(node *command.ConfNode, depth int, c command.CmdClient) {
	ident := strings.Repeat(" ", depth)
	var last string
	if node.Path == "" {
		last = "config:"
	} else {
		last = command.LastToken(node.Path)
	}

	// show node path
	p := fmt.Sprintf("%s%s", ident, last)
	log.Printf(p)
	c.Sendln(p)

	// show node values
	for _, v := range node.Value {
		msg := fmt.Sprintf("%s %s", ident, v)
		log.Printf(msg)
		c.Sendln(msg)
	}

	for _, n := range node.Children {
		showConf(n, depth+1, c)
	}
}

func cmdShowIPAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPInt(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPRoute(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowRun(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	ribApp := ctx.(*RibApp)
	log.Printf("daemon: %v", ribApp.daemonName)
}
