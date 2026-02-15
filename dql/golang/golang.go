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

package golang

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"iter"
	"reflect"

	"github.com/goplus/xgo/dql"
	"github.com/goplus/xgo/dql/reflects"
)

const (
	XGoPackage = "github.com/goplus/xgo/dql/reflects"
)

// -----------------------------------------------------------------------------

// Node represents a Go AST node.
type Node = reflects.Node

// NodeSet represents a set of Go AST nodes.
type NodeSet struct {
	reflects.NodeSet
}

// NodeSet(seq) casts a NodeSet from a sequence of nodes.
func NodeSet_Cast(seq iter.Seq[Node]) NodeSet {
	return NodeSet{
		NodeSet: reflects.NodeSet{Data: seq},
	}
}

// Root creates a NodeSet containing the provided root node.
func Root(doc Node) NodeSet {
	return NodeSet{
		NodeSet: reflects.Root(doc),
	}
}

// Nodes creates a NodeSet containing the provided nodes.
func Nodes(nodes ...Node) NodeSet {
	return NodeSet{
		NodeSet: reflects.Nodes(nodes...),
	}
}

// New creates a NodeSet from the given *ast.File.
func New(f *ast.File) NodeSet {
	return NodeSet{
		NodeSet: reflects.New(reflect.ValueOf(f)),
	}
}

// Config represents the configuration for parsing Go source code.
type Config struct {
	Mode parser.Mode
	Fset *token.FileSet
}

const (
	defaultMode = parser.ParseComments
)

// parse parses Go source code from the given filename or source.
func parse(filename string, src any, conf ...Config) (f *ast.File, err error) {
	var c Config
	if len(conf) > 0 {
		c = conf[0]
	} else {
		c.Mode = defaultMode
	}
	if c.Fset == nil {
		c.Fset = token.NewFileSet()
	}
	return parser.ParseFile(c.Fset, filename, src, c.Mode)
}

// From parses Go source code from the given filename or source, returning a NodeSet.
// An optional Config can be provided to customize the parsing behavior.
func From(filename string, src any, conf ...Config) NodeSet {
	f, err := parse(filename, src, conf...)
	if err != nil {
		return NodeSet{NodeSet: reflects.NodeSet{Err: err}}
	}
	return New(f)
}

// Source creates a NodeSet from various types of Go sources.
// It supports the following source types:
// - string: treats the string as a file path, opens the file, and reads Go source code from it.
// - []byte: treated as Go source code.
// - *bytes.Buffer: treated as Go source code.
// - io.Reader: treated as Go source code.
// - *ast.File: creates a NodeSet from the provided *ast.File.
// - reflect.Value: creates a NodeSet from the provided reflect.Value (expected to be *ast.File).
// - Node: creates a NodeSet containing the single provided node.
// - iter.Seq[Node]: returns the provided sequence as a NodeSet.
// - NodeSet: returns the provided NodeSet as is.
// If the source type is unsupported, it panics.
func Source(r any, conf ...Config) (ret NodeSet) {
	switch v := r.(type) {
	case string:
		return From(v, nil, conf...)
	case []byte:
		return From("", v, conf...)
	case *bytes.Buffer:
		return From("", v, conf...)
	case io.Reader:
		return From("", v, conf...)
	case *ast.File:
		return New(v)
	case reflect.Value:
		return NodeSet{NodeSet: reflects.New(v)}
	case Node:
		return NodeSet{NodeSet: reflects.Root(v)}
	case iter.Seq[Node]:
		return NodeSet{NodeSet: reflects.NodeSet{Data: v}}
	case NodeSet:
		return v
	default:
		panic("dql/golang.Source: unsupport source type")
	}
}

// -----------------------------------------------------------------------------

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
	return NodeSet{
		NodeSet: p.NodeSet.XGo_Select(name),
	}
}

// XGo_Elem returns a NodeSet containing the child nodes with the specified name.
//   - .name
//   - .“element-name”
func (p NodeSet) XGo_Elem(name string) NodeSet {
	return NodeSet{
		NodeSet: p.NodeSet.XGo_Elem(name),
	}
}

// XGo_Child returns a NodeSet containing all child nodes of the nodes in the NodeSet.
func (p NodeSet) XGo_Child() NodeSet {
	return NodeSet{
		NodeSet: p.NodeSet.XGo_Child(),
	}
}

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// nodes themselves) with the specified name.
// If name is "", it returns all nodes.
//   - .**.name
//   - .**.“element-name”
//   - .**.*
func (p NodeSet) XGo_Any(name string) NodeSet {
	return NodeSet{
		NodeSet: p.NodeSet.XGo_Any(name),
	}
}

// -----------------------------------------------------------------------------

// All returns a NodeSet containing all nodes.
// It's a cache operation for performance optimization when you need to traverse
// the nodes multiple times.
func (p NodeSet) All() NodeSet {
	return NodeSet{
		NodeSet: p.NodeSet.XGo_all(),
	}
}

// One returns a NodeSet containing the first node.
// It's a performance optimization when you only need the first node (stop early).
func (p NodeSet) One() NodeSet {
	return NodeSet{
		NodeSet: p.NodeSet.XGo_one(),
	}
}

// Single returns a NodeSet containing the single node.
// If there are zero or more than one nodes, it returns an error.
// ErrNotFound or ErrMultipleResults is returned accordingly.
func (p NodeSet) Single() NodeSet {
	return NodeSet{
		NodeSet: p.NodeSet.XGo_single(),
	}
}

// -----------------------------------------------------------------------------

// XGo_Attr returns the value of the specified attribute from the first node in the
// NodeSet. It only retrieves the attribute from the first node.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr__0(name string) any {
	val, _ := p.XGo_Attr__1(name)
	return val
}

// XGo_Attr returns the value of the specified attribute from the first node in the
// NodeSet. It only retrieves the attribute from the first node.
//   - $name
//   - $“attr-name”
func (p NodeSet) XGo_Attr__1(name string) (val any, err error) {
	val, err = p.NodeSet.XGo_Attr__1(name)
	if err == nil {
		switch v := val.(type) {
		case *ast.Ident:
			return v.Name, nil
		case *ast.BasicLit:
			return v.Value, nil
		}
	}
	return
}

// Class returns the class name of the first node in the NodeSet.
func (p NodeSet) Class() string {
	return p.XGo_class()
}

// -----------------------------------------------------------------------------
