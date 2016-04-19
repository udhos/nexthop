package main

import (
	//"bufio"
	"fmt"
	"log"
	"strings"
	"text/scanner"
)

type CmdFunc func(root *CmdNode, c *TelnetClient, line string)

type CmdNode struct {
	Path     string
	Desc     string
	MinLevel int
	Handler  CmdFunc
	Children []*CmdNode
}

func lastToken(path string) string {
	// fixme with tokenizer
	f := strings.Fields(path)
	return f[len(f)-1]
}

/*
func dumpChildren(node *CmdNode) string {
	str := ""
	for _, p := range node.Children {
		str += fmt.Sprintf(",%s", p.Path)
	}
	return str
}
*/

func pushChild(node, child *CmdNode) {
	//log.Printf("pushChild: parent=[%s] child=[%s] before: [%v]", node.Path, child.Path, dumpChildren(node))
	node.Children = append(node.Children, child)
	//log.Printf("pushChild: parent=[%s] child=[%s] after: [%v]", node.Path, child.Path, dumpChildren(node))
}

func cmdInstall(root *CmdNode, path string, min int, cmd CmdFunc, desc string) {
	if err := cmdAdd(root, path, min, cmd, desc); err != nil {
		log.Printf("cmdInstall: error %s", err)
	}
}

func cmdAdd(root *CmdNode, path string, min int, cmd CmdFunc, desc string) error {
	//log.Printf("cmdInstall: [%s]", path)

	labelList := strings.Fields(path)
	size := len(labelList)
	parent := root
	for i, label := range labelList {
		currPath := strings.Join(labelList[:i+1], " ")
		//log.Printf("cmdInstall: %d: curr=[%s] label=[%s]", i, currPath, label)
		child := findChild(parent, label)
		if child != nil {
			// found, search next
			//log.Printf("cmdInstall: found [%s]", currPath)
			parent = child
			continue
		}

		// not found

		//log.Printf("cmdInstall: new [%s]", currPath)

		for ; i < size-1; i++ {
			// intermmediate label
			label = labelList[i]
			currPath = strings.Join(labelList[:i+1], " ")
			//log.Printf("cmdInstall: %d: intermmediate curr=[%s] label=[%s]", i, currPath, label)
			newNode := &CmdNode{Path: currPath, MinLevel: EXEC}
			pushChild(parent, newNode)
			parent = newNode
		}

		// last label
		label = labelList[size-1]
		//log.Printf("cmdInstall: %d: leaf curr=[%s] label=[%s]", i, path, label)
		newNode := &CmdNode{Path: path, Desc: desc, MinLevel: min, Handler: cmd}
		pushChild(parent, newNode)

		return nil
	}

	// command node found

	return fmt.Errorf("[%s] already exists", path)
}

func findChild(node *CmdNode, label string) *CmdNode {

	for _, c := range node.Children {
		last := lastToken(c.Path)
		//log.Printf("findChild: searching [%s] against (%s)[%s] under [%s]", label, last, c.Path, node.Path)
		if label == last {
			//log.Printf("findChild: found [%s] as [%s] under [%s]", label, c.Path, node.Path)
			return c
		}
	}

	//log.Printf("findChild: not found [%s] under [%s]", label, node.Path)

	return nil
}

func matchChildren(children []*CmdNode, label string) []*CmdNode {
	c := []*CmdNode{}

	for _, n := range children {
		last := lastToken(n.Path)
		if strings.HasPrefix(last, label) {
			c = append(c, n)
		}
	}

	return c
}

func cmdFind(root *CmdNode, path string, level int) (*CmdNode, error) {

	var s scanner.Scanner
	s.Error = func(s *scanner.Scanner, msg string) {
		log.Printf("command scan error: %s [%s]", msg, path)
	}
	s.Init(strings.NewReader(path))

	parent := root
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		//log.Printf("cmdFind: token: [%s]", s.TokenText())
		label := s.TokenText()
		children := matchChildren(parent.Children, label)
		size := len(children)
		if size < 1 {
			return nil, fmt.Errorf("cmdFind: not found: [%s] under [%s]", label, parent.Path)
		}
		if size > 1 {
			return nil, fmt.Errorf("cmdFind: ambiguous: [%s] under [%s]", label, parent.Path)
		}
		parent = children[0]
	}

	//log.Printf("cmdFind: found [%s] as [%s]", path, parent.Path)

	return parent, nil
}
