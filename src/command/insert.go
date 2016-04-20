package command

import (
	"fmt"
)

func pushCmdChild(node, child *CmdNode) {

	size := len(node.Children)
	if size < 1 {
		// first element is special
		// because both insert position=0 and current size=0
		node.Children = append(node.Children, child)
		return
	}

	// FIXME replace linear search with binary search
	newLabel := LastToken(child.Path)
	found := size
	for i, n := range node.Children {
		label := LastToken(n.Path)
		if newLabel < label {
			found = i // 0..size-1
			break
		}
	}

	if found == size {
		// not found - insert into last position can be optimized as append
		node.Children = append(node.Children, child)
		return
	}

	// insert
	node.Children = append(node.Children, nil)           // grow
	copy(node.Children[found+1:], node.Children[found:]) // shift
	node.Children[found] = child                         // insert
}

func pushConfChild(node, child *ConfNode) {

	size := len(node.Children)
	if size < 1 {
		// first element is special
		// because both insert position=0 and current size=0
		node.Children = append(node.Children, child)
		return
	}

	// FIXME replace linear search with binary search
	newLabel := LastToken(child.Path)
	found := size
	for i, n := range node.Children {
		label := LastToken(n.Path)
		if newLabel < label {
			found = i // 0..size-1
			break
		}
	}

	if found == size {
		// not found - insert into last position can be optimized as append
		node.Children = append(node.Children, child)
		return
	}

	// insert
	node.Children = append(node.Children, nil)           // grow
	copy(node.Children[found+1:], node.Children[found:]) // shift
	node.Children[found] = child                         // insert
}

func (n *ConfNode) deleteChildByIndex(i int) {
	// delete preserving order: a, a[len(a)-1] = append(a[:i], a[i+1:]...), nil
	last := len(n.Children) - 1
	n.Children, n.Children[last] = append(n.Children[:i], n.Children[i+1:]...), nil
}

func (n *ConfNode) deleteChildByLabel(label string) error {
	i := n.FindChild(label)
	if i < 0 {
		return fmt.Errorf("deleteChildByLabel: not found: path=[%s] label=[%s]", n.Path, label)
	}

	n.deleteChildByIndex(i)

	return nil
}

/*
func (n *ConfNode) ValueAdd(value string) error {
	newPath := fmt.Sprintf("%s %s", n.Path, value)
	newNode := &ConfNode{Path: newPath}
	pushConfChild(n, newNode)
	return nil
}

func (n *ConfNode) ValueDelete(value string) error {
	i := n.FindChild(value)
	if i < 0 {
		return fmt.Errorf("ConfNode.ValueDelete: value not found: path=[%s] value=[%s]", n.Path, value)
	}

	n.deleteChildByIndex(i)

	return nil
}
*/
