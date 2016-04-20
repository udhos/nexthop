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

func showConfTree(node *ConfNode, depth int, c CmdClient) {

	label := LastToken(node.Path)
	c.Sendln(fmt.Sprintf("%*s%s", depth, "", label))

	newDepth := depth + 2

	for _, n := range node.Children {
		showConfTree(n, newDepth, c)
	}
	if len(node.Children) > 0 {
		c.Sendln(fmt.Sprintf("%*s%s", depth, "", "exit"))
	}
}
