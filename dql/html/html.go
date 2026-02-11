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

func (p NodeSet) One() NodeSet {
	panic("todo")
}

func (p NodeSet) ParentN(n int) NodeSet {
	panic("todo")
}

func (p NodeSet) NextSibling() NodeSet {
	panic("todo")
}

func (p NodeSet) FirstElementChild() NodeSet {
	panic("todo")
}

func (p NodeSet) TextNode() NodeSet {
	panic("todo")
}

// -----------------------------------------------------------------------------

// XGo_Attr returns the value of the first specified attribute found in the NodeSet.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr(name string) (val string, err error) {
	if p.Err != nil {
		return "", p.Err
	}
	err = dql.ErrNotFound
	p.Data(func(node *Node) bool {
		for _, attr := range node.Attr {
			if attr.Key == name {
				val, err = attr.Val, nil
				return false
			}
		}
		return true
	})
	return
}

// Text returns the text content of the first text node found in the NodeSet.
func (p NodeSet) Text() (val string, err error) {
	return p.valByNodeType(html.TextNode)
}

func (p NodeSet) valByNodeType(typ html.NodeType) (val string, err error) {
	err = dql.ErrNotFound
	p.Data(func(node *Node) bool {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == typ {
				val, err = c.Data, nil
				return false
			}
		}
		return true
	})
	return
}

// Int parses the text content of the first text node found in the NodeSet as an integer.
func (p NodeSet) Int() (int, error) {
	text, err := p.Text()
	if err != nil {
		return 0, err
	}
	return dql.Int__0(text)
}

// -----------------------------------------------------------------------------
