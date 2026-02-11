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

package xml

import (
	"bytes"
	"encoding/xml"
	"io"
	"iter"
	"unsafe"

	"github.com/goplus/xgo/dql"
	"github.com/goplus/xgo/dql/stream"
)

// -----------------------------------------------------------------------------

// Node represents a generic XML node with its name, attributes, and children.
type Node struct {
	Name     xml.Name
	Attr     []xml.Attr
	Children []any // can be *Node or xml.CharData
}

// UnmarshalXML implements the xml.Unmarshaler interface for the Node struct.
func (n *Node) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Name = start.Name
	n.Attr = start.Attr
	for {
		token, err := d.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			child := &Node{}
			if err := d.DecodeElement(child, &t); err != nil {
				return err
			}
			n.Children = append(n.Children, child)

		case xml.CharData:
			n.Children = append(n.Children, t)

		case xml.EndElement:
			return nil
		}
	}
}

// -----------------------------------------------------------------------------

// NodeSet represents a set of XML nodes.
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

// New parses the XML document from the provided reader and returns a NodeSet
// containing the root node. If there is an error during parsing, the NodeSet's
// Err field is set.
func New(r io.Reader) NodeSet {
	var doc Node
	err := xml.NewDecoder(r).Decode(&doc)
	if err != nil {
		return NodeSet{Err: err}
	}
	return NodeSet{
		Data: func(yield func(*Node) bool) {
			yield(&doc)
		},
	}
}

// Source creates a NodeSet from various types of sources:
// - string: treated as an URL to read XML content from.
// - []byte: treated as raw XML content.
// - io.Reader: reads XML content from the reader.
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
		panic("dql/xml.Source: unsupport source type")
	}
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
	if node.Name.Local == name {
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
	for _, c := range n.Children {
		if child, ok := c.(*Node); ok {
			if child.Name.Local == name {
				if !yield(child) {
					return false
				}
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
	for _, c := range n.Children {
		if child, ok := c.(*Node); ok {
			if !yield(child) {
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
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return rangeAnyNodes(node, name, yield)
			})
		},
	}
}

// rangeAnyNodes yields all descendant nodes of the given node that match the
// specified name.
func rangeAnyNodes(n *Node, name string, yield func(*Node) bool) bool {
	if n.Name.Local == name {
		if !yield(n) {
			return false
		}
	}
	for _, c := range n.Children {
		if child, ok := c.(*Node); ok {
			if !rangeAnyNodes(child, name, yield) {
				return false
			}
		}
	}
	return true
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
			if attr.Name.Local == name {
				val, err = attr.Value, nil
				return false
			}
		}
		return true
	})
	return
}

// Text returns the text content of the first text node found in the NodeSet.
func (p NodeSet) Text() (val string, err error) {
	if p.Err != nil {
		return "", p.Err
	}
	err = dql.ErrNotFound
	p.Data(func(node *Node) bool {
		for _, c := range node.Children {
			if data, ok := c.(xml.CharData); ok {
				val, err = unsafe.String(unsafe.SliceData(data), len(data)), nil
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
