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

func (n *ConfNode) ValueAdd(value string) error {

	size := len(n.Value)
	if size < 1 {
		// first element is special
		// because both insert position=0 and current size=0
		n.Value = append(n.Value, value)
		return nil
	}

	// FIXME replace linear search with binary search
	found := size
	for i, v := range n.Value {
		if value == v {
			return nil // already exists
		}
		if value < v {
			found = i // 0..size-1
			break
		}
	}

	if found == size {
		// not found - insert into last position can be optimized as append
		n.Value = append(n.Value, value)
		return nil
	}

	// insert
	n.Value = append(n.Value, "")            // grow
	copy(n.Value[found+1:], n.Value[found:]) // shift
	n.Value[found] = value                   // insert

	return nil
}

func (n *ConfNode) ValueDelete(value string) error {
	i := n.ValueIndex(value)
	if i < 0 {
		return fmt.Errorf("ConfNode.ValueDelete: value not found: path=[%s] value=[%s]", n.Path, value)
	}

	// delete preserving order: a, a[len(a)-1] = append(a[:i], a[i+1:]...), nil
	last := len(n.Value) - 1
	n.Value, n.Value[last] = append(n.Value[:i], n.Value[i+1:]...), ""

	return nil
}
