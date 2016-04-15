package command

import (
	"fmt"
	//"strings"
)

type LineSender interface {
	Sendln(string) int
}

type configSender struct {
	sender LineSender
}

func (s *configSender) WriteLine(line string) (int, error) {
	return s.sender.Sendln(line), nil
}

func ShowConf(root *ConfNode, node *CmdNode, c CmdClient, treeMode, infoType bool) {

	sender := &configSender{c}

	for _, n := range root.Children {
		if treeMode {
			showConfTree(n, 0, c)
		} else {
			//showConfLine(n, c)
			WriteConfig(n, sender, infoType)
		}
	}
}

/*
func showConfLine(node *ConfNode, c CmdClient) {

	if len(node.Value) == 0 && len(node.Children) == 0 {
		c.Sendln(node.Path)
		return
	}

	for _, v := range node.Value {
		c.Sendln(fmt.Sprintf("%s %s", node.Path, v))
	}

	for _, n := range node.Children {
		showConfLine(n, c)
	}
}
*/

func showConfTree(node *ConfNode, depth int, c CmdClient) {

	label := LastToken(node.Path)
	c.Sendln(fmt.Sprintf("%*s%s", depth, "", label))

	newDepth := depth + 2

	/*
		for _, v := range node.Value {
			c.Sendln(fmt.Sprintf("%*s%s", newDepth, "", v))
		}
		if len(node.Value) > 0 {
			c.Sendln(fmt.Sprintf("%*s%s", depth, "", "exit"))
		}
	*/

	for _, n := range node.Children {
		showConfTree(n, newDepth, c)
	}
	if len(node.Children) > 0 {
		c.Sendln(fmt.Sprintf("%*s%s", depth, "", "exit"))
	}
}

/*
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
*/
