/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package maps

import (
	"iter"

	"github.com/goplus/xgo/dql"
)

const (
	XGoPackage = true
)

// -----------------------------------------------------------------------------

// Node represents a named value in a DQL query tree.
type Node struct {
	Name  string
	Value any
}

// NodeSet represents a set of nodes.
type NodeSet struct {
	Data iter.Seq[Node]
	Err  error
}

// NodeSet(seq) casts a NodeSet from a sequence of nodes.
func NodeSet_Cast(seq iter.Seq[Node]) NodeSet {
	return NodeSet{Data: seq}
}

// Root creates a NodeSet containing the provided root node.
func Root(doc Node) NodeSet {
	return NodeSet{
		Data: func(yield func(Node) bool) {
			yield(doc)
		},
	}
}

// Nodes creates a NodeSet containing the provided nodes.
func Nodes(nodes ...Node) NodeSet {
	return NodeSet{
		Data: func(yield func(Node) bool) {
			for _, node := range nodes {
				if !yield(node) {
					break
				}
			}
		},
	}
}

// New creates a NodeSet containing a single node from the provided document.
// The document should be of type map[string]any or []any.
// If the document type is invalid, it panics.
func New(doc any) NodeSet {
	switch doc.(type) {
	case map[string]any, []any:
	default:
		panic("dql/maps.New: invalid document type, should be map[string]any or []any")
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			yield(Node{Name: "", Value: doc})
		},
	}
}

// Source creates a NodeSet from various types of sources:
// - map[string]any: creates a NodeSet containing the single provided node.
// - []any: creates a NodeSet containing the single provided node.
// - Node: creates a NodeSet containing the single provided node.
// - iter.Seq[Node]: directly uses the provided sequence of nodes.
// - NodeSet: returns the provided NodeSet as is.
// If the source type is unsupported, it panics.
func Source(r any) (ret NodeSet) {
	switch v := r.(type) {
	case map[string]any, []any:
		return New(v)
	case Node:
		return Root(v)
	case iter.Seq[Node]:
		return NodeSet{Data: v}
	case NodeSet:
		return v
	default:
		panic("dql/maps.Source: unsupported source type")
	}
}

// -----------------------------------------------------------------------------

// XGo_Enum returns an iterator over the nodes in the NodeSet.
func (p NodeSet) XGo_Enum() iter.Seq[NodeSet] {
	if p.Err != nil {
		return dql.NopIter[NodeSet]
	}
	return func(yield func(NodeSet) bool) {
		p.Data(func(node Node) bool {
			return yield(Root(node))
		})
	}
}

// XGo_Select returns a NodeSet containing the nodes with the specified name.
//   - @name
//   - @"element-name"
func (p NodeSet) XGo_Select(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			p.Data(func(node Node) bool {
				if node.Name == name {
					return yield(node)
				}
				return true
			})
		},
	}
}

// XGo_Elem returns a NodeSet containing the child nodes with the specified name.
//   - .name
//   - .“element-name”
func (p NodeSet) XGo_Elem(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			p.Data(func(node Node) bool {
				return yieldElem(node, name, yield)
			})
		},
	}
}

// yieldElem yields the child node with the specified name if it exists.
func yieldElem(node Node, name string, yield func(Node) bool) bool {
	if children, ok := node.Value.(map[string]any); ok {
		if v, ok := children[name]; ok {
			return yield(Node{Name: name, Value: v})
		}
	}
	return true
}

// XGo_Child returns a NodeSet containing all child nodes of the nodes in the NodeSet.
func (p NodeSet) XGo_Child() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			p.Data(func(node Node) bool {
				return yieldChildNodes(node, yield)
			})
		},
	}
}

// yieldChildNodes yields all child nodes of the given node.
func yieldChildNodes(node Node, yield func(Node) bool) bool {
	switch children := node.Value.(type) {
	case map[string]any:
		for k, v := range children {
			if !yield(Node{Name: k, Value: v}) {
				return false
			}
		}
	case []any:
		for _, v := range children {
			if !yield(Node{Name: "", Value: v}) {
				return false
			}
		}
	}
	return true
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// nodes themselves) with the specified name.
// If name is "", it returns all nodes.
//   - .**.name
//   - .**.“element-name”
//   - .**.*
func (p NodeSet) XGo_Any(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			p.Data(func(node Node) bool {
				return yieldAnyNodes(name, node, yield)
			})
		},
	}
}

// yieldAnyNodes yields all descendant nodes of the given node that match the
// specified name. If name is "", it yields all nodes.
func yieldAnyNodes(name string, node Node, yield func(Node) bool) bool {
	if name == "" || node.Name == name {
		if !yield(node) {
			return false
		}
	}
	switch children := node.Value.(type) {
	case map[string]any:
		for k, v := range children {
			if !yieldAnyNode(name, k, v, yield) {
				return false
			}
		}
	case []any:
		for _, v := range children {
			if !yieldAnyNode(name, "", v, yield) {
				return false
			}
		}
	}
	return true
}

// yieldAnyNode recursively traverses into v if it is a map[string]any or []any,
// looking for descendant nodes matching name.
func yieldAnyNode(name, k string, v any, yield func(Node) bool) bool {
	switch v.(type) {
	case map[string]any, []any:
		return yieldAnyNodes(name, Node{Name: k, Value: v}, yield)
	}
	return true
}

// -----------------------------------------------------------------------------

// _all returns a NodeSet containing all nodes.
// It's a cache operation for performance optimization when you need to traverse
// the nodes multiple times.
func (p NodeSet) XGo_all() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	nodes := dql.Collect(p.Data)
	return Nodes(nodes...)
}

// _one returns a NodeSet containing the first node.
// It's a performance optimization when you only need the first node (stop early).
func (p NodeSet) XGo_one() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	n, err := dql.First(p.Data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(n)
}

// _single returns a NodeSet containing the single node.
// If there are zero or more than one nodes, it returns an error.
// ErrNotFound or ErrMultipleResults is returned accordingly.
func (p NodeSet) XGo_single() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	n, err := dql.Single(p.Data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(n)
}

// -----------------------------------------------------------------------------

// _ok returns true if there is no error in the NodeSet.
func (p NodeSet) XGo_ok() bool {
	return p.Err == nil
}

// _first returns the first node in the NodeSet.
func (p NodeSet) XGo_first() (Node, error) {
	if p.Err != nil {
		return Node{}, p.Err
	}
	return dql.First(p.Data)
}

// _name returns the name of the first node in the NodeSet.
// empty string is returned if the NodeSet is empty or error occurs.
func (p NodeSet) XGo_name__0() string {
	val, _ := p.XGo_name__1()
	return val
}

// _name returns the name of the first node in the NodeSet.
// If the NodeSet is empty, it returns ErrNotFound.
func (p NodeSet) XGo_name__1() (ret string, err error) {
	node, err := p.XGo_first()
	if err == nil {
		ret = node.Name
	}
	return
}

// _value returns the value of the first node in the NodeSet.
// nil is returned if the NodeSet is empty or error occurs.
func (p NodeSet) XGo_value__0() any {
	val, _ := p.XGo_value__1()
	return val
}

// _value returns the value of the first node in the NodeSet.
// If the NodeSet is empty, it returns ErrNotFound.
func (p NodeSet) XGo_value__1() (ret any, err error) {
	node, err := p.XGo_first()
	if err == nil {
		ret = node.Value
	}
	return
}

// _hasAttr returns true if the first node in the NodeSet has the specified attribute.
// It returns false otherwise.
func (p NodeSet) XGo_hasAttr(name string) bool {
	node, err := p.XGo_first()
	if err == nil {
		switch children := node.Value.(type) {
		case map[string]any:
			_, ok := children[name]
			return ok
		}
	}
	return false
}

// XGo_Attr returns the value of the specified attribute from the first node in the
// NodeSet. It only retrieves the attribute from the first node.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr__0(name string) any {
	val, _ := p.XGo_Attr__1(name)
	return val
}

// XGo_Attr returns the value of the specified attribute from the first node in the
// NodeSet. It only retrieves the attribute from the first node.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr__1(name string) (val any, err error) {
	node, err := p.XGo_first()
	if err == nil {
		switch children := node.Value.(type) {
		case map[string]any:
			if v, ok := children[name]; ok {
				return v, nil
			}
		}
		err = dql.ErrNotFound
	}
	return
}

// -----------------------------------------------------------------------------
