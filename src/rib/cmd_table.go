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
	command.CmdInstall(root, cmdConf, "interface {IFNAME} description {ANY}", command.CONF, cmdDescr, "Set interface description")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv4 address {IPADDR}", command.CONF, cmdIfaceAddr, "Assign IPv4 address to interface")
	command.CmdInstall(root, cmdConf, "interface {IFNAME} ipv6 address {IPADDR6}", command.CONF, cmdIfaceAddrIPv6, "Assign IPv6 address to interface")
	command.CmdInstall(root, cmdConf, "ip routing", command.CONF, cmdIPRouting, "Enable IP routing")
	command.CmdInstall(root, cmdConf, "hostname {HOSTNAME}", command.CONF, cmdHostname, "Assign hostname")
	command.CmdInstall(root, cmdNone, "list", command.EXEC, cmdList, "List command tree")
	command.CmdInstall(root, cmdNone, "no {ANY}", command.EXEC, cmdNo, "Remove a configuration item")
	command.CmdInstall(root, cmdNone, "quit", command.EXEC, cmdQuit, "Quit session")
	command.CmdInstall(root, cmdNone, "reload", command.ENAB, cmdReload, "Reload")
	command.CmdInstall(root, cmdNone, "reload", command.ENAB, cmdReload, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "rollback", command.CONF, cmdRollback, "Reset candidate configuration from active configuration")
	command.CmdInstall(root, cmdNone, "rollback {ID}", command.CONF, cmdRollback, "Reset candidate configuration from rollback configuration")
	command.CmdInstall(root, cmdNone, "show interface", command.EXEC, cmdShowInt, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show", command.EXEC, cmdShowInt, "Ugh") // duplicated command
	command.CmdInstall(root, cmdNone, "show configuration", command.EXEC, cmdShowConf, "Show candidate configuration")
	command.CmdInstall(root, cmdNone, "show configuration line-mode", command.EXEC, cmdShowConf, "Show candidate configuration in line-mode")
	command.CmdInstall(root, cmdNone, "show ip address", command.EXEC, cmdShowIPAddr, "Show addresses")
	command.CmdInstall(root, cmdNone, "show ip interface", command.EXEC, cmdShowIPInt, "Show interfaces")
	command.CmdInstall(root, cmdNone, "show ip interface detail", command.EXEC, cmdShowIPInt, "Show interface detail")
	command.CmdInstall(root, cmdNone, "show ip route", command.EXEC, cmdShowIPRoute, "Show routing table")
	command.CmdInstall(root, cmdNone, "show running-configuration", command.EXEC, cmdShowRun, "Show active configuration")
	command.CmdInstall(root, cmdNone, "show version", command.EXEC, cmdVersion, "Show version")
}

func cmdCommit(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	// get diff from active conf to candidate conf
	// build command list to apply diff to active conf
	//  - include preparatory commands, like deleting addresses from interfaces affected by address change
	//  - if any command fails, revert previously applied commands
	// save new active conf with new commit id
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

	path := c.ConfigPath()
	if path == "" {
		cc.StatusExit()
		//log.Printf("exit: new status=%d", cc.Status())
		return
	}

	fields := strings.Fields(path)
	newPath := strings.Join(fields[:len(fields)-1], " ")

	c.ConfigPathSet(newPath)
}

func cmdDescr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	// line: "interf  XXXX   descrip   YYY ZZZ WWW"
	//                                 ^^^^^^^^^^^

	// find 3rd space
	ln := strings.TrimLeft(line, " ") // drop leading spaces
	findSpc := false                  // find space
	found := 0
	var i int
	for i = 0; i < len(ln); i++ {
		if findSpc {
			if ln[i] == ' ' {
				found++
				if found == 3 {
					break
				}
				findSpc = false
			}
		} else {
			if ln[i] != ' ' {
				findSpc = true
			}
		}
	}

	if found != 3 {
		c.Sendln(fmt.Sprintf("cmdDescr: could not find description argument: [%s]", line))
		return
	}

	desc := ln[i+1:]

	lineFields := strings.Fields(line)
	linePath := strings.Join(lineFields[:3], " ")

	fields := strings.Fields(node.Path)
	path := strings.Join(fields[:3], " ") // interface XXX description

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, linePath)
	if err != nil {
		log.Printf("description: error: %v", err)
		return
	}

	confNode.ValueSet(desc)
}

func cmdIfaceAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {

	linePath, addr := command.StripLastToken(line)
	log.Printf("cmdIfaceAddr: FIXME check IPv4/plen syntax: ipv4=%s", addr)

	path, _ := command.StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, linePath)
	if err != nil {
		log.Printf("iface addr: error: %v", err)
		return
	}

	confNode.ValueAdd(addr)
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

func cmdIPRouting(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdHostname(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	line, host := command.StripLastToken(line)

	path, _ := command.StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		log.Printf("hostname: error: %v", err)
		return
	}

	confNode.ValueSet(host)
}

func cmdQuit(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	c.SendlnNow("")
	c.SendlnNow("bye")
	log.Printf("cmdQuit: requesting intputLoop to quit")
	c.InputQuit()
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

func cmdNo(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	c.Sendln(fmt.Sprintf("no: [%s]", line))
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
	showConfig(ctx.ConfRootCandidate(), node, line, c, "candidate configuration:")
}

func showConfig(root *command.ConfNode, node *command.CmdNode, line string, c command.CmdClient, head string) {
	fields := strings.Fields(line)
	lineMode := len(fields) > 2 && strings.HasPrefix("line-mode", fields[2])
	c.Sendln(head)
	for _, n := range root.Children {
		if lineMode {
			showConfLine(n, 0, c)
		} else {
			showConf(n, 0, 1, c, false)
		}
	}
}

func showConfLine(node *command.ConfNode, depth int, c command.CmdClient) {

	// show node values
	for _, v := range node.Value {
		c.Sendln(fmt.Sprintf("%s %s", node.Path, v))
	}

	for _, n := range node.Children {
		showConfLine(n, depth+1, c)
	}
}

func showConf(node *command.ConfNode, depth, valueDepth int, c command.CmdClient, hasSibling bool) {

	var ident string
	nodeIdent := strings.Repeat(" ", 2*depth)
	last := command.LastToken(node.Path)

	childrenCount := len(node.Children)
	valueCount := len(node.Value)
	fanout := childrenCount + valueCount

	// show path
	if hasSibling {
		ident = nodeIdent
	} else {
		ident = ""
	}
	var identValue bool // need to increment identation for value
	if fanout > 1 {
		c.Sendln(fmt.Sprintf("%s%s", ident, last))
		identValue = true
	} else {
		c.Send(fmt.Sprintf("%s%s ", ident, last))
		identValue = false
	}

	// show value
	if valueCount == 1 {
		c.Sendln(fmt.Sprintf("%s", node.Value[0]))
	} else {
		tab := strings.Repeat(" ", 2*valueDepth)
		for _, v := range node.Value {
			c.Sendln(fmt.Sprintf("%s%s", tab, v))
		}
	}

	if identValue {
		valueDepth++
	}

	for _, n := range node.Children {
		showConf(n, depth+1, valueDepth, c, childrenCount > 1)
	}
}

func cmdShowIPAddr(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPInt(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowIPRoute(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
}

func cmdShowRun(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	showConfig(ctx.ConfRootActive(), node, line, c, "running configuration:")
}

func cmdVersion(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	ribApp := ctx.(*RibApp)
	log.Printf("daemon: %v", ribApp.daemonName)
}
