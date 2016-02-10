package command

import (
	//"bufio"
	"fmt"
	"log"
	"strings"
	//"text/scanner"
)

const NexthopVersion = "nexthop v0.0"

const NexthopCopyright = "The MIT License (MIT)\r\n" +
	"\r\n" +
	"Copyright (c) 2016 The Nexthop Authors\r\n" +
	"\r\n" +
	"Permission is hereby granted, free of charge, to any person obtaining a copy\r\n" +
	"of this software and associated documentation files (the \"Software\"), to deal\r\n" +
	"in the Software without restriction, including without limitation the rights\r\n" +
	"to use, copy, modify, merge, publish, distribute, sublicense, and/or sell\r\n" +
	"copies of the Software, and to permit persons to whom the Software is\r\n" +
	"furnished to do so, subject to the following conditions:\r\n" +
	"\r\n" +
	"The above copyright notice and this permission notice shall be included in all\r\n" +
	"copies or substantial portions of the Software.\r\n" +
	"\r\n" +
	"THE SOFTWARE IS PROVIDED \"AS IS\", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR\r\n" +
	"IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,\r\n" +
	"FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE\r\n" +
	"AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER\r\n" +
	"LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,\r\n" +
	"OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE\r\n" +
	"SOFTWARE.\r\n"

const (
	MOTD = iota
	USER = iota
	PASS = iota
	EXEC = iota
	ENAB = iota
	CONF = iota
)

const CMD_WILDCARD_ANY = "{ANY}"

const DefaultMaxConfigFiles = 1000

type CmdClient interface {
	ConfigPath() string
	ConfigPathSet(path string)
	Send(msg string)
	SendNow(msg string)
	Newline()
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
	LineBufferComplete(autoComplete string, attach bool)
}

type CmdFunc func(ctx ConfContext, node *CmdNode, line string, c CmdClient)
type CommitFunc func(ctx ConfContext, node *CmdNode, action CommitAction, c CmdClient) error

const (
	CMD_NONE = uint64(0 << 0)
	CMD_CONF = uint64(1 << 0) // command is a config tree navigation "path"
	//CMD_HELP = uint64(2 << 0) // show help/completion for command within config mode
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

/*
func (n *CmdNode) HelpUnderConfig() bool {
	return n.Options&CMD_HELP != 0
}
*/

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
			pushConfChild(parent, newNode)
			parent = newNode
		}

		// last label
		label = labels[size-1]
		newNode := &ConfNode{Path: expanded}
		pushConfChild(parent, newNode)

		return newNode, nil, false
	}

	// existing node found

	return parent, nil, true
}

func (n *ConfNode) Get(path string) (*ConfNode, error) {

	labels := strings.Fields(path)
	parent := n
	for _, label := range labels {
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

func LastToken(path string) string {
	_, last := StripLastToken(path)
	return last
}

func StripLastToken(path string) (string, string) {
	path = strings.TrimRight(path, " ")
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

func CmdInstall(root *CmdNode, opt uint64, path string, min int, cmd CmdFunc, apply CommitFunc, desc string) {
	if _, err := cmdAdd(root, opt, path, min, cmd, apply, desc); err != nil {
		log.Printf("cmdInstall: error %s", err)
	}
}

func cmdAdd(root *CmdNode, opt uint64, path string, min int, cmd CmdFunc, apply CommitFunc, desc string) (*CmdNode, error) {

	isConfig := ConfigNodeFlag(opt)

	if isConfig && apply == nil {
		return nil, fmt.Errorf("cmdAdd: [%s] configuration node missing commit func", path)
	}

	if !isConfig && apply != nil {
		return nil, fmt.Errorf("cmdAdd: [%s] non-configuration node does not use commit func", path)
	}

	labelList := strings.Fields(path)

	// report undefined pattern keywords
	for _, label := range labelList {
		if IsUserPatternKeyword(label) && findKeyword(label) == nil {
			// warning only
			log.Printf("cmdAdd: command [%s] using unknown keyword '%s'", path, label)
		}
	}

	size := len(labelList)
	parent := root
	for i, label := range labelList {
		currPath := strings.Join(labelList[:i+1], " ")

		//log.Printf("cmdAdd: [%s] [%s]", path, label)

		child := findChild(parent, label)
		if child != nil {
			// found, search next
			parent = child
			continue
		}

		// not found

		for ; i < size-1; i++ {
			// intermmediate label
			label = labelList[i]
			currPath = strings.Join(labelList[:i+1], " ")
			newNode := &CmdNode{Path: currPath, MinLevel: min, Options: opt}
			pushCmdChild(parent, newNode)
			parent = newNode
		}

		// last label
		label = labelList[size-1]
		newNode := &CmdNode{Path: path, Desc: desc, MinLevel: min, Handler: cmd, Apply: apply, Options: opt}
		pushCmdChild(parent, newNode)

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

func CmdFindRelative(root *CmdNode, line, configPath string, status int, checkPattern bool) (*CmdNode, string, error) {

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

	node, err := CmdFind(root, lookupPath, status, checkPattern)
	if err != nil {
		return nil, lookupPath, fmt.Errorf("CmdFindRelative: command not found: %s", err)
	}

	return node, lookupPath, nil
}

func CmdFind(root *CmdNode, path string, level int, checkPattern bool) (*CmdNode, error) {

	tokens := strings.Fields(path)

	parent := root
	for _, label := range tokens {

		/*
			if len(parent.Children) == 1 && LastToken(parent.Children[0].Path) == CMD_WILDCARD_ANY {
				// {ANY} is special construct for consuming anything
				return checkLevel(parent.Children[0], "CmdNode", path, level) // found
			}
		*/

		children, isAny, err := matchChildren(parent.Children, label, checkPattern)
		if err != nil {
			return nil, fmt.Errorf("CmdFind: bad command: [%s] under [%s]: %v", label, parent.Path, err)
		}
		if isAny {
			return checkLevel(children[0], "CmdFind", path, level) // found
		}

		size := len(children)
		if size < 1 {
			return nil, fmt.Errorf("CmdFind: not found: [%s] under [%s]", label, parent.Path)
		}
		if size > 1 {
			return nil, fmt.Errorf("CmdFind: ambiguous: [%s] under [%s]", label, parent.Path)
		}

		parent = children[0]
	}

	return checkLevel(parent, "CmdNode", path, level) // found
}

func matchChildren(children []*CmdNode, prefix string, checkPattern bool) ([]*CmdNode, bool, error) {

	if len(children) == 1 && LastToken(children[0].Path) == CMD_WILDCARD_ANY {
		// {ANY} is special construct for consuming anything
		return children, true, nil // found
	}

	c := []*CmdNode{}

	for _, n := range children {
		last := LastToken(n.Path)
		if IsUserPatternKeyword(last) {
			if checkPattern {
				if err := MatchKeyword(last, prefix); err != nil {
					return nil, false, err
				}
			}
			c = append(c, n)
			continue
		}
		if strings.HasPrefix(last, prefix) {
			c = append(c, n)
			continue
		}
	}

	return c, false, nil
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

func Dispatch(ctx ConfContext, rawLine string, c CmdClient, status int, history bool) error {

	if helpKey(ctx, rawLine, c, status) {
		return nil
	}

	line := strings.TrimLeft(rawLine, " ")

	if line == "" {
		return nil // ignore empty lines
	}

	if history {
		c.HistoryAdd(rawLine)
	}

	if line[0] == '!' || line[0] == '#' {
		return nil // ignore comments
	}

	const checkPattern = true
	node, lookupPath, err := CmdFindRelative(ctx.CmdRoot(), line, c.ConfigPath(), status, checkPattern)
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

func helpKey(ctx ConfContext, rawLine string, c CmdClient, status int) bool {
	size := len(rawLine)
	if size < 1 {
		return false
	}

	b := rawLine[size-1] // help key command

	line := rawLine[:len(rawLine)-1]
	lineSize := len(line)
	listChildren := lineSize < 1 || line[lineSize-1] == ' '
	// listChildren=true  -> search for children
	// listChildren=false -> search for siblings

	switch b {
	case '?': // question mark key
		c.Newline()
		helpKeyQuestion(ctx, line, c, status, listChildren)
		return true
	case byte(9): // tab key
		c.Newline()
		helpKeyTab(ctx, line, c, status, listChildren)
		return true
	}

	return false
}

func helpKeyQuestion(ctx ConfContext, line string, c CmdClient, status int, listChildren bool) {

	//log.Printf("helpKeyQuestion: listChildren=%v (true=children false=siblings)", listChildren)

	children := helpOptions(ctx, line, c, status, listChildren)

	siblingsPrefix := LastToken(line)

	options, help := expandOptions(children, listChildren, siblingsPrefix)

	showOptions(c, options, help)
}

func helpKeyTab(ctx ConfContext, line string, c CmdClient, status int, listChildren bool) {

	//log.Printf("helpKeyTab: listChildren=%v (true=children false=siblings)", listChildren)

	children := helpOptions(ctx, line, c, status, listChildren)

	siblingsPrefix := LastToken(line)

	options, help := expandOptions(children, listChildren, siblingsPrefix)

	//log.Printf("helpKeyTab: options=%v", options)

	if len(options) != 1 {

		// full auto-complete not possible - there is completion ambiguity

		if prefix := longestCommonPrefix(options); prefix != "" {
			// partial auto-complete
			c.LineBufferComplete(prefix, listChildren)
		}

		// behave like question mark key
		showOptions(c, options, help)
		return
	}

	// full auto-complete possible

	autoComplete := LastToken(options[0])

	if IsUserPatternKeyword(autoComplete) {
		// do not autocomplete with pattern

		// behave like question mark key
		showOptions(c, options, help)
		return
	}

	//c.Sendln(fmt.Sprintf("helpKeyTab: auto-complete='%s' FIXME WRITEME", autoComplete))

	c.LineBufferComplete(autoComplete+" ", listChildren)
}

func expandOptions(children []*CmdNode, listChildren bool, siblingsPrefix string) ([]string, []string) {
	var expanded, help []string

	for _, child := range children {

		label := LastToken(child.Path)

		options := []string{label} // assume it's not a {}-keywrod

		if IsUserPatternKeyword(label) {
			// but it's {}-keyword
			k := findKeyword(label)
			if k != nil && k.options != nil {
				// found function for listing completion options
				options = k.options()
			}
		}

		for _, opt := range options {

			if !listChildren {
				// we are looking for siblings
				if !strings.HasPrefix(opt, siblingsPrefix) {
					// this sibling does not match the required prefix
					continue
				}
			}

			expanded = append(expanded, opt)
			help = append(help, child.Desc)
		}
	}

	return expanded, help
}

func showOptions(c CmdClient, options, help []string) {

	for i, opt := range options {
		desc := help[i]
		if desc == "" {
			c.Sendln(fmt.Sprintf("%s", opt))
		} else {
			c.Sendln(fmt.Sprintf("%s - %s", opt, desc))
		}
	}
}

func helpOptions(ctx ConfContext, line string, c CmdClient, status int, listChildren bool) []*CmdNode {

	var children []*CmdNode

	const checkPattern = false

	if listChildren {

		// List children

		node, _, err := CmdFindRelative(ctx.CmdRoot(), line, c.ConfigPath(), status, checkPattern)
		if err != nil {
			c.Sendln(fmt.Sprintf("helpOptions: not found [%s]: %v", line, err))
			return nil
		}

		children = node.Children

	} else {

		// List ambiguous siblings

		parentPath, prefix := StripLastToken(line)

		//log.Printf("helpOptions: siblings path=[%s] parent=[%s] prefix=[%s]", line, parentPath, prefix)

		parent, _, err1 := CmdFindRelative(ctx.CmdRoot(), parentPath, c.ConfigPath(), status, checkPattern)
		if err1 != nil {
			c.Sendln(fmt.Sprintf("helpOptions: not found [%s]: %v", parentPath, err1))
			return nil
		}

		var err2 error
		children, _, err2 = matchChildren(parent.Children, prefix, checkPattern)
		if err2 != nil {
			c.Sendln(fmt.Sprintf("helpOptions: bad command: [%s] under [%s]: %v", prefix, parent.Path, err2))
			return nil
		}

	}

	var visible []*CmdNode

	for _, child := range children {
		if status < child.MinLevel {
			continue // Hide prohibited command
		}
		/*
			// FIXME???
			if status == CONF && !child.HelpUnderConfig() {
				continue // Hide command in config mode
			}
		*/
		visible = append(visible, child)

		//log.Printf("helpOptions: child[%d]=[%s]", i, LastToken(child.Path))
	}

	return visible
}

func DescInstall(root *CmdNode, path, desc string) {
	checkPattern := false
	node, err := CmdFind(root, path, CONF, checkPattern)
	if err != nil {
		log.Printf("DescInstall: not found [%s]: %v", path, err)
		return
	}
	if node.Desc != "" {
		log.Printf("DescInstall: description not empty [%s]: description=[%s]", path, node.Desc)
		return
	}
	node.Desc = desc
}

func MissingDescription(root *CmdNode) {
	for _, c := range root.Children {
		missDesc(c)
	}
}

func missDesc(node *CmdNode) {

	if node.Desc == "" {
		log.Printf("MissingDescription: [%s]", node.Path)
	}

	for _, c := range node.Children {
		missDesc(c)
	}
}
