package command

import (
	"fmt"
	"log"
	"strings"
)

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
