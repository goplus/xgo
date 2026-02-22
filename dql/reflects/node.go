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

const (
	lookupNormal = iota
	lookupCallMethod
	lookupDontCallMethod
)

func lookup(obj reflect.Value, name string, mode int) (ret reflect.Value) {
	kind, node := deref(obj)
	switch kind {
	case reflect.Struct:
		name = capitalize(name)
		ret = node.FieldByName(name)
		if !ret.IsValid() && mode != lookupNormal {
			if mth := obj.MethodByName(name); mth.IsValid() {
				if t := mth.Type(); t.NumIn() == 0 && t.NumOut() == 1 {
					if mode == lookupDontCallMethod {
						return mth // only find the method, do not call it
					}
					// method call as attribute value
					ret = mth.Call(nil)[0]
				}
			}
		}
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

// -----------------------------------------------------------------------------

// Node represents a named value in a DQL query tree.
type Node struct {
	Name  string
	Value reflect.Value
}

// XGo_Elem returns the child node with the specified name.
//   - .name
//   - .“element-name”
func (n Node) XGo_Elem(name string) (ret Node) {
	if v := lookup(n.Value, name, lookupNormal); v.IsValid() {
		ret = Node{Name: name, Value: v}
	}
	return
}

// XGo_Child returns a NodeSet containing all child nodes of the node.
//   - .*
func (n Node) XGo_Child() NodeSet {
	return Root(n).XGo_Child()
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// node itself) with the specified name.
// If name is "", it returns all nodes.
//   - .**.name
//   - .**.“element-name”
//   - .**.*
func (n Node) XGo_Any(name string) NodeSet {
	return Root(n).XGo_Any(name)
}

// -----------------------------------------------------------------------------

// _hasAttr checks if the node has an attribute with the specified name.
func (n Node) XGo_hasAttr(name string) bool {
	return lookup(n.Value, name, lookupDontCallMethod).IsValid()
}

// XGo_Attr returns the value of the attribute with the specified name.
// If the attribute does not exist, it returns nil.
//   - $name
//   - $“attr-name”
func (n Node) XGo_Attr__0(name string) any {
	val, _ := n.XGo_Attr__1(name)
	return val
}

// XGo_Attr returns the value of the attribute with the specified name.
// If the attribute does not exist, it returns ErrNotFound.
//   - $name
//   - $“attr-name”
func (n Node) XGo_Attr__1(name string) (any, error) {
	if v := lookup(n.Value, name, lookupCallMethod); v.IsValid() {
		return v.Interface(), nil
	}
	return nil, dql.ErrNotFound
}

// -----------------------------------------------------------------------------
