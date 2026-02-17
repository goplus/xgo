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

// capitalize capitalizes the first letter of the given name.
func capitalize(name string) string {
	if name != "" {
		if c := name[0]; c >= 'a' && c <= 'z' {
			return string(c-'a'+'A') + name[1:]
		}
	}
	return name
}

// uncapitalize uncapitalizes the first letter of the given name.
func uncapitalize(name string) string {
	if name != "" {
		if c := name[0]; c >= 'A' && c <= 'Z' {
			return string(c-'A'+'a') + name[1:]
		}
	}
	return name
}

// -----------------------------------------------------------------------------

// Node represents a reflect.Value node.
type Node struct {
	Name     string
	Children reflect.Value
}

// NodeSet represents a set of reflect.Value nodes.
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
			yield(Node{Name: "", Children: doc})
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
	if v := lookup(node.Children, name); isNode(v) {
		return yield(Node{Name: name, Children: v})
	}
	return true
}

func lookup(node reflect.Value, name string) (ret reflect.Value) {
	kind, node := deref(node)
	switch kind {
	case reflect.Struct:
		ret = node.FieldByName(capitalize(name))
	case reflect.Map:
		ret = node.MapIndex(reflect.ValueOf(name))
	}
	return
}

func deref(v reflect.Value) (reflect.Kind, reflect.Value) {
	kind := v.Kind()
	if kind == reflect.Interface {
		v = v.Elem()
		kind = v.Kind()
	}
	if kind == reflect.Pointer {
		v = v.Elem()
		kind = v.Kind()
	}
	return kind, v
}

func isNodeType(v reflect.Type) bool {
	kind := v.Kind()
	if kind == reflect.Pointer {
		v = v.Elem()
		kind = v.Kind()
	}
	switch kind {
	case reflect.Struct, reflect.Map, reflect.Interface:
		return true
	case reflect.Slice:
		ekind := v.Elem().Kind()
		return ekind > reflect.Complex128 && ekind != reflect.String
	}
	return false
}

func isNode(v reflect.Value) bool {
	kind, v := deref(v)
	switch kind {
	case reflect.Struct, reflect.Map:
		return true
	case reflect.Slice:
		ekind := v.Type().Elem().Kind()
		return ekind > reflect.Complex128 && ekind != reflect.String
	}
	return false
}

func yieldChildNodes(node reflect.Value, yield func(Node) bool) bool {
	kind, node := deref(node)
	switch kind {
	case reflect.Struct:
		typ := node.Type()
		for i, n := 0, typ.NumField(); i < n; i++ {
			v := node.Field(i)
			if isNode(v) {
				if !yield(Node{Name: uncapitalize(typ.Field(i).Name), Children: v}) {
					return false
				}
			}
		}
	case reflect.Map:
		typ := node.Type()
		if typ.Key().Kind() != reflect.String || !isNodeType(typ.Elem()) {
			break
		}
		it := node.MapRange()
		for it.Next() {
			if !yield(Node{Name: it.Key().String(), Children: it.Value()}) {
				return false
			}
		}
	case reflect.Slice:
		elemType := node.Type().Elem()
		if !isNodeType(elemType) {
			break
		}
		for i := 0; i < node.Len(); i++ {
			if !yield(Node{Name: "", Children: node.Index(i)}) {
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
	return yieldChildNodes(node.Children, func(n Node) bool {
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
				return yieldChildNodes(node.Children, yield)
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
		return lookup(node.Children, name).IsValid()
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
		if v := lookup(node.Children, name); v.IsValid() {
			return v.Interface(), nil
		}
		err = dql.ErrNotFound
	}
	return
}

// XGo_class returns the class name of the first node in the NodeSet.
func (p NodeSet) XGo_class() (class string) {
	node, err := p.XGo_first()
	if err != nil {
		return
	}
	_, v := deref(node.Children)
	if v.IsValid() {
		return v.Type().Name()
	}
	return ""
}

// -----------------------------------------------------------------------------
