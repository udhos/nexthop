package command

import (
	"fmt"
	"log"
	"strings"
)

func InstallCommonHelpers(root *CmdNode) {

	cmdNone := CMD_NONE
	//cmdConf := CMD_CONF

	CmdInstall(root, cmdNone, "commit", CONF, cmdCommit, nil, "Apply current candidate configuration")
	CmdInstall(root, cmdNone, "configure", ENAB, cmdConfig, nil, "Enter configuration mode")
	CmdInstall(root, cmdNone, "enable", EXEC, cmdEnable, nil, "Enter privileged mode")
	CmdInstall(root, cmdNone, "exit", EXEC, cmdExit, nil, "Exit current location")
	CmdInstall(root, cmdNone, "list", EXEC, cmdList, nil, "List command tree")
	CmdInstall(root, cmdNone, "no {ANY}", CONF, HelperNo, nil, "Remove a configuration item")
	CmdInstall(root, cmdNone, "quit", EXEC, cmdQuit, nil, "Quit session")
	CmdInstall(root, cmdNone, "reload", ENAB, cmdReload, nil, "Reload")
	CmdInstall(root, cmdNone, "rollback", CONF, cmdRollback, nil, "Reset candidate configuration from active configuration")
	CmdInstall(root, cmdNone, "rollback {ID}", CONF, cmdRollback, nil, "Reset candidate configuration from rollback configuration")
	CmdInstall(root, cmdNone, "show configuration", EXEC, cmdShowConf, nil, "Show candidate configuration")
	CmdInstall(root, cmdNone, "show configuration compare", EXEC, cmdShowCompare, nil, "Show differences between active and candidate configurations")
	CmdInstall(root, cmdNone, "show configuration tree", EXEC, cmdShowConf, nil, "Show candidate configuration tree")
	CmdInstall(root, cmdNone, "show history", EXEC, cmdShowHistory, nil, "Show command history")
	CmdInstall(root, cmdNone, "show running-configuration", EXEC, cmdShowRun, nil, "Show active configuration")
	CmdInstall(root, cmdNone, "show running-configuration tree", EXEC, cmdShowRun, nil, "Show active configuration tree")
}

func ApplyBogus(ctx ConfContext, node *CmdNode, enable bool, c CmdClient) error {
	return nil
}

func cmdCommit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {

	if err := Commit(ctx, c, true); err != nil {
		msg := fmt.Sprintf("cmdCommit: commit failed: %v", err)
		log.Printf(msg)
		c.Sendln(msg)
		return
	}

	if err := SaveNewConfig(ctx.ConfigPathPrefix(), ctx.ConfRootCandidate()); err != nil {
		msg := fmt.Sprintf("cmdCommit: unable to save new current configuration: %v", err)
		log.Printf(msg)
		c.Sendln(msg)
	}

	SwitchConf(ctx)
}

func SwitchConf(ctx ConfContext) {
	log.Printf("SwitchConf: cloning configuration from candidate to active")
	ctx.SetActive(ctx.ConfRootCandidate().Clone())
}

func findDeleted(root1, root2 *ConfNode) []string {
	list := []string{}
	searchDeletedNodes(root1, root2, &list)
	return list
}

func searchDeletedNodes(n1, root2 *ConfNode, list *[]string) {
	//log.Printf("searchDeletedNodes: [%s]", n1.Path)

	if len(n1.Children) > 0 {
		for _, i := range n1.Children {
			searchDeletedNodes(i, root2, list)
		}
		return
	}
	if len(n1.Value) > 0 {
		searchDeletedValues(n1, root2, list)
		return
	}

	if _, err := root2.Get(n1.Path); err != nil {
		// not found
		*list = append(*list, n1.Path)
	}
}

func searchDeletedValues(n1, root2 *ConfNode, list *[]string) {
	n2, err := root2.Get(n1.Path)
	if err != nil {
		// not found
		for _, v := range n1.Value {
			*list = append(*list, fmt.Sprintf("%s %s", n1.Path, v))
		}
		return
	}

	for _, v := range n1.Value {
		i := n2.ValueIndex(v)
		if i < 0 {
			*list = append(*list, fmt.Sprintf("%s %s", n1.Path, v))
		}
	}
}

func cmdConfig(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	status := c.Status()
	if status < CONF {
		c.StatusConf()
		reportUncommitedChanges(ctx, c)
	}
}

func cmdEnable(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	status := c.Status()
	if status < ENAB {
		c.StatusEnable()
	}
}

func reportUncommitedChanges(ctx ConfContext, c CmdClient) {
	confChanged := !ConfEqual(ctx.ConfRootActive(), ctx.ConfRootCandidate())
	if confChanged {
		c.Sendln("candidate configuration has uncommited changes")
		c.Sendln("use: 'commit' to apply changes")
		c.Sendln("     'rollback' to discard changes")
		c.Sendln("     'show configuration compare' to see uncommited changes")
	}
}

func cmdExit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {

	path := c.ConfigPath()
	if path == "" {
		status := c.Status()
		if status <= EXEC {
			c.Sendln("use 'quit' to exit remote terminal")
			return
		}
		if status == CONF {
			reportUncommitedChanges(ctx, c)
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

func cmdShowCompare(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln("difference from active to candidate:")

	confAct := ctx.ConfRootActive()
	confCand := ctx.ConfRootCandidate()

	cmdList1 := findDeleted(confAct, confCand)
	for _, conf := range cmdList1 {
		c.Sendln(fmt.Sprintf("no %s", conf))
	}

	cmdList2 := findDeleted(confCand, confAct)
	for _, conf := range cmdList2 {
		c.Sendln(conf)
	}
}

func cmdShowConf(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	showConfig(ctx.ConfRootCandidate(), node, line, c, "candidate configuration:")
}

func showConfig(root *ConfNode, node *CmdNode, line string, c CmdClient, head string) {
	fields := strings.Fields(line)
	treeMode := len(fields) > 2 && strings.HasPrefix("tree", fields[2])
	c.Sendln(head)
	ShowConf(root, node, c, treeMode)
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
	//c.Sendln(fmt.Sprintf("cmdNo: [%s]", line))

	sep := strings.IndexByte(line, ' ')
	if sep < 0 {
		c.Sendln(fmt.Sprintf("cmdNo: missing argument: %v", line))
		return
	}

	arg := line[sep:]

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

	//c.Sendln(fmt.Sprintf("cmdNo: [%s] len=%d matchAny=%v childMatchAny=%v", node.Path, len(strings.Fields(node.Path)), matchAny, childMatchAny))

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
