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

// -----------------------------------------------------------------------------

// Node represents a map[string]any node.
type Node struct {
	Name     string
	Children map[string]any
}

// NodeSet represents a set of map[string]any nodes.
type NodeSet struct {
	Data iter.Seq[Node]
	Err  error
}

// Root creates a NodeSet containing the provided root node.
func Root(doc Node) NodeSet {
	return NodeSet{
		Data: func(yield func(Node) bool) {
			yield(doc)
		},
	}
}

// New creates a NodeSet containing a single node from the provided map.
func New(doc map[string]any) NodeSet {
	return NodeSet{
		Data: func(yield func(Node) bool) {
			yield(Node{Name: "", Children: doc})
		},
	}
}

// Source creates a NodeSet from various types of sources:
// - map[string]any: creates a NodeSet containing the single provided node.
// - Node: creates a NodeSet containing the single provided node.
// - iter.Seq[Node]: directly uses the provided sequence of nodes.
// - NodeSet: returns the provided NodeSet as is.
// If the source type is unsupported, it panics.
func Source(r any) (ret NodeSet) {
	switch v := r.(type) {
	case map[string]any:
		return New(v)
	case Node:
		return Root(v)
	case iter.Seq[Node]:
		return NodeSet{Data: v}
	case NodeSet:
		return v
	default:
		panic("dql/maps.Source: unsupport source type")
	}
}

// -----------------------------------------------------------------------------

// XGo_Node returns the first node in the NodeSet.
func (p NodeSet) XGo_Node() (ret Node, err error) {
	if p.Err != nil {
		err = p.Err
		return
	}
	return dql.First(p.Data)
}

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
				return yieldNode(node, name, yield)
			})
		},
	}
}

// yieldNode yields the child node with the specified name if it exists.
func yieldNode(node Node, name string, yield func(Node) bool) bool {
	if v, ok := node.Children[name].(map[string]any); ok {
		return yield(Node{Name: name, Children: v})
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
				return rangeChildNodes(node, yield)
			})
		},
	}
}

// rangeChildNodes yields all child nodes of the given node.
func rangeChildNodes(node Node, yield func(Node) bool) bool {
	for k, v := range node.Children {
		if child, ok := v.(map[string]any); ok {
			if !yield(Node{Name: k, Children: child}) {
				return false
			}
		}
	}
	return true
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// nodes themselves) with the specified name.
//   - .**.name
//   - .**.“element-name”
func (p NodeSet) XGo_Any(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			p.Data(func(node Node) bool {
				return rangeAnyNodes(name, node, yield)
			})
		},
	}
}

// rangeAnyNodes yields all descendant nodes of the given node that match the
// specified name.
func rangeAnyNodes(name string, node Node, yield func(Node) bool) bool {
	if node.Name == name {
		if !yield(node) {
			return false
		}
	}
	for k, v := range node.Children {
		if child, ok := v.(map[string]any); ok {
			if !rangeAnyNodes(name, Node{Name: k, Children: child}, yield) {
				return false
			}
		}
	}
	return true
}

// -----------------------------------------------------------------------------

// _one returns a NodeSet containing the first node.
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

// _first returns the first node in the NodeSet.
func (p NodeSet) XGo_first() (Node, error) {
	if p.Err != nil {
		return Node{}, p.Err
	}
	return dql.First(p.Data)
}

// _hasAttr returns true if the first node in the NodeSet has the specified attribute.
// It returns false otherwise.
func (p NodeSet) XGo_hasAttr(name string) bool {
	node, err := p.XGo_first()
	if err == nil {
		_, ok := node.Children[name]
		return ok
	}
	return false
}

// XGo_Attr returns the value of the specified attribute from the first node in the
// NodeSet. It only retrieves the attribute from the first node.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr(name string) (val any, err error) {
	node, err := p.XGo_first()
	if err == nil {
		if v, ok := node.Children[name]; ok {
			return v, nil
		}
		err = dql.ErrNotFound
	}
	return
}

// -----------------------------------------------------------------------------
