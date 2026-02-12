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
	"bytes"
	"io"
	"iter"

	"github.com/goplus/xgo/dql"
	"github.com/goplus/xgo/dql/stream"
	"golang.org/x/net/html"
)

// -----------------------------------------------------------------------------

// Node represents an HTML node.
type Node = html.Node

// NodeSet represents a set of HTML nodes.
type NodeSet struct {
	Data iter.Seq[*Node]
	Err  error
}

// Root creates a NodeSet containing the provided root node.
func Root(doc *Node) NodeSet {
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			yield(doc)
		},
	}
}

// New parses the HTML document from the provided reader and returns a NodeSet
// containing the root node. If there is an error during parsing, the NodeSet's
// Err field is set.
func New(r io.Reader) NodeSet {
	doc, err := html.Parse(r)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(doc)
}

// Source creates a NodeSet from various types of sources:
// - string: treated as an URL to read HTML content from.
// - []byte: treated as raw HTML content.
// - io.Reader: reads HTML content from the reader.
// - *Node: creates a NodeSet containing the single provided node.
// - iter.Seq[*Node]: directly uses the provided sequence of nodes.
// - NodeSet: returns the provided NodeSet as is.
// If the source type is unsupported, it panics.
func Source(r any) (ret NodeSet) {
	switch v := r.(type) {
	case string:
		f, err := stream.Open(v)
		if err != nil {
			return NodeSet{Err: err}
		}
		defer f.Close()
		return New(f)
	case []byte:
		r := bytes.NewReader(v)
		return New(r)
	case io.Reader:
		return New(v)
	case *Node:
		return Root(v)
	case iter.Seq[*Node]:
		return NodeSet{Data: v}
	case NodeSet:
		return v
	default:
		panic("dql/html.Source: unsupport source type")
	}
}

// -----------------------------------------------------------------------------

// XGo_Node returns the first node in the NodeSet.
func (p NodeSet) XGo_Node() (ret *Node, err error) {
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
		p.Data(func(node *Node) bool {
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
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return selectNode(node, name, yield)
			})
		},
	}
}

// selectNode yields the node if it matches the specified name.
func selectNode(node *Node, name string, yield func(*Node) bool) bool {
	if node.Type == html.ElementNode && node.Data == name {
		return yield(node)
	}
	return true
}

// XGo_Elem returns a NodeSet containing the child nodes with the specified name.
//   - .name
//   - .“element-name”
func (p NodeSet) XGo_Elem(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldNode(node, name, yield)
			})
		},
	}
}

// yieldNode yields the child node with the specified name if it exists.
func yieldNode(n *Node, name string, yield func(*Node) bool) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == name {
			if !yield(c) {
				return false
			}
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
		Data: func(yield func(*Node) bool) {
			p.Data(func(n *Node) bool {
				return rangeChildNodes(n, yield)
			})
		},
	}
}

// rangeChildNodes yields all child nodes of the given node.
func rangeChildNodes(n *Node, yield func(*Node) bool) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !yield(c) {
			return false
		}
	}
	return true
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// nodes themselves) with the specified name.
// If name is "textNode", it returns all text nodes.
//   - .**.name
//   - .**.“element-name”
func (p NodeSet) XGo_Any(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return rangeAnyNodes(node, name, yield)
			})
		},
	}
}

// rangeAnyNodes yields all descendant nodes of the given node that match the
// specified name. If name is "textNode", it yields text nodes.
func rangeAnyNodes(n *Node, name string, yield func(*Node) bool) bool {
	switch name {
	case "textNode":
		if n.Type == html.TextNode {
			if !yield(n) {
				return false
			}
		}
	default:
		if n.Type == html.ElementNode && n.Data == name {
			if !yield(n) {
				return false
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !rangeAnyNodes(c, name, yield) {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------

// One returns a NodeSet containing the first node.
func (p NodeSet) One() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	n, err := dql.First(p.Data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(n)
}

// Single returns a NodeSet containing the single node.
// If there are zero or more than one nodes, it returns an error.
// ErrNotFound or ErrMultiEntities is returned accordingly.
func (p NodeSet) Single() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	n, err := dql.Single(p.Data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(n)
}

// ParentN returns a NodeSet containing the N-th parent nodes.
func (p NodeSet) ParentN(n int) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldParentN(node, n, yield)
			})
		},
	}
}

func yieldParentN(node *Node, n int, yield func(*Node) bool) bool {
	if n > 0 {
		for {
			node = node.Parent
			if node == nil {
				break
			}
			n--
			if n == 0 {
				return yield(node)
			}
		}
	}
	return true
}

// Parent returns a NodeSet containing the parent nodes.
func (p NodeSet) Parent() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				if next := node.Parent; next != nil {
					return yield(next)
				}
				return true
			})
		},
	}
}

// PrevSibling returns a NodeSet containing the previous sibling nodes.
func (p NodeSet) PrevSibling() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				if next := node.PrevSibling; next != nil {
					return yield(next)
				}
				return true
			})
		},
	}
}

// NextSibling returns a NodeSet containing the next sibling nodes.
func (p NodeSet) NextSibling() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				if next := node.NextSibling; next != nil {
					return yield(next)
				}
				return true
			})
		},
	}
}

// FirstElementChild returns a NodeSet containing the first element
// child of each node.
func (p NodeSet) FirstElementChild() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode {
						return yield(c)
					}
				}
				return true
			})
		},
	}
}

// TextNode returns a NodeSet containing all text nodes.
func (p NodeSet) TextNode() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldNodeType(node, html.TextNode, yield)
			})
		},
	}
}

func yieldNodeType(node *Node, typ html.NodeType, yield func(*Node) bool) bool {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == typ {
			if !yield(c) {
				return false
			}
		}
	}
	return true
}

// -----------------------------------------------------------------------------

// Collect retrieves all nodes from the NodeSet.
func (p NodeSet) Collect() ([]*Node, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return dql.Collect(p.Data), nil
}

// First returns the first node in the NodeSet.
func (p NodeSet) First() (*Node, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return dql.First(p.Data)
}

// Value returns the data content of the first node in the NodeSet.
func (p NodeSet) Value() (val string, err error) {
	node, err := p.First()
	if err == nil {
		return node.Data, nil
	}
	return
}

// HasAttr returns true if the first node in the NodeSet has the specified attribute.
// It returns false otherwise.
func (p NodeSet) HasAttr(name string) bool {
	node, err := p.First()
	if err == nil {
		for _, attr := range node.Attr {
			if attr.Key == name {
				return true
			}
		}
	}
	return false
}

// XGo_Attr returns the value of the specified attribute from the first node in the
// NodeSet. It only retrieves the attribute from the first node.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr(name string) (val string, err error) {
	node, err := p.First()
	if err == nil {
		for _, attr := range node.Attr {
			if attr.Key == name {
				return attr.Val, nil
			}
		}
		err = dql.ErrNotFound // attribute not found on first node
	}
	return
}

// Text retrieves the text content of the first child text node.
// It only retrieves from the first node in the NodeSet.
func (p NodeSet) Text() (val string, err error) {
	return p.valByNodeType(html.TextNode)
}

// valByNodeType retrieves the data content of the first child node of the specified
// type. It only retrieves from the first node in the NodeSet.
func (p NodeSet) valByNodeType(typ html.NodeType) (val string, err error) {
	node, err := p.First()
	if err == nil {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == typ {
				return c.Data, nil
			}
		}
		err = dql.ErrNotFound // nodeType not found on first node
	}
	return
}

// Int retrieves the integer value from the text content of the first child
// text node. It only retrieves from the first node in the NodeSet.
func (p NodeSet) Int() (int, error) {
	text, err := p.Text()
	if err != nil {
		return 0, err
	}
	return dql.Int__0(text)
}

// -----------------------------------------------------------------------------
