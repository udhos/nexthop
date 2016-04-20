package command

import (
	"fmt"
)

// path: parent command node path
// line: parent original user line
// value: last label of original user line
func SingleValueSet(ctx ConfContext, c LineSender, path, line, value string) {
	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		c.Sendln(fmt.Sprintf("SingleValueSet: error: %v", err))
		return
	}

	confNode.Children = nil
	newPath := fmt.Sprintf("%s %s", confNode.Path, value)
	newNode := &ConfNode{Path: newPath}
	pushConfChild(confNode, newNode)
}

/*
func SingleValueSetSimple(ctx ConfContext, c LineSender, nodePath, fullLine string) {
	line, value := StripLastToken(fullLine)
	path, _ := StripLastToken(nodePath)

	SingleValueSet(ctx, c, path, line, value)
}

func MultiValueAdd(ctx ConfContext, c LineSender, nodePath, fullLine string) {
	line, value := StripLastToken(fullLine)
	path, _ := StripLastToken(nodePath)

	confCand := ctx.ConfRootCandidate()
	confNode, err, _ := confCand.Set(path, line)
	if err != nil {
		c.Sendln(fmt.Sprintf("MultiValueAdd: error: %v", err))
		return
	}

	confNode.ValueAdd(value)
}
*/

func SetSimple(ctx ConfContext, c LineSender, nodePath, fullLine string) {
	_, err, _ := ctx.ConfRootCandidate().Set(nodePath, fullLine)
	if err != nil {
		c.Sendln(fmt.Sprintf("SetSimple: error: %v", err))
	}
}
