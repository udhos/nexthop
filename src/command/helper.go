package command

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

func InstallCommonHelpers(root *CmdNode) {

	cmdNone := CMD_NONE
	cmdConf := CMD_CONF
	//cmdHelp := CMD_HELP
	cmdHelp := cmdNone

	CmdInstall(root, cmdHelp, "commit", CONF, HelperCommit, nil, "Apply current candidate configuration")
	CmdInstall(root, cmdHelp, "commit force", CONF, cmdCommitForce, nil, "Force saving candidate configuration even if unchanged")
	CmdInstall(root, cmdNone, "configure", ENAB, cmdConfig, nil, "Enter configuration mode")
	CmdInstall(root, cmdNone, "enable", EXEC, cmdEnable, nil, "Enter privileged mode")
	CmdInstall(root, cmdHelp, "exit", EXEC, cmdExit, nil, "Exit current location")
	CmdInstall(root, cmdHelp, "list", EXEC, cmdList, nil, "List command tree")
	CmdInstall(root, cmdHelp, "list brief", EXEC, cmdList, nil, "List only nodes with attached handlers")
	CmdInstall(root, cmdHelp, "list description", EXEC, cmdList, nil, "List command tree showing descriptions")
	CmdInstall(root, cmdHelp, "no {ANY}", CONF, HelperNo, nil, "Remove this configuration item")
	CmdInstall(root, cmdHelp, "quit", EXEC, cmdQuit, nil, "Quit session")
	CmdInstall(root, cmdNone, "reload", ENAB, cmdReload, nil, "Reload")
	CmdInstall(root, cmdHelp, "rollback", CONF, cmdRollback, nil, "Reset candidate configuration from active configuration")
	CmdInstall(root, cmdHelp, "rollback {COMMITID}", CONF, cmdRollback, nil, "Reset candidate configuration from rollback configuration")
	CmdInstall(root, cmdHelp, "show configuration", EXEC, HelperShowConf, nil, "Show candidate configuration")
	CmdInstall(root, cmdHelp, "show configuration compare", EXEC, HelperShowCompare, nil, "Show differences between active and candidate configurations")
	CmdInstall(root, cmdHelp, "show configuration rollback", EXEC, cmdShowCommitList, nil, "Show list of saved configurations")
	CmdInstall(root, cmdHelp, "show configuration rollback {COMMITID}", EXEC, cmdShowCommit, nil, "Show saved configuration")
	CmdInstall(root, cmdHelp, "show configuration tree", EXEC, HelperShowConf, nil, "Show candidate configuration tree")
	CmdInstall(root, cmdHelp, "show configuration info", EXEC, cmdShowConfInfo, nil, "Show candidate configuration info type")
	CmdInstall(root, cmdHelp, "show history", EXEC, cmdShowHistory, nil, "Show command history")
	CmdInstall(root, cmdHelp, "show running-configuration", EXEC, cmdShowRun, nil, "Show active configuration")
	CmdInstall(root, cmdHelp, "show running-configuration tree", EXEC, cmdShowRun, nil, "Show active configuration tree")
	CmdInstall(root, cmdHelp, "show running-configuration info", EXEC, cmdShowRunInfo, nil, "Show active configuration info type")
	CmdInstall(root, cmdConf, "username {USERNAME} password {PASSWORD}", EXEC, cmdUsername, ApplyBogus, "User clear-text password")

	DescInstall(root, "no", "Remove a configuration item")
	DescInstall(root, "show", "Show configuration item")
	DescInstall(root, "username", "Configure user parameter")
	DescInstall(root, "username {USERNAME}", "Configure username")
	DescInstall(root, "username {USERNAME} password", "Set user password")
}

func ApplyBogus(ctx ConfContext, node *CmdNode, action CommitAction, c CmdClient) error {
	// do nothing
	return nil
}

func cmdCommitForce(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	doCommit(ctx, node, line, c, true)
}

func HelperCommit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	doCommit(ctx, node, line, c, false)
}

func doCommit(ctx ConfContext, node *CmdNode, line string, c CmdClient, force bool) {

	forceFailure := false

	if err := Commit(ctx, c, forceFailure); err != nil {
		msg := fmt.Sprintf("cmdCommit: commit failed: %v", err)
		log.Printf(msg)
		c.Sendln(msg)
		return
	}

	c.Sendln("cmdCommit: configuration changes commited")

	if ConfEqual(ctx.ConfRootActive(), ctx.ConfRootCandidate()) && !force {
		c.Sendln("cmdCommit: refusing to save unchanged configuration - consider 'commit force'")
	} else {
		path, err := SaveNewConfig(ctx.ConfigPathPrefix(), ctx.ConfRootCandidate(), ctx.MaxConfigFiles())
		if err != nil {
			msg := fmt.Sprintf("cmdCommit: unable to save new current configuration: %v", err)
			log.Printf(msg)
			c.Sendln(msg)
		} else {
			c.Sendln(fmt.Sprintf("cmdCommit: new configuration saved: [%s]", path))
		}
	}

	ConfActiveFromCandidate(ctx)

	c.Sendln("cmdCommit: active configuration updated")
}

func findDeleted(root1, root2 *ConfNode) ([]string, []*ConfNode) {
	pathList := []string{}
	nodeList := []*ConfNode{}
	searchDeletedNodes(root1, root2, &pathList, &nodeList)
	return pathList, nodeList
}

func searchDeletedNodes(n1, root2 *ConfNode, pathList *[]string, nodeList *[]*ConfNode) {

	if /*len(n1.Value) == 0 &&*/ len(n1.Children) == 0 {
		if _, err := root2.Get(n1.Path); err != nil {
			// not found
			*pathList = append(*pathList, n1.Path)
			*nodeList = append(*nodeList, n1)
		}
		return
	}

	/*
		if len(n1.Value) > 0 {
			searchDeletedValues(n1, root2, pathList, nodeList)
		}
	*/

	for _, i := range n1.Children {
		searchDeletedNodes(i, root2, pathList, nodeList)
	}
}

/*
func searchDeletedValues(n1, root2 *ConfNode, pathList *[]string, nodeList *[]*ConfNode) {
	n2, err := root2.Get(n1.Path)
	if err != nil {
		// not found
		for _, v := range n1.Value {
			*pathList = append(*pathList, fmt.Sprintf("%s %s", n1.Path, v))
			*nodeList = append(*nodeList, n1)
		}
		return
	}

	for _, v := range n1.Value {
		i := n2.ValueIndex(v)
		if i < 0 {
			*pathList = append(*pathList, fmt.Sprintf("%s %s", n1.Path, v))
			*nodeList = append(*nodeList, n1)
		}
	}
}
*/

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
		c.Sendln("     'rollback' to discard uncommited changes")
		c.Sendln("     'show configuration compare' to view uncommited changes")
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
	log.Printf("cmdReload: not implemented FIXME WRITEME")
}

func cmdRollback(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	fields := strings.Fields(line)
	if len(fields) == 1 {
		if ConfEqual(ctx.ConfRootActive(), ctx.ConfRootCandidate()) {
			c.Sendln("rollback: notice: there is no uncommited change to discard")
		}

		c.Sendln("rollback: restoring candidate configuration from active configuration")

		ConfCandidateFromActive(ctx)

		return
	}

	if !ConfEqual(ctx.ConfRootActive(), ctx.ConfRootCandidate()) {
		c.Sendln("rollback: refusing to load rollback config over uncommited changes")
		return
	}

	// clear candidate configuration because we will load entire new config over it
	ctx.SetCandidate(&ConfNode{})

	id := fields[1]

	//path := fmt.Sprintf("%s%s", ctx.ConfigPathPrefix(), id)
	path := getConfigPath(ctx.ConfigPathPrefix(), id)

	abortOnError := false
	goodLines, err := LoadConfig(ctx, path, c, abortOnError)
	if err != nil {
		c.Sendln(fmt.Sprintf("rollback: CAUTION: there was error loading config from: [%s]: %v", path, err))
	}

	if goodLines > 0 {
		c.Sendln(fmt.Sprintf("rollback: commit '%s' loaded from [%s] as candidate config", id, path))
		if ConfEqual(ctx.ConfRootActive(), ctx.ConfRootCandidate()) {
			c.Sendln(fmt.Sprintf("rollback: notice: loaded commit '%s' is identical to active configuration", id))
		}

		c.Sendln("rollback: use 'show configuration compare' to verify candidate changes")
		c.Sendln("rollback: use 'commit' to apply candidate changes")
		c.Sendln("rollback: use 'rollback' to discard candidate changes")
	}

}

func cmdShowCommitList(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	dirname, matches, err := ListConfig(ctx.ConfigPathPrefix(), true)
	if err != nil {
		c.Sendln(fmt.Sprintf("cound't retrieve list of configuration files: %v", err))
		return
	}

	c.Sendln(fmt.Sprintf("found %d configuration files:", len(matches)))

	for _, m := range matches {
		path := filepath.Join(dirname, m)
		c.Sendln(path)
	}
}

func cmdShowCommit(ctx ConfContext, node *CmdNode, line string, c CmdClient) {

	id := LastToken(line)

	c.Sendln(fmt.Sprintf("configuration file for commit id '%s':", id))

	path := getConfigPath(ctx.ConfigPathPrefix(), id)

	lineCount := 0

	consume := func(line string) error {
		c.Sendln(line)
		lineCount++
		return nil // no error
	}

	abortOnError := false
	err := scanConfigFile(consume, path, abortOnError)
	if err != nil {
		c.Sendln(fmt.Sprintf("error scaning config file '%s': %v", path, err))
	}
	c.Sendln(fmt.Sprintf("(found %d lines)", lineCount))
}

func HelperShowCompare(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln("difference from active to candidate:")

	confAct := ctx.ConfRootActive()
	confCand := ctx.ConfRootCandidate()

	pathList1, _ := findDeleted(confAct, confCand)
	for _, conf := range pathList1 {
		c.Sendln(fmt.Sprintf("no %s", conf))
	}

	pathList2, _ := findDeleted(confCand, confAct)
	for _, conf := range pathList2 {
		c.Sendln(conf)
	}
}

func HelperShowConf(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	showConfig(ctx.ConfRootCandidate(), node, line, c, "candidate configuration:")
}

func showConfig(root *ConfNode, node *CmdNode, line string, c CmdClient, head string) {
	fields := strings.Fields(line)
	treeMode := len(fields) > 2 && strings.HasPrefix("tree", fields[2])
	c.Sendln(head)
	ShowConf(root, node, c, treeMode, false)
}

func cmdShowHistory(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln("command history:")
	c.HistoryShow()
}

func cmdShowRun(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	showConfig(ctx.ConfRootActive(), node, line, c, "running configuration:")
}

func cmdShowRunInfo(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln("running configuration:")
	ShowConf(ctx.ConfRootActive(), node, c, false, true)
}

func cmdShowConfInfo(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	c.Sendln("candidate configuration:")
	ShowConf(ctx.ConfRootCandidate(), node, c, false, true)
}

// Iface addr config should not be a helper function,
// since it only applies to RIB daemon.
// However it is currently being used for helping in tests.
func HelperIfaceAddr(ctx ConfContext, node *CmdNode, line string, c CmdClient) {

	/*
		linePath, addr := StripLastToken(line)

		path, _ := StripLastToken(node.Path)

		confCand := ctx.ConfRootCandidate()
		confNode, err, _ := confCand.Set(path, linePath)
		if err != nil {
			c.Sendln(fmt.Sprintf("iface addr: error: %v", err))
			return
		}

		confNode.ValueAdd(addr)
	*/

	MultiValueAdd(ctx, c, node.Path, line)
}

func HelperDescription(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	// line: "interf  XXXX   descrip   YYY ZZZ WWW"
	//                                 ^^^^^^^^^^^

	ln := strings.TrimLeft(line, " ") // drop leading spaces

	// find 3rd space
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

	SingleValueSet(ctx, c, path, linePath, DescriptionEncode(desc))
}

func DescriptionEncode(desc string) string {
	return strings.Replace(desc, " ", "(_)", -1)
}

func DescriptionDecode(desc string) string {
	return strings.Replace(desc, "(_)", " ", -1)
}

func HelperHostname(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	/*
		line, host := StripLastToken(line)

		path, _ := StripLastToken(node.Path)

		confCand := ctx.ConfRootCandidate()
		confNode, err, _ := confCand.Set(path, line)
		if err != nil {
			log.Printf("hostname: error: %v", err)
			return
		}

		confNode.ValueSet(host)
	*/

	SingleValueSetSimple(ctx, c, node.Path, line)
}

func cmdList(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	var showDesc, handlerOnly bool

	f := strings.Fields(line)
	if len(f) > 1 {
		if strings.HasPrefix("brief", f[1]) {
			handlerOnly = true
		}
		if strings.HasPrefix("description", f[1]) {
			showDesc = true
		}
	}

	for _, n := range ctx.CmdRoot().Children {
		list(n, 0, c, showDesc, handlerOnly)
	}
}

func list(node *CmdNode, depth int, c CmdClient, showDesc, handlerOnly bool) {
	handler := "----"
	if node.Handler != nil {
		handler = "LEAF"
	}

	if node.Handler != nil || !handlerOnly {
		ident := strings.Repeat(" ", 2*depth)
		c.Send(fmt.Sprintf("%s %d %s%s", handler, node.MinLevel, ident, node.Path))
		if showDesc {
			c.Send(fmt.Sprintf(" [%s]", node.Desc))
		}
		c.Newline()
	}

	for _, n := range node.Children {
		list(n, depth+1, c, showDesc, handlerOnly)
	}
}

func HelperNo(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	if err := CmdNo(ctx, node, line, c); err != nil {
		c.Sendln(fmt.Sprintf("error: %v", err))
	}
}

func CmdNo(ctx ConfContext, node *CmdNode, line string, c CmdClient) error {
	sep := strings.IndexByte(line, ' ')
	if sep < 0 {
		return fmt.Errorf("cmdNo: missing argument: %v", line)
	}

	arg := line[sep:]

	status := c.Status()

	const checkPattern = true
	node, _, err := CmdFindRelative(ctx.CmdRoot(), arg, c.ConfigPath(), status, checkPattern)
	if err != nil {
		return fmt.Errorf("cmdNo: not found [%s]: %v", arg, err)
	}

	if !node.IsConfig() {
		return fmt.Errorf("cmdNo: not a configuration command: [%s]", arg)
	}

	matchAny := node.MatchAny()
	childMatchAny := !matchAny && len(node.Children) == 1 && node.Children[0].MatchAny()

	//c.SendlnNow(fmt.Sprintf("cmdNo: [%s] len=%d matchAny=%v childMatchAny=%v", node.Path, len(strings.Fields(node.Path)), matchAny, childMatchAny))

	expanded, e := CmdExpand(arg, node.Path)
	if e != nil {
		return fmt.Errorf("cmdNo: could not expand arg=[%s] cmd=[%s]: %v", arg, node.Path, e)
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
			return fmt.Errorf("cmdNo: config parent node not found [%s]: %v", parentPath, e)
		}

		childIndex = parentConf.FindChild(childLabel)

	case childMatchAny:
		// arg,node.Path is parent of single child: ... parent child value

		parentPath, childLabel := StripLastToken(expanded)

		parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
		if e != nil {
			return fmt.Errorf("cmdNo: config parent node not found [%s]: %v", parentPath, e)
		}

		childIndex = parentConf.FindChild(childLabel)

	default:
		// arg,node.Path is one of: intermediate node, leaf node, value of single-value leaf node, value of multi-value leaf node

		parentPath, childLabel := StripLastToken(expanded)

		//log.Printf("CmdNo: default: expanded=[%s] parent=[%s] child=[%s]", expanded, parentPath, childLabel)

		parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
		if e != nil {
			return fmt.Errorf("cmdNo: config parent node not found [%s]: %v", parentPath, e)
		}

		childIndex = parentConf.FindChild(childLabel)

		_, cmdLast := StripLastToken(node.Path)

		//c.SendlnNow(fmt.Sprintf("cmdNo: node.Path=%s cmdLast=%s children=%d childLabel=%s childIndex=%d", node.Path, cmdLast, len(node.Children), childLabel, childIndex))

		if IsUserPatternKeyword(cmdLast) && len(node.Children) == 0 {

			// {}-pattern and no children: try to remove value

			if e2 := parentConf.ValueDelete(childLabel); e2 != nil {
				return fmt.Errorf("cmdNo: could not delete value: %v", e2)
			}

			/*
				if len(parentConf.Value) > 0 {
					return nil // done, can't delete node
				}
			*/

			//c.SendlnNow(fmt.Sprintf("cmdNo: value deleted: %s", childLabel))

			// node without value

			parentPath, childLabel = StripLastToken(parentPath)

			parentConf, e = ctx.ConfRootCandidate().Get(parentPath)
			if e != nil {
				return fmt.Errorf("cmdNo: config parent node not found [%s]: %v", parentPath, e)
			}

			childIndex = parentConf.FindChild(childLabel)
		}
	}

	//c.SendlnNow(fmt.Sprintf("cmdNo: parent=[%v] childIndex=%d", parentConf, childIndex))
	//c.Sendln(fmt.Sprintf("cmdNo: parent=[%s] childIndex=%d", parentConf.Path, childIndex))
	//c.Sendln(fmt.Sprintf("cmdNo: parent=[%s] child=[%s]", parentConf.Path, parentConf.Children[childIndex].Path))

	ctx.ConfRootCandidate().Prune(ctx.CmdRoot(), parentConf, parentConf.Children[childIndex], c)

	return nil // ok
}

func HelperShowVersion(daemonName string, c CmdClient) {
	c.Sendln(fmt.Sprintf("%s daemon", daemonName))
	c.Sendln(NexthopVersion)
	c.Send(NexthopCopyright)
}

func cmdUsername(ctx ConfContext, node *CmdNode, line string, c CmdClient) {
	/*
		userLine, pass := StripLastToken(line)

		path, _ := StripLastToken(node.Path)

		confCand := ctx.ConfRootCandidate()
		confNode, err, _ := confCand.Set(path, userLine)
		if err != nil {
			c.Sendln(fmt.Sprintf("unable to set user password: error: %v", err))
			return
		}

		confNode.ValueSet(pass)
	*/

	SingleValueSetSimple(ctx, c, node.Path, line)
}
