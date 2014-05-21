package main

import (
	"fmt"
)

type CmdFunc func(c *TelnetClient, line string)

type CmdNode struct {
	Path     string
	Desc     string
	MinLevel int
	Handler  CmdFunc
	Children []*CmdNode
}

func cmdInstall(root *CmdNode, path string, min int, cmd CmdFunc, desc string) {
}

func cmdFind(root *CmdNode, path string, level int) (*CmdNode, error) {
	return nil, fmt.Errorf("fixme")
}
