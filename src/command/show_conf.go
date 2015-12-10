package command

import (
	"fmt"
	"strings"
)

func ShowConf(root *ConfNode, node *CmdNode, c CmdClient, lineMode bool) {
	for _, n := range root.Children {
		if lineMode {
			showConfLine(n, 0, c)
		} else {
			showConf(n, 0, 1, c, false)
		}
	}
}

func showConfLine(node *ConfNode, depth int, c CmdClient) {

	if len(node.Value) == 0 && len(node.Children) == 0 {
		c.Sendln(node.Path)
		return
	}

	// show node values
	for _, v := range node.Value {
		c.Sendln(fmt.Sprintf("%s %s", node.Path, v))
	}

	for _, n := range node.Children {
		showConfLine(n, depth+1, c)
	}
}

func showConf(node *ConfNode, depth, valueDepth int, c CmdClient, hasSibling bool) {

	var ident string
	nodeIdent := strings.Repeat(" ", 2*depth)
	last := LastToken(node.Path)

	childrenCount := len(node.Children)
	valueCount := len(node.Value)
	fanout := childrenCount + valueCount

	// show path
	if hasSibling {
		ident = nodeIdent
	} else {
		ident = ""
	}

	if fanout == 0 {
		c.Sendln(fmt.Sprintf("%s%s", ident, last))
		return
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
