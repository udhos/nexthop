package main

import (
	"fmt"
	"log"
	"strings"

	"cli"
	"command"
)

func installRibCommands(root *command.CmdNode) {
	command.CmdInstall(root, "configure", command.ENAB, cmdConfig, "Enter configuration mode")
	command.CmdInstall(root, "enable", command.EXEC, cmdEnable, "Enter privileged mode")
	command.CmdInstall(root, "interface {IFNAME} ipv4 address {IPADDR}", command.CONF, cmdIfaceAddr, "Assign IPv4 address to interface")
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
	command.CmdInstall(root, "show version", command.EXEC, cmdVersion, "Show version")
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

func cmdIfaceAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {

	line, addr := command.StripLastToken(line)
	log.Printf("cmdIfaceAddr: FIXME check IPv4/plen syntax: ipv4=%s", addr)

	path, _ := command.StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		output := fmt.Sprintf("iface addr: error: %v", err)
		log.Printf(output)
		return
	}

	confNode.Value = append(confNode.Value, addr)

	log.Printf(fmt.Sprintf("iface addr: config node: %v", confNode))

	log.Printf(fmt.Sprintf("iface addr: config full: %v", confCand))
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
	//c.userOut <- fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	//sendln(c, fmt.Sprintf("%s %d %s[%s] desc=[%s]", handler, node.MinLevel, ident, node.Path, node.Desc))
	output := fmt.Sprintf("%s %d %s[%s] desc=[%s]\r\n", handler, node.MinLevel, ident, node.Path, node.Desc)
	log.Printf(output)
	for _, n := range node.Children {
		list(n, depth+1, c)
	}
}

func cmdList(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	list(ctx.CmdRoot(), 0, c)
}

func cmdReload(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
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
	log.Printf("%s%s", ident, last)
	for _, v := range node.Value {
		log.Printf("%s %s", ident, v)
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
