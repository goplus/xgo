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

	"golang.org/x/net/html"
)

// -----------------------------------------------------------------------------

// File represents an HTML file.
type File struct {
	html.Node
	// File must contain only the embedded html.Node field.
}

// Parse returns the parse tree for the HTML from the given Reader.
func Parse(r io.Reader) (f *File, err error) {
	doc, err := html.Parse(r)
	if err == nil {
		f = (*File)(unsafe.Pointer(doc))
	}
	return
}

// -----------------------------------------------------------------------------

// XGo_Elem returns a NodeSet containing the child nodes with the specified name.
//   - .name
//   - .“element-name”
func (f *File) XGo_Elem(name string) NodeSet {
	return Root(&f.Node).XGo_Elem(name)
}

// XGo_Child returns a NodeSet containing all child nodes of the node.
//   - .*
func (f *File) XGo_Child() NodeSet {
	return Root(&f.Node).XGo_Child()
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// node itself) with the specified name.
// If name is "", it returns all nodes.
//   - .**.name
//   - .**.“element-name”
//   - .**.*
func (f *File) XGo_Any(name string) NodeSet {
	return Root(&f.Node).XGo_Any(name)
}

// Dump prints the node for debugging purposes.
func (f *File) Dump() NodeSet {
	return Root(&f.Node).Dump()
}

// -----------------------------------------------------------------------------
