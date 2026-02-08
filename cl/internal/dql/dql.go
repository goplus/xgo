package dql

import "iter"

type ValueSet struct {
}

// XGo_0 returns the first value in the ValueSet, or ErrNotFound if the set is empty.
func (p ValueSet) XGo_0() (val string, err error) {
	return
}

type Node struct {
}

type NodeSet struct {
}

func New() NodeSet {
	return NodeSet{}
}

// XGo_Enum returns an iterator over the nodes in the NodeSet.
func (p NodeSet) XGo_Enum() iter.Seq[*Node] {
	return nil
}

// XGo_Any returns a NodeSet containing all descendant nodes of the nodes in
// the NodeSet, including the nodes themselves.
func (p NodeSet) XGo_Any() NodeSet {
	return NodeSet{}
}

// XGo_Child returns a NodeSet containing all child nodes of the nodes in the NodeSet.
func (p NodeSet) XGo_Child() NodeSet {
	return NodeSet{}
}

// XGo_Node returns a NodeSet containing the child nodes with the specified name.
func (p NodeSet) XGo_Node(name string) NodeSet {
	return NodeSet{}
}

// XGo_Attr returns a ValueSet containing the values of the specified attribute
// for each node in the NodeSet. If a node does not have the specified attribute,
// the Value will contain ErrNotFound.
func (p NodeSet) XGo_Attr(name string) ValueSet {
	return ValueSet{}
}
