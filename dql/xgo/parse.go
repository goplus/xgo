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

package xgo

import (
	"unsafe"

	"github.com/goplus/xgo/ast"
)

// -----------------------------------------------------------------------------

// File represents a XGo file.
type File struct {
	ast.File
	// File must contain only the embedded ast.File field.
}

// ParseFile parses XGo source code from the given filename or source, returning a File
// object. An optional Config can be provided to customize the parsing behavior.
func ParseFile(filename string, src any, conf ...Config) (f *File, err error) {
	doc, err := parse(filename, src, conf...)
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
	return New(&f.File).XGo_Elem(name)
}

// XGo_Child returns a NodeSet containing all child nodes of the node.
//   - .*
func (f *File) XGo_Child() NodeSet {
	return New(&f.File).XGo_Child()
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// node itself) with the specified name.
// If name is "", it returns all nodes.
//   - .**.name
//   - .**.“element-name”
//   - .**.*
func (f *File) XGo_Any(name string) NodeSet {
	return New(&f.File).XGo_Any(name)
}

// -----------------------------------------------------------------------------
