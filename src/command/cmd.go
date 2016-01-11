package command

import (
	//"bufio"
	"fmt"
	"log"
	"strings"
	//"text/scanner"
)

const NexthopVersion = "nexthop version 0.0"

const (
	MOTD = iota
	USER = iota
	PASS = iota
	EXEC = iota
	ENAB = iota
	CONF = iota
)

const CMD_WILDCARD_ANY = "{ANY}"

type CmdClient interface {
	ConfigPath() string
	ConfigPathSet(path string)
	Send(msg string)
	SendNow(msg string)
	Sendln(msg string)
	SendlnNow(msg string)
	InputQuit()
	Output() chan<- string
	Status() int
	StatusConf()
	StatusEnable()
	StatusExit()
	HistoryAdd(cmd string)
	HistoryShow()
}

type CmdFunc func(ctx ConfContext, node *CmdNode, line string, c CmdClient)
type CommitFunc func(ctx ConfContext, node *CmdNode, action CommitAction, c CmdClient) error

const (
	CMD_NONE = uint64(0)
	CMD_CONF = uint64(1 << 0)
)

type CmdNode struct {
	Path     string
	Desc     string
	MinLevel int
	Handler  CmdFunc
	Apply    CommitFunc
	Children []*CmdNode
	Options  uint64
}

func (n *CmdNode) IsConfig() bool {
	return ConfigNodeFlag(n.Options)
}

func ConfigNodeFlag(options uint64) bool {
	return options&CMD_CONF != 0
}

func (n *CmdNode) MatchAny() bool {
	last := strings.LastIndexByte(n.Path, ' ')
	if last < 0 {
		return false
	}
	if last+1 >= len(n.Path) {
		return false
	}
	return n.Path[last+1:] == CMD_WILDCARD_ANY
}

type ConfNode struct {
	Path     string
	Value    []string
	Children []*ConfNode
}

func (n *ConfNode) Clone() *ConfNode {
	newNode := &ConfNode{Path: n.Path}

	// clone values
	/*
		for _, v := range n.Value {
			newNode.Value = append(newNode.Value, v)
		}
	*/
	/*
		newNode.Value = make([]string, len(n.Value))
		copy(newNode.Value, n.Value)
	*/
	newNode.Value = append([]string{}, n.Value...)

	// clone children
	newNode.Children = make([]*ConfNode, len(n.Children))
	for i, node := range n.Children {
		newNode.Children[i] = node.Clone()
	}

	return newNode
}

func (n *ConfNode) ValueAdd(value string) error {
	i := n.ValueIndex(value)
	if i >= 0 {
		return nil // already exists
	}
	n.Value = append(n.Value, value) // append new
	return nil
}

func (n *ConfNode) ValueIndex(value string) int {
	for i, v := range n.Value {
		if v == value {
			return i
		}
	}
	return -1
}

func (n *ConfNode) ValueSet(value string) {
	n.Value = []string{value}
}

func (n *ConfNode) ValueDelete(value string) error {
	i := n.ValueIndex(value)
	if i < 0 {
		return fmt.Errorf("ConfNode.ValueDelete: value not found: path=[%s] value=[%s]", n.Path, value)
	}

	size := len(n.Value)
	last := size - 1
	n.Value[i] = n.Value[last]
	n.Value = n.Value[:last]

	return nil
}

// remove node from tree.
// any node which loses all children is purged as well.
func (n *ConfNode) Prune(parent, child *ConfNode, out CmdClient) bool {

	if n == parent {
		// found parent node
		if err := n.deleteChild(child); err != nil {
			msg := fmt.Sprintf("command.Prune: error: %v", err)
			log.Printf(msg)
			out.Sendln(msg)
		}
		deleteMe := len(n.Children) == 0 // lost all children, kill me
		return deleteMe
	}

	// keep searching parent node...
	for _, c := range n.Children {
		// ...recursively
		if deleteChild := c.Prune(parent, child, out); deleteChild {

			// child lost all children, then we kill it
			if err := n.deleteChild(c); err != nil {
				msg := fmt.Sprintf("command.Prune: error: %v", err)
				log.Printf(msg)
				out.Sendln(msg)
			}
			deleteMe := len(n.Children) == 0 // lost all children, kill me

			if size := len(n.Value); deleteMe && size > 0 {
				msg := fmt.Sprintf("command.Prune: error: child=[%s] valueCount=%d: should not delete node with value", n.Path, size)
				log.Printf(msg)
				out.Sendln(msg)
				return false
			}

			return deleteMe
		}
	}

	return false
}

func (n *ConfNode) deleteChild(child *ConfNode) error {
	for i, c := range n.Children {
		if c == child {
			n.deleteChildByIndex(i)
			return nil
		}
	}

	return fmt.Errorf("command.deleteChild: child not found: node=[%s] child[%s]", n.Path, child.Path)
}

// remove child node unconditionally
func (n *ConfNode) deleteChildByIndex(i int) {
	size := len(n.Children)
	last := size - 1
	n.Children[i] = n.Children[last]
	n.Children = n.Children[:last]
}

func (n *ConfNode) Set(path, line string) (*ConfNode, error, bool) {

	expanded, err := CmdExpand(line, path)
	if err != nil {
		return nil, fmt.Errorf("ConfNode.Set error: %v", err), false
	}

	//log.Printf("ConfNode.Set: line=[%v] path=[%v] expand=[%s]", line, path, expanded)

	labels := strings.Fields(expanded)
	size := len(labels)
	parent := n
	for i, label := range labels {
		child := parent.FindChild(label)
		if child >= 0 {
			// found, search next
			parent = parent.Children[child]
			continue
		}

		// not found

		for ; i < size-1; i++ {
			// intermmediate label
			label = labels[i]
			currPath := strings.Join(labels[:i+1], " ")
			newNode := &ConfNode{Path: currPath}
			parent.Children = append(parent.Children, newNode)
			parent = newNode
		}

		// last label
		label = labels[size-1]
		newNode := &ConfNode{Path: expanded}
		parent.Children = append(parent.Children, newNode)

		return newNode, nil, false
	}

	// existing node found

	return parent, nil, true
}

/*
func (n *ConfNode) GetParent(childPath string) (*ConfNode, error) {

	childFields := strings.Fields(childPath)
	parentFields := childFields[:len(childFields)-1]
	parentPath := strings.Join(parentFields, " ")

	node, err := n.Get(parentPath)
	if err != nil {
		return node, fmt.Errorf("ConfNode.GetParent: not found child=[%s] parent=[%s]: %v", childPath, parentPath, err)
	}

	return node, nil
}
*/

func (n *ConfNode) Get(path string) (*ConfNode, error) {

	/*
		if strings.TrimSpace(path) == "" {
			panic("command.Get: bad path")
		}
	*/

	labels := strings.Fields(path)
	parent := n
	for _, label := range labels {
		//log.Printf("ConfNode.Get: search label=[%s] under parent=[%s]", label, parent.Path)
		child := parent.FindChild(label)
		if child >= 0 {
			// found, search next
			parent = parent.Children[child]
			continue
		}

		// not found
		return nil, fmt.Errorf("ConfNode.Get: not found: path=[%s] parent=[%s] label=[%s]", path, parent.Path, label)
	}

	if path != parent.Path {
		err := fmt.Errorf("ConfNode.Get: want=[%s] found=[%s]", path, parent.Path)
		log.Print(err)
		panic(err)
		//return nil, err
	}

	return parent, nil // found
}

func (n *ConfNode) FindChild(label string) int {

	for i, c := range n.Children {
		last := LastToken(c.Path)
		if label == last {
			return i
		}
	}

	return -1
}

type ConfContext interface {
	CmdRoot() *CmdNode
	ConfRootCandidate() *ConfNode
	ConfRootActive() *ConfNode
	SetCandidate(newCand *ConfNode)
	SetActive(newActive *ConfNode)
	ConfigPathPrefix() string
	MaxConfigFiles() int
}

func ConfActiveFromCandidate(ctx ConfContext) {
	log.Printf("cloning configuration from candidate to active")
	ctx.SetActive(ctx.ConfRootCandidate().Clone())
}

func ConfCandidateFromActive(ctx ConfContext) {
	log.Printf("restoring candidate configuration from active")
	ctx.SetCandidate(ctx.ConfRootActive().Clone())
}

/*
func firstToken(path string) string {
	// fixme with tokenizer
	return strings.Fields(path)[0]
}
*/

func LastToken(path string) string {
	// fixme with tokenizer
	f := strings.Fields(path)
	return f[len(f)-1]
}

func StripLastToken(path string) (string, string) {
	last := strings.LastIndexByte(path, ' ')
	if last < 0 {
		return "", path
	}
	return path[:last], path[last+1:]
}

func dumpChildren(node *CmdNode) string {
	str := ""
	for _, p := range node.Children {
		str += fmt.Sprintf(",%s", p.Path)
	}
	return str
}

func pushChild(node, child *CmdNode) {
	//log.Printf("pushChild: parent=[%s] child=[%s] before: [%v]", node.Path, child.Path, dumpChildren(node))
	node.Children = append(node.Children, child)
	//log.Printf("pushChild: parent=[%s] child=[%s] after: [%v]", node.Path, child.Path, dumpChildren(node))
}

func CmdInstall(root *CmdNode, opt uint64, path string, min int, cmd CmdFunc, apply CommitFunc, desc string) {
	if _, err := cmdAdd(root, opt, path, min, cmd, apply, desc); err != nil {
		log.Printf("cmdInstall: error %s", err)
	}
}

func cmdAdd(root *CmdNode, opt uint64, path string, min int, cmd CmdFunc, apply CommitFunc, desc string) (*CmdNode, error) {
	//log.Printf("cmdInstall: [%s]", path)

	isConfig := ConfigNodeFlag(opt)

	if isConfig && apply == nil {
		return nil, fmt.Errorf("cmdAdd: [%s] configuration node missing commit func", path)
	}

	if !isConfig && apply != nil {
		return nil, fmt.Errorf("cmdAdd: [%s] non-configuration node does not use commit func", path)
	}

	labelList := strings.Fields(path)
	size := len(labelList)
	parent := root
	for i, label := range labelList {
		currPath := strings.Join(labelList[:i+1], " ")
		//log.Printf("cmdInstall: %d: curr=[%s] label=[%s]", i, currPath, label)

		if IsUserPatternKeyword(label) && findKeyword(label) == nil {
			// warning only
			log.Printf("cmdAdd: command [%s] using unknown keyword '%s'", path, label)
		}

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
			newNode := &CmdNode{Path: currPath, MinLevel: min, Options: opt}
			pushChild(parent, newNode)
			parent = newNode
		}

		// last label
		label = labelList[size-1]
		//log.Printf("cmdInstall: %d: leaf curr=[%s] label=[%s]", i, path, label)
		newNode := &CmdNode{Path: path, Desc: desc, MinLevel: min, Handler: cmd, Apply: apply, Options: opt}
		pushChild(parent, newNode)

		// did this command create an unreachable location?

		n, err := CmdFind(root, path, CONF, false)
		if err != nil {
			return newNode, fmt.Errorf("root=[%s] cmd=[%s] created unreachable command node: %v", root.Path, path, err)
		}

		if n != newNode {
			return newNode, fmt.Errorf("root=[%s] cmd=[%s] created wrong command node: %v", root.Path, path, err)
		}

		return newNode, nil
	}

	// command node found

	return parent, fmt.Errorf("cmdAdd: [%s] already exists", path)
}

func findChild(node *CmdNode, label string) *CmdNode {

	for _, c := range node.Children {
		last := LastToken(c.Path)
		if label == last {
			return c
		}
	}

	return nil
}

func CmdFindRelative(root *CmdNode, line, configPath string, status int) (*CmdNode, string, error) {

	prependConfigPath := true // assume it's a config cmd
	n, e := CmdFind(root, line, status, true)
	if e == nil {
		// found at root
		if !n.IsConfig() {
			// not a config cmd -- ignore prepend path
			prependConfigPath = false
		}
	}

	var lookupPath string
	if prependConfigPath && configPath != "" {
		// prepend path to config command
		lookupPath = fmt.Sprintf("%s %s", configPath, line)
	} else {
		lookupPath = line
	}

	node, err := CmdFind(root, lookupPath, status, true)
	if err != nil {
		return nil, lookupPath, fmt.Errorf("CmdFindRelative: command not found: %s", err)
	}

	return node, lookupPath, nil
}

func CmdFind(root *CmdNode, path string, level int, checkPattern bool) (*CmdNode, error) {

	tokens := strings.Fields(path)

	parent := root
	for _, label := range tokens {

		if len(parent.Children) == 1 && LastToken(parent.Children[0].Path) == CMD_WILDCARD_ANY {
			// {ANY} is special construct for consuming anything
			return checkLevel(parent.Children[0], "CmdNode", path, level) // found
		}

		children, err := matchChildren(parent.Children, label, checkPattern)
		if err != nil {
			return nil, fmt.Errorf("CmdFind: bad command: [%s] under [%s]: %v", label, parent.Path, err)
		}
		size := len(children)
		if size < 1 {
			return nil, fmt.Errorf("CmdFind: not found: [%s] under [%s]", label, parent.Path)
		}
		if size > 1 {
			return nil, fmt.Errorf("CmdFind: ambiguous: [%s] under [%s]", label, parent.Path)
		}

		//log.Printf("CmdFind: full=[%s] label=[%s] OK", path, label)

		parent = children[0]
	}

	return checkLevel(parent, "CmdNode", path, level) // found
}

func matchChildren(children []*CmdNode, label string, checkPattern bool) ([]*CmdNode, error) {
	c := []*CmdNode{}

	for _, n := range children {
		last := LastToken(n.Path)
		if IsUserPatternKeyword(last) {
			if checkPattern {
				if err := MatchKeyword(last, label); err != nil {
					return nil, err
				}
			}
			c = append(c, n)
			continue
		}
		if strings.HasPrefix(last, label) {
			c = append(c, n)
			continue
		}
	}

	return c, nil
}

func checkLevel(node *CmdNode, caller, path string, level int) (*CmdNode, error) {
	if node.MinLevel > level {
		return nil, fmt.Errorf("%s: command level prohibited: [%s]", caller, path)
	}

	return node, nil // found
}

// originalLine:    int eth0 ipv4 addr 1.1.1.1/30
// commandFullPath: interface {IFNAME} ipv4 address {ADDR}
// output:          interface eth0 ipv4 address 1.1.1.1/30
func CmdExpand(originalLine, commandFullPath string) (string, error) {
	lineFields := strings.Fields(originalLine)
	pathFields := strings.Fields(commandFullPath)

	lineLen := len(lineFields)
	pathLen := len(pathFields)

	if len(lineFields) != len(pathFields) {
		return "", fmt.Errorf("cmdExpand: length mismatch: line=%d path=%d", lineLen, pathLen)
	}

	for i, label := range pathFields {
		if IsUserPatternKeyword(label) {
			pathFields[i] = lineFields[i]
			continue
		}
	}

	return strings.Join(pathFields, " "), nil
}

func Dispatch(ctx ConfContext, rawLine string, c CmdClient, status int) error {

	line := strings.TrimLeft(rawLine, " ")

	if line == "" || line[0] == '!' || line[0] == '#' {
		return nil // ignore empty lines
	}

	c.HistoryAdd(rawLine)

	node, lookupPath, err := CmdFindRelative(ctx.CmdRoot(), line, c.ConfigPath(), status)
	if err != nil {
		e := fmt.Errorf("dispatchCommand: not found [%s]: %v", line, err)
		return e
	}

	if node.Handler == nil {
		if node.IsConfig() {
			c.ConfigPathSet(lookupPath) // enter config path
			return nil
		}
		err := fmt.Errorf("dispatchCommand: command missing handler: [%s]", line)
		return err
	}

	node.Handler(ctx, node, lookupPath, c)

	return nil
}
