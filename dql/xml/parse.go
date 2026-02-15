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
	"encoding/xml"
	"io"
)

// -----------------------------------------------------------------------------

// A CharData represents XML character data (raw text),
// in which XML escape sequences have been replaced by
// the characters they represent.
type CharData = xml.CharData

// A Name represents an XML name (Local) annotated
// with a name space identifier (Space).
// In tokens returned by [Decoder.Token], the Space identifier
// is given as a canonical URL, not the short prefix used
// in the document being parsed.
type Name = xml.Name

// An Attr represents an attribute in an XML element (Name=Value).
type Attr = xml.Attr

// Node represents a generic XML node with its name, attributes, and children.
type Node struct {
	Name     xml.Name
	Attr     []xml.Attr
	Children []any // can be *Node or xml.CharData
}

// Parse returns the parse tree for the XML from the given Reader.
func Parse(r io.Reader) (doc *Node, err error) {
	doc = new(Node)
	err = xml.NewDecoder(r).Decode(doc)
	return
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

// -----------------------------------------------------------------------------
