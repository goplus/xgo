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

package html

import (
	"io"
	"unsafe"

	"github.com/goplus/xgo/dql"
	"golang.org/x/net/html"
)

// -----------------------------------------------------------------------------

// Node represents an HTML node.
type Node struct {
	html.Node
	// Node must contain only the embedded html.Node field.
}

// Parse returns the parse tree for the HTML from the given Reader.
func Parse(r io.Reader) (n *Node, err error) {
	doc, err := html.Parse(r)
	if err == nil {
		n = toNode(doc)
	}
	return
}

func toNode(n *html.Node) *Node {
	return (*Node)(unsafe.Pointer(n))
}

// -----------------------------------------------------------------------------

// XGo_Elem returns a NodeSet containing the child nodes with the specified name.
//   - .name
//   - .“element-name”
func (n *Node) XGo_Elem(name string) NodeSet {
	return Root(n).XGo_Elem(name)
}

// XGo_Child returns a NodeSet containing all child nodes of the node.
//   - .*
func (n *Node) XGo_Child() NodeSet {
	return Root(n).XGo_Child()
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// node itself) with the specified name.
// If name is "", it returns all nodes.
//   - .**.name
//   - .**.“element-name”
//   - .**.*
func (n *Node) XGo_Any(name string) NodeSet {
	return Root(n).XGo_Any(name)
}

// Dump prints the node for debugging purposes.
func (n *Node) Dump() NodeSet {
	return Root(n).Dump()
}

// -----------------------------------------------------------------------------

// Name returns the name of the node if it's an element node, or an empty string
// otherwise.
func (n *Node) Name() string {
	if n.Type == html.ElementNode {
		return n.Data
	}
	return ""
}

// Value returns the data of the node.
func (n *Node) Value() string {
	return n.Data
}

// HasAttr returns true if the node has the specified attribute.
func (n *Node) HasAttr(name string) bool {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return true
		}
	}
	return false
}

// XGo_Attr returns the value of the specified attribute from the node.
// If the attribute is not found, it returns an empty string.
//   - $name
//   - $“attr-name”
func (n *Node) XGo_Attr__0(name string) string {
	val, _ := n.XGo_Attr__1(name)
	return val
}

// XGo_Attr returns the value of the specified attribute from the node.
// If the attribute is not found, it returns ErrNotFound.
//   - $name
//   - $“attr-name”
func (n *Node) XGo_Attr__1(name string) (string, error) {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val, nil
		}
	}
	return "", dql.ErrNotFound
}

// -----------------------------------------------------------------------------
