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
	command.CmdInstall(root, cmdConf, "interface {IFNAME} shutdown", command.CONF, cmdIfaceShutdown, "Disable interface")
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

	i := command.IndexByte(ln, ' ', 3)
	if i < 0 {
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

func cmdIfaceShutdown(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	confCand := ctx.ConfRootCandidate()
	_, err, _ := confCand.Set(node.Path, line)
	if err != nil {
		log.Printf("iface shutdown: error: %v", err)
		return
	}
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

func cmdList(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	command.List(ctx.CmdRoot(), 0, c)
}

func cmdNo(ctx command.ConfContext, node *command.CmdNode, line string, c command.CmdClient) {
	c.Sendln(fmt.Sprintf("cmdNo: [%s]", line))

	sep := strings.IndexByte(line, ' ')
	if sep < 0 {
		c.Sendln(fmt.Sprintf("cmdNo: missing argument: %v", line))
		return
	}

	arg := line[sep:]

	cc := c.(*cli.Client)
	status := cc.Status()

	node, lookupPath, err := command.CmdFindRelative(ctx.CmdRoot(), arg, c.ConfigPath(), status)
	if err != nil {
		c.Sendln(fmt.Sprintf("cmdNo: not found [%s]: %v", arg, err))
		return
	}

	c.Sendln(fmt.Sprintf("cmdNo: found lookup=[%s] path=[%s]", lookupPath, node.Path))

	if !node.IsConfig() {
		c.Sendln(fmt.Sprintf("cmdNo: not a configuration command: [%s]", arg))
		return
	}

	expanded, e := command.CmdExpand(arg, node.Path)
	if e != nil {
		c.Sendln(fmt.Sprintf("cmdNo: could not expand path: %v", e))
		return
	}

	parentConf, e2 := ctx.ConfRootCandidate().GetParent(expanded)
	if e2 != nil {
		c.Sendln(fmt.Sprintf("cmdNo: config parent node not found [%s]: %v", expanded, e2))
		return
	}

	c.Sendln(fmt.Sprintf("cmdNo: config parent node found: parent=[%s] lookup=[%s]", parentConf.Path, expanded))
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
	command.ShowConf(root, node, c, lineMode)
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
