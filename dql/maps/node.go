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
	"github.com/goplus/xgo/dql"
)

// -----------------------------------------------------------------------------

// Node represents a named value in a DQL query tree.
type Node struct {
	Name  string
	Value any
}

// XGo_Elem returns the child node with the specified name.
//   - .name
//   - .“element-name”
func (n Node) XGo_Elem(name string) (ret Node) {
	if children, ok := n.Value.(map[string]any); ok {
		if v, ok := children[name]; ok {
			ret = Node{Name: name, Value: v}
		}
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

// _hasAttr returns true if the node has the specified attribute.
func (n Node) XGo_hasAttr(name string) bool {
	switch children := n.Value.(type) {
	case map[string]any:
		_, ok := children[name]
		return ok
	}
	return false
}

// XGo_Attr returns the value of the specified attribute from the node.
// If the attribute does not exist, it returns nil.
//   - $name
//   - $“attr-name”
func (n Node) XGo_Attr__0(name string) any {
	val, _ := n.XGo_Attr__1(name)
	return val
}

// XGo_Attr returns the value of the specified attribute from the node.
// If the attribute does not exist, it returns ErrNotFound.
//   - $name
//   - $“attr-name”
func (n Node) XGo_Attr__1(name string) (any, error) {
	switch children := n.Value.(type) {
	case map[string]any:
		if v, ok := children[name]; ok {
			return v, nil
		}
	}
	return nil, dql.ErrNotFound
}

// -----------------------------------------------------------------------------
