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

package reflects

import (
	"iter"
	"reflect"

	"github.com/goplus/xgo/dql"
)

const (
	XGoPackage = true
)

// -----------------------------------------------------------------------------

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

// New creates a NodeSet containing a single provided node.
func New(doc reflect.Value) NodeSet {
	return NodeSet{
		Data: func(yield func(Node) bool) {
			yield(Node{Name: "", Value: doc})
		},
	}
}

// Source creates a NodeSet from various types of sources:
// - reflect.Value: creates a NodeSet containing the single provided node.
// - Node: creates a NodeSet containing the single provided node.
// - iter.Seq[Node]: directly uses the provided sequence of nodes.
// - NodeSet: returns the provided NodeSet as is.
// - any other type: uses reflect.ValueOf to create a NodeSet.
func Source(r any) (ret NodeSet) {
	switch v := r.(type) {
	case reflect.Value:
		return New(v)
	case Node:
		return Root(v)
	case iter.Seq[Node]:
		return NodeSet{Data: v}
	case NodeSet:
		return v
	default:
		return New(reflect.ValueOf(r))
	}
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
				return yieldElem(node, name, yield)
			})
		},
	}
}

// yieldElem yields the child node with the specified name if it exists.
func yieldElem(node Node, name string, yield func(Node) bool) bool {
	if v := lookup(node.Value, name); v.IsValid() {
		return yield(Node{Name: name, Value: v})
	}
	return true
}

func yieldChildNodes(node reflect.Value, yield func(Node) bool) bool {
	kind, node := deref(node)
	switch kind {
	case reflect.Struct:
		typ := node.Type()
		for i, n := 0, typ.NumField(); i < n; i++ {
			if v := node.Field(i); v.CanInterface() { // only yield exported fields
				if !yield(Node{Name: uncapitalize(typ.Field(i).Name), Value: v}) {
					return false
				}
			}
		}
	case reflect.Map:
		typ := node.Type()
		if typ.Key().Kind() != reflect.String { // Only support map[string]T
			break
		}
		it := node.MapRange()
		for it.Next() {
			if !yield(Node{Name: it.Key().String(), Value: it.Value()}) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < node.Len(); i++ {
			if !yield(Node{Name: "", Value: node.Index(i)}) {
				return false
			}
		}
	}
	return true
}

// yieldAnyNodes yields all descendant nodes of the given node that match the
// specified name. If name is "", it yields all nodes.
func yieldAnyNodes(name string, node Node, yield func(Node) bool) bool {
	if name == "" || node.Name == name {
		if !yield(node) {
			return false
		}
	}
	return yieldChildNodes(node.Value, func(n Node) bool {
		return yieldAnyNodes(name, n, yield)
	})
}

// XGo_Child returns a NodeSet containing all child nodes of the nodes in the NodeSet.
func (p NodeSet) XGo_Child() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(Node) bool) {
			p.Data(func(node Node) bool {
				return yieldChildNodes(node.Value, yield)
			})
		},
	}
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
		ret = node.Value.Interface()
	}
	return
}

// XGo_class returns the class name of the first node in the NodeSet.
func (p NodeSet) XGo_class() (class string) {
	node, err := p.XGo_first()
	if err != nil {
		return
	}
	_, v := deref(node.Value)
	if v.IsValid() {
		return v.Type().Name()
	}
	return ""
}

// _hasAttr returns true if the first node in the NodeSet has the specified attribute.
// It returns false otherwise.
func (p NodeSet) XGo_hasAttr(name string) bool {
	node, err := p.XGo_first()
	if err == nil {
		return lookup(node.Value, name).IsValid()
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
func (p NodeSet) XGo_Attr__1(name string) (any, error) {
	node, err := p.XGo_first()
	if err == nil {
		return node.XGo_Attr__1(name)
	}
	return nil, err
}

// -----------------------------------------------------------------------------
