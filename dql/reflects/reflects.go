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

// Root creates a NodeSet containing the provided root node.
func Root(doc Node) NodeSet {
	return NodeSet{
		Data: func(yield func(Node) bool) {
			yield(doc)
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
				return yieldNode(node, name, yield)
			})
		},
	}
}

// yieldNode yields the child node with the specified name if it exists.
func yieldNode(node Node, name string, yield func(Node) bool) bool {
	if v := lookup(node.Children, name); isNode(v) {
		return yield(Node{Name: name, Children: v})
	}
	return true
}

func lookup(node reflect.Value, name string) (ret reflect.Value) {
	kind := node.Kind()
	switch kind {
	case reflect.Pointer, reflect.Interface:
		node = node.Elem()
		kind = node.Kind()
	}
	switch kind {
	case reflect.Struct:
		ret = node.FieldByName(capitalize(name))
	case reflect.Map:
		ret = node.MapIndex(reflect.ValueOf(name))
	}
	return
}

func isNode(v reflect.Value) bool {
	kind := v.Kind()
	switch kind {
	case reflect.Invalid:
		return false
	case reflect.Pointer, reflect.Interface:
		v = v.Elem()
		kind = v.Kind()
	}
	return kind == reflect.Struct || kind == reflect.Map
}

func rangeChildNodes(node reflect.Value, yield func(Node) bool) bool {
	kind := node.Kind()
	switch kind {
	case reflect.Pointer, reflect.Interface:
		node = node.Elem()
		kind = node.Kind()
	}
	switch kind {
	case reflect.Struct:
		typ := node.Type()
		for i := 0; i < typ.NumField(); i++ {
			v := node.Field(i)
			if isNode(v) {
				if !yield(Node{Name: uncapitalize(typ.Field(i).Name), Children: v}) {
					return false
				}
			}
		}
	case reflect.Map:
		typ := node.Type()
		if typ.Key().Kind() != reflect.String {
			return true // only string keys are supported
		}
		it := node.MapRange()
		for it.Next() {
			v := it.Value()
			if isNode(v) {
				if !yield(Node{Name: it.Key().String(), Children: v}) {
					return false
				}
			}
		}
	}
	return true
}

// rangeAnyNodes yields all descendant nodes of the given node that match the
// specified name.
func rangeAnyNodes(name string, node Node, yield func(Node) bool) bool {
	if node.Name == name {
		if !yield(node) {
			return false
		}
	}
	return rangeChildNodes(node.Children, func(n Node) bool {
		return rangeAnyNodes(name, n, yield)
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
				return rangeChildNodes(node.Children, yield)
			})
		},
	}
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

// XGo_Attr returns the value of the first specified attribute found in the NodeSet.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr(name string) (val any, err error) {
	if p.Err != nil {
		return "", p.Err
	}
	err = dql.ErrNotFound
	p.Data(func(node Node) bool {
		if v := lookup(node.Children, name); v.IsValid() {
			val, err = v.Interface(), nil
			return false
		}
		return true
	})
	return
}

// -----------------------------------------------------------------------------
