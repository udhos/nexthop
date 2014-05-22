package main

import (
	//"bufio"
	"fmt"
	"log"
	"strings"
	"text/scanner"
)

type CmdFunc func(c *TelnetClient, line string)

type CmdNode struct {
	Path     string
	Desc     string
	MinLevel int
	Handler  CmdFunc
	Children []*CmdNode
}

func firstToken(path string) string {
	// fixme with tokenizer
	return strings.Fields(path)[0]
}

func lastToken(path string) string {
	// fixme with tokenizer
	f := strings.Fields(path)
	return f[len(f)-1]
}

func cmdInstall(root *CmdNode, path string, min int, cmd CmdFunc, desc string) {
	log.Printf("cmdInstall: [%s]", path)

	cmdFind(root, path, 0)
}

func cmdFind(root *CmdNode, path string, level int) (*CmdNode, error) {

	var s scanner.Scanner
	s.Error = func(s *scanner.Scanner, msg string) {
		log.Printf("command scan error: %s [%s]", msg, path)
	}
	s.Init(strings.NewReader(path))

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		log.Printf("cmdFind: token: [%s]", s.TokenText())
	}

	return nil, fmt.Errorf("fixme")
}
