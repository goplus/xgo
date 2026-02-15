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

	"github.com/goplus/xgo/dql/reflects"
)

const (
	XGoPackage = "github.com/goplus/xgo/dql/reflects"
)

// -----------------------------------------------------------------------------

type Node = reflects.Node

// NodeSet represents a set of Go AST nodes.
type NodeSet = reflects.NodeSet

// New creates a NodeSet from the given *ast.File.
func New(f *ast.File) NodeSet {
	return reflects.New(reflect.ValueOf(f))
}

// Config represents the configuration for parsing Go source code.
type Config struct {
	Mode parser.Mode
	Fset *token.FileSet
}

const (
	defaultMode = parser.ParseComments
)

// From parses Go source code from the given filename or source, returning a NodeSet.
// An optional Config can be provided to customize the parsing behavior.
func From(filename string, src any, conf ...Config) NodeSet {
	var c Config
	if len(conf) > 0 {
		c = conf[0]
	} else {
		c.Mode = defaultMode
	}
	if c.Fset == nil {
		c.Fset = token.NewFileSet()
	}
	f, err := parser.ParseFile(c.Fset, filename, src, c.Mode)
	if err != nil {
		return NodeSet{Err: err}
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
		return reflects.New(v)
	case Node:
		return reflects.Root(v)
	case iter.Seq[Node]:
		return NodeSet{Data: v}
	case NodeSet:
		return v
	default:
		panic("dql/golang.Source: unsupport source type")
	}
}

// -----------------------------------------------------------------------------
