package command

import (
	"fmt"
	"log"
	"strings"
)

func InstallCommonHelpers(root *CmdNode) {

	cmdNone := CMD_NONE
	//cmdConf := CMD_CONF

	CmdInstall(root, cmdNone, "commit", CONF, cmdCommit, "Apply current candidate configuration")
	CmdInstall(root, cmdNone, "configure", ENAB, cmdConfig, "Enter configuration mode")
	CmdInstall(root, cmdNone, "enable", EXEC, cmdEnable, "Enter privileged mode")
	CmdInstall(root, cmdNone, "exit", EXEC, cmdExit, "Exit current location")
	CmdInstall(root, cmdNone, "list", EXEC, cmdList, "List command tree")
	CmdInstall(root, cmdNone, "no {ANY}", CONF, HelperNo, "Remove a configuration item")
	CmdInstall(root, cmdNone, "quit", EXEC, cmdQuit, "Quit session")
	CmdInstall(root, cmdNone, "reload", ENAB, cmdReload, "Reload")
	CmdInstall(root, cmdNone, "rollback", CONF, cmdRollback, "Reset candidate configuration from active configuration")
	CmdInstall(root, cmdNone, "rollback {ID}", CONF, cmdRollback, "Reset candidate configuration from rollback configuration")
	CmdInstall(root, cmdNone, "show configuration", EXEC, cmdShowConf, "Show candidate configuration")
	CmdInstall(root, cmdNone, "show configuration line-mode", EXEC, cmdShowConf, "Show candidate configuration in line-mode")
	CmdInstall(root, cmdNone, "show history", EXEC, cmdShowHistory, "Show command history")
	CmdInstall(root, cmdNone, "show running-configuration", EXEC, cmdShowRun, "Show active configuration")
}

func cmdCommit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	// get diff from active conf to candidate conf
	// build command list to apply diff to active conf
	//  - include preparatory commands, like deleting addresses from interfaces affected by address change
	//  - if any command fails, revert previously applied commands
	// save new active conf with new commit id

	confOld := ctx.ConfRootActive()
	confNew := ctx.ConfRootCandidate()
	cmdList := diff(confOld, confNew)
	for _, conf := range cmdList {
		c.Sendln(fmt.Sprintf("commit: %s", conf))
	}
}

func diff(root1, root2 *ConfNode) []string {
	list := []string{"command1", "command2"}
	return list
}

func cmdConfig(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	status := c.Status()
	if status < CONF {
		c.StatusConf()
	}
}

func cmdEnable(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	status := c.Status()
	if status < ENAB {
		c.StatusEnable()
	}
}

func cmdExit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {

	path := c.ConfigPath()
	if path == "" {
		if c.Status() <= EXEC {
			c.Sendln("use 'quit' to exit remote terminal")
			return
		}
		c.StatusExit()
		return
	}

	fields := strings.Fields(path)
	newPath := strings.Join(fields[:len(fields)-1], " ")

	c.ConfigPathSet(newPath)
}

func cmdQuit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.SendlnNow("")
	c.SendlnNow("bye")
	log.Printf("cmdQuit: requesting intputLoop to quit")
	c.InputQuit()
}

func cmdReload(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
}

func cmdRollback(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	fields := strings.Fields(line)
	if len(fields) > 1 {
		id := fields[1]
		log.Printf("cmdRollback: reset candidate config from rollback: %s", id)
	} else {
		log.Printf("cmdRollback: reset candidate config from active configuration")
	}
}

func cmdShowConf(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	showConfig(ctx.ConfRootCandidate(), node, line, c, "candidate configuration:")
}

func showConfig(root *ConfNode, node *CmdNode, line string, c CmdClient, head string) {
	fields := strings.Fields(line)
	lineMode := len(fields) > 2 && strings.HasPrefix("line-mode", fields[2])
	c.Sendln(head)
	ShowConf(root, node, c, lineMode)
}

func cmdShowHistory(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln("command history:")
	c.HistoryShow()
}

func cmdShowRun(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	showConfig(ctx.ConfRootActive(), node, line, c, "running configuration:")
}

// Iface addr config should not be a helper function,
// since it only applies to RIB daemon.
// However it is currently being used for helping in tests.
func HelperIfaceAddr(ctx ConfContext, node *CmdNode, line string, c CmdClient) {

	linePath, addr := StripLastToken(line)
	log.Printf("cmdIfaceAddr: FIXME check IPv4/plen syntax: ipv4=%s", addr)

	path, _ := StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, linePath)
	if err != nil {
		log.Printf("iface addr: error: %v", err)
		return
	}

	confNode.ValueAdd(addr)
}

func HelperDescription(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	// line: "interf  XXXX   descrip   YYY ZZZ WWW"
	//                                 ^^^^^^^^^^^

	// find 3rd space
	ln := strings.TrimLeft(line, " ") // drop leading spaces

	i := IndexByte(ln, ' ', 3)
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

func HelperHostname(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	line, host := StripLastToken(line)

	path, _ := StripLastToken(node.Path)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		log.Printf("hostname: error: %v", err)
		return
	}

	confNode.ValueSet(host)
}

func cmdList(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	list(ctx.CmdRoot(), 0, c)
}

func list(node *CmdNode, depth int, c CmdClient) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}
	ident := strings.Repeat(" ", 4*depth)
	output := fmt.Sprintf("%s %d %s[%s] desc=[%s]", handler, node.MinLevel, ident, node.Path, node.Desc)
	//log.Printf(output)
	c.Sendln(output)
	for _, n := range node.Children {
		list(n, depth+1, c)
	}
}

func HelperNo(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln(fmt.Sprintf("cmdNo: [%s]", line))

	sep := strings.IndexByte(line, ' ')
	if sep < 0 {
		c.Sendln(fmt.Sprintf("cmdNo: missing argument: %v", line))
		return
	}

	arg := line[sep:]

	//cc := c.(*cli.Client)
	//status := cc.Status()
	status := c.Status()

	node, _, err := CmdFindRelative(ctx.CmdRoot(), arg, c.ConfigPath(), status)
	if err != nil {
		c.Sendln(fmt.Sprintf("cmdNo: not found [%s]: %v", arg, err))
		return
	}

	if !node.IsConfig() {
		c.Sendln(fmt.Sprintf("cmdNo: not a configuration command: [%s]", arg))
		return
	}

	matchAny := node.MatchAny()
	childMatchAny := !matchAny && len(node.Children) == 1 && node.Children[0].MatchAny()

	c.Sendln(fmt.Sprintf("cmdNo: [%s] len=%d matchAny=%v childMatchAny=%v", node.Path, len(strings.Fields(node.Path)), matchAny, childMatchAny))

	expanded, e := CmdExpand(arg, node.Path)
	if e != nil {
		c.Sendln(fmt.Sprintf("cmdNo: could not expand arg=[%s] cmd=[%s]: %v", arg, node.Path, e))
		return
	}

	var parentConf *ConfNode
	var childIndex int

	switch {
	case matchAny:
		// arg,node.Path is child: ... parent child value

		parentPath, childLabel := StripLastToken(expanded)
		parentPath, childLabel = StripLastToken(parentPath)

		parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
		if e != nil {
			c.Sendln(fmt.Sprintf("cmdNo: config parent node not found [%s]: %v", parentPath, e))
			return
		}

		childIndex = parentConf.FindChild(childLabel)

	case childMatchAny:
		// arg,node.Path is parent of single child: ... parent child value

		parentPath, childLabel := StripLastToken(expanded)

		parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
		if e != nil {
			c.Sendln(fmt.Sprintf("cmdNo: config parent node not found [%s]: %v", parentPath, e))
			return
		}

		childIndex = parentConf.FindChild(childLabel)

	default:
		// arg,node.Path is one of: intermediate node, leaf node, value of single-value leaf node, value of multi-value leaf node

		parentPath, childLabel := StripLastToken(expanded)

		parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
		if e != nil {
			c.Sendln(fmt.Sprintf("cmdNo: config parent node not found [%s]: %v", parentPath, e))
			return
		}

		childIndex = parentConf.FindChild(childLabel)

		_, cmdLast := StripLastToken(node.Path)
		if IsConfigValueKeyword(cmdLast) {
			if e2 := parentConf.ValueDelete(childLabel); e2 != nil {
				c.Sendln(fmt.Sprintf("cmdNo: could not delete value: %v", e2))
				return
			}

			if len(parentConf.Value) > 0 {
				return // done, can't delete node
			}

			// node without value

			parentPath, childLabel = StripLastToken(parentPath)

			parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
			if e != nil {
				c.Sendln(fmt.Sprintf("cmdNo: config parent node not found [%s]: %v", parentPath, e))
				return
			}

			childIndex = parentConf.FindChild(childLabel)
		}
	}

	c.Sendln(fmt.Sprintf("cmdNo: parent=[%s] childIndex=%d", parentConf.Path, childIndex))
	c.Sendln(fmt.Sprintf("cmdNo: parent=[%s] child=[%s]", parentConf.Path, parentConf.Children[childIndex].Path))

	ctx.ConfRootCandidate().Prune(parentConf, parentConf.Children[childIndex], c)
}
