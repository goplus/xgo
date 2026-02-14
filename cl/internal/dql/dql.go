package dql

import "iter"

const (
	XGoPackage = true
)

type Node struct {
}

type NodeSet struct {
}

func New() NodeSet {
	return NodeSet{}
}

// NodeSet(seq func(func(Node) bool))
func NodeSet_Cast(func(yield func(*Node) bool)) NodeSet {
	return NodeSet{}
}

// XGo_Enum returns an iterator over the nodes in the NodeSet.
func (p NodeSet) XGo_Enum() iter.Seq[NodeSet] {
	return nil
}

// XGo_Any returns a NodeSet containing all descendant nodes with the specified name.
func (p NodeSet) XGo_Any(name string) NodeSet {
	return NodeSet{}
}

// XGo_Child returns a NodeSet containing all child nodes of the nodes in the NodeSet.
func (p NodeSet) XGo_Child() NodeSet {
	return NodeSet{}
}

// XGo_first returns the first node in the NodeSet, or an error if the NodeSet is empty.
func (p NodeSet) XGo_first() (*Node, error) {
	return nil, nil
}

// XGo_Elem returns a NodeSet containing the child nodes with the specified name.
func (p NodeSet) XGo_Elem(name string) NodeSet {
	return NodeSet{}
}

func (p NodeSet) XGo_Attr__0(name string) int {
	return 0
}

func (p NodeSet) XGo_Attr__1(name string) (int, error) {
	return 0, nil
}
