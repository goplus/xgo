/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
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

package gopq

import (
	"errors"
	"io/fs"
	"syscall"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/parser/fsx"
	"github.com/goplus/xgo/token"
)

// -----------------------------------------------------------------------------

const (
	XGoPackage = true
)

var (
	// ErrBreak - break
	ErrBreak = syscall.ELOOP

	// ErrNotFound - not found
	ErrNotFound = errors.New("not found")

	// ErrTooManyNodes - too may nodes
	ErrTooManyNodes = errors.New("too many nodes")

	// ErrUnexpectedNode - unexpected node
	ErrUnexpectedNode = errors.New("unexpected node")
)

// Node - node interface
type Node interface {
	ast.Node
	ForEach(filter func(node Node) error) error
	Obj() any
}

// NodeEnum - node enumerator
type NodeEnum interface {
	ForEach(filter func(node Node) error) error
}

// NodeSet - node set
type NodeSet struct {
	Data NodeEnum
	Err  error
}

// FromFile calls ParseFile for a single file and returns *ast.File node set.
func FromFile(fset *token.FileSet, filename string, src any, mode parser.Mode) (doc NodeSet, err error) {
	file, err := parser.ParseFile(fset, filename, src, mode)
	if err != nil {
		return
	}
	return NodeSet{Data: &oneNode{astFile{file}}}, nil
}

// FromFSFile calls ParseFSFile for a single file and returns *ast.File node set.
func FromFSFile(
	fset *token.FileSet, fs fsx.FileSystem,
	filename string, src any, mode parser.Mode) (doc NodeSet, err error) {
	file, err := parser.ParseFSFile(fset, fs, filename, src, mode)
	if err != nil {
		return
	}
	return NodeSet{Data: &oneNode{astFile{file}}}, nil
}

// FromDir calls ParseFile for all XGo files in the directory specified by path
// and returns a map of package name -> package AST with all the packages found.
//
// If filter != nil, only the XGo files with fs.FileInfo entries passing through
// the filter are considered. The mode bits are passed to ParseFile unchanged.
// Position information is recorded in fset, which must not be nil.
//
// If the directory couldn't be read, a nil map and the respective error are
// returned. If a parse error occurred, a non-nil but incomplete map and the
// first error encountered are returned.
func FromDir(
	fset *token.FileSet, path string,
	filter func(fs.FileInfo) bool, mode parser.Mode) (doc NodeSet, err error) {

	pkgs, err := parser.ParseDir(fset, path, filter, mode)
	if err != nil {
		return
	}
	return NodeSet{Data: &oneNode{astPackages(pkgs)}}, nil
}

// FromFSDir calls ParseFile for all XGo files in the directory specified by path
// and returns a map of package name -> package AST with all the packages found.
//
// If filter != nil, only the XGo files with fs.FileInfo entries passing through
// the filter are considered. The mode bits are passed to ParseFile unchanged.
// Position information is recorded in fset, which must not be nil.
//
// If the directory couldn't be read, a nil map and the respective error are
// returned. If a parse error occurred, a non-nil but incomplete map and the
// first error encountered are returned.
func FromFSDir(
	fset *token.FileSet, fs parser.FileSystem, path string,
	filter func(fs.FileInfo) bool, mode parser.Mode) (doc NodeSet, err error) {

	pkgs, err := parser.ParseFSDir(fset, fs, path, parser.Config{Filter: filter, Mode: mode})
	if err != nil {
		return
	}
	return NodeSet{Data: &oneNode{astPackages(pkgs)}}, nil
}

// Ok returns if node set is valid or not.
func (p NodeSet) Ok() bool {
	return p.Err == nil
}

// -----------------------------------------------------------------------------

// FuncDecl returns *ast.FuncDecl node set.
func (p NodeSet) FuncDecl__0() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astDecl); ok {
			_, ok = node.Decl.(*ast.FuncDecl)
			return ok
		}
		return false
	})
}

// FuncDecl returns *ast.FuncDecl node set.
func (p NodeSet) FuncDecl__1(name string) NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astDecl); ok {
			if fn, ok := node.Decl.(*ast.FuncDecl); ok {
				return fn.Name.Name == name
			}
		}
		return false
	})
}

// GenDecl returns *ast.GenDecl node set.
func (p NodeSet) GenDecl__0(tok token.Token) NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astDecl); ok {
			if decl, ok := node.Decl.(*ast.GenDecl); ok {
				return decl.Tok == tok
			}
		}
		return false
	})
}

// TypeSpec returns *ast.TypeSpec node set.
func (p NodeSet) TypeSpec() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astSpec); ok {
			_, ok = node.Spec.(*ast.TypeSpec)
			return ok
		}
		return false
	})
}

// ValueSpec returns *ast.ValueSpec node set.
func (p NodeSet) ValueSpec() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astSpec); ok {
			_, ok = node.Spec.(*ast.ValueSpec)
			return ok
		}
		return false
	})
}

// ImportSpec returns *ast.ImportSpec node set.
func (p NodeSet) ImportSpec() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astSpec); ok {
			_, ok = node.Spec.(*ast.ImportSpec)
			return ok
		}
		return false
	})
}

// ExprStmt returns *ast.ExprStmt node set.
func (p NodeSet) ExprStmt() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astStmt); ok {
			_, ok = node.Stmt.(*ast.ExprStmt)
			return ok
		}
		return false
	})
}

// AssignStmt returns *ast.AssignStmt node set.
func (p NodeSet) AssignStmt() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astStmt); ok {
			_, ok = node.Stmt.(*ast.AssignStmt)
			return ok
		}
		return false
	})
}

// CallExpr returns *ast.CallExpr node set.
func (p NodeSet) CallExpr__0() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astExpr); ok {
			_, ok = node.Expr.(*ast.CallExpr)
			return ok
		}
		return false
	})
}

// CallExpr returns *ast.CallExpr node set.
func (p NodeSet) CallExpr__1(name string) NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astExpr); ok {
			if expr, ok := node.Expr.(*ast.CallExpr); ok {
				return getName(expr.Fun, true) == name
			}
		}
		return false
	})
}

// CompositeLit returns *ast.CompositeLit node set.
func (p NodeSet) CompositeLit__0() NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astExpr); ok {
			_, ok = node.Expr.(*ast.CompositeLit)
			return ok
		}
		return false
	})
}

// CompositeLit returns *ast.CompositeLit node set.
func (p NodeSet) CompositeLit__1(name string) NodeSet {
	return p.Match(func(node Node) bool {
		if node, ok := node.(*astExpr); ok {
			if lit, ok := node.Expr.(*ast.CompositeLit); ok && lit.Type != nil {
				return getName(lit.Type, true) == name
			}
		}
		return false
	})
}

// -----------------------------------------------------------------------------

func (p NodeSet) XGo_Enum(callback func(node NodeSet)) {
	if p.Err == nil {
		p.Data.ForEach(func(node Node) error {
			t := NodeSet{Data: &oneNode{node}}
			callback(t)
			return nil
		})
	}
}

func (p NodeSet) ForEach(callback func(node NodeSet)) {
	p.XGo_Enum(callback)
}

// -----------------------------------------------------------------------------

type oneNode struct {
	Node
}

func (p *oneNode) ForEach(filter func(node Node) error) error {
	return filter(p.Node)
}

func (p *oneNode) Cached() int {
	return 1
}

// One returns the first node as a node set.
func (p NodeSet) One() NodeSet {
	if _, ok := p.Data.(*oneNode); ok {
		return p
	}
	node, err := p.CollectOne__0()
	if err != nil {
		return NodeSet{Err: err}
	}
	return NodeSet{Data: &oneNode{node}}
}

// One creates a node set that only contains a signle node.
func One__0(node Node) NodeSet {
	return NodeSet{Data: &oneNode{node}}
}

// One creates a node set that only contains a signle node.
func One__1(f *ast.File) NodeSet {
	return NodeSet{Data: &oneNode{astFile{f}}}
}

// One creates a node set that only contains a signle node.
func One__2(pkg *ast.Package) NodeSet {
	return NodeSet{Data: &oneNode{astPackage{pkg}}}
}

// -----------------------------------------------------------------------------

type fixNodes struct {
	nodes []Node
}

func (p *fixNodes) ForEach(filter func(node Node) error) error {
	for _, node := range p.nodes {
		if filter(node) == ErrBreak {
			return ErrBreak
		}
	}
	return nil
}

func (p *fixNodes) Cached() int {
	return len(p.nodes)
}

// Nodes creates a fixed node set.
func Nodes(nodes ...Node) NodeSet {
	return NodeSet{Data: &fixNodes{nodes}}
}

// -----------------------------------------------------------------------------

type cached interface {
	Cached() int
}

// Cache caches node set.
func (p NodeSet) Cache() NodeSet {
	if _, ok := p.Data.(cached); ok {
		return p
	}
	nodes, err := p.Collect()
	if err != nil {
		return NodeSet{Err: err}
	}
	return NodeSet{Data: &fixNodes{nodes}}
}

// -----------------------------------------------------------------------------

type anyNodes struct {
	data NodeEnum
}

func (p *anyNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		return anyForEach(node, filter)
	})
}

func anyForEach(p Node, filter func(node Node) error) error {
	if err := filter(p); err == ErrBreak {
		return err
	}
	return p.ForEach(func(node Node) error {
		return anyForEach(node, filter)
	})
}

// Any returns deeply visiting node set.
func (p NodeSet) Any() (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &anyNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type childNodes struct {
	data NodeEnum
}

func (p *childNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		return node.ForEach(filter)
	})
}

// Child returns child node set.
func (p NodeSet) Child() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &childNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type bodyNodes struct {
	data NodeEnum
}

func (p *bodyNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astDecl:
			if fn, ok := node.Decl.(*ast.FuncDecl); ok && fn.Body != nil {
				return filter(&astStmt{fn.Body})
			}
		}
		return ErrNotFound
	})
}

// Body returns body node set.
func (p NodeSet) Body() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &bodyNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type xNodes struct {
	data NodeEnum
}

func (p *xNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astStmt:
			switch stmt := node.Stmt.(type) {
			case *ast.ExprStmt:
				return filter(&astExpr{stmt.X})
			}
		case *astExpr:
			switch expr := node.Expr.(type) {
			case *ast.SelectorExpr:
				return filter(&astExpr{expr.X})
			case *ast.UnaryExpr:
				return filter(&astExpr{expr.X})
			}
		}
		return ErrNotFound
	})
}

// X returns x node set.
func (p NodeSet) X() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &xNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type funNodes struct {
	data NodeEnum
}

func (p *funNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astExpr:
			switch expr := node.Expr.(type) {
			case *ast.CallExpr:
				return filter(&astExpr{expr.Fun})
			}
		}
		return ErrNotFound
	})
}

// Fun returns fun node set.
func (p NodeSet) Fun() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &funNodes{p.Data}}
}

// -----------------------------------------------------------------------------

type argNodes struct {
	data NodeEnum
	i    int
	varg bool
}

func (p *argNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astExpr:
			switch expr := node.Expr.(type) {
			case *ast.CallExpr:
				args := expr.Args
				if p.varg {
					for i, n := p.i, len(args); i < n; i++ {
						if err := filter(&astExpr{args[i]}); err == ErrBreak {
							return err
						}
					}
					return nil
				} else {
					return filter(&astExpr{args[p.i]})
				}
			}
		}
		return ErrNotFound
	})
}

// Arg returns args[i] node set.
func (p NodeSet) Arg(i int) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &argNodes{p.Data, i, false}}
}

// Varg returns args[i:] node set.
func (p NodeSet) Varg(i int) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &argNodes{p.Data, i, true}}
}

// -----------------------------------------------------------------------------

type ieltNodes struct {
	data NodeEnum
	i    int
}

func (p *ieltNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astExpr:
			switch expr := node.Expr.(type) {
			case *ast.CompositeLit:
				if i := p.i; i == -1 {
					for _, elt := range expr.Elts {
						if err := filter(&astExpr{elt}); err == ErrBreak {
							return err
						}
					}
				} else {
					return filter(&astExpr{expr.Elts[i]})
				}
			}
		}
		return ErrNotFound
	})
}

// Elt returns elts[i] node set.
func (p NodeSet) Elt__0(i int) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &ieltNodes{p.Data, i}}
}

// Elt returns elts[:] node set.
func (p NodeSet) Elt__1() NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &ieltNodes{p.Data, -1}}
}

// -----------------------------------------------------------------------------

type eltNodes struct {
	data NodeEnum
	name string
}

func (p *eltNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astExpr:
			switch expr := node.Expr.(type) {
			case *ast.CompositeLit:
				if elt, ok := getElt(expr.Elts, p.name); ok {
					return filter(&astExpr{elt})
				}
			}
		}
		return ErrNotFound
	})
}

// Elt returns elts[name] node set.
func (p NodeSet) Elt__2(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &eltNodes{p.Data, name}}
}

// -----------------------------------------------------------------------------

type rhsNodes struct {
	data NodeEnum
	i    int
}

func (p *rhsNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		switch node := node.(type) {
		case *astStmt:
			switch stmt := node.Stmt.(type) {
			case *ast.AssignStmt:
				return filter(&astExpr{stmt.Rhs[p.i]})
			}
		}
		return ErrNotFound
	})
}

// Rhs returns rhs[i] node set.
func (p NodeSet) Rhs(i int) NodeSet {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &rhsNodes{p.Data, i}}
}

// -----------------------------------------------------------------------------

type matchedNodes struct {
	data  NodeEnum
	match func(node Node) bool
}

func (p *matchedNodes) ForEach(filter func(node Node) error) error {
	return p.data.ForEach(func(node Node) error {
		if p.match(node) {
			return filter(node)
		}
		return ErrNotFound
	})
}

// Match filters the node set.
func (p NodeSet) Match(match func(node Node) bool) (ret NodeSet) {
	if p.Err != nil {
		return p
	}
	return NodeSet{Data: &matchedNodes{p.Data, match}}
}

// -----------------------------------------------------------------------------

// Name returns names of the node set.
func (p NodeSet) Name() []string {
	return p.ToString(NameOf)
}

// ToString returns string values of the node set.
func (p NodeSet) ToString(str func(node Node) string) (items []string) {
	if p.Err != nil {
		return nil
	}
	p.Data.ForEach(func(node Node) error {
		items = append(items, str(node))
		return nil
	})
	return
}

// Collect collects all nodes of the node set.
func (p NodeSet) Collect() (items []Node, err error) {
	if p.Err != nil {
		return nil, p.Err
	}
	p.Data.ForEach(func(node Node) error {
		items = append(items, node)
		return nil
	})
	return
}

// CollectOne collects one node of a node set.
// If exactly is true, it returns ErrTooManyNodes when node set is more than one.
func (p NodeSet) CollectOne__1(exactly bool) (item Node, err error) {
	if p.Err != nil {
		return nil, p.Err
	}
	err = ErrNotFound
	if exactly {
		p.Data.ForEach(func(node Node) error {
			if err == ErrNotFound {
				item, err = node, nil
				return nil
			}
			err = ErrTooManyNodes
			return ErrBreak
		})
	} else {
		p.Data.ForEach(func(node Node) error {
			item, err = node, nil
			return ErrBreak
		})
	}
	return
}

// CollectOne returns the first node.
func (p NodeSet) CollectOne__0() (item Node, err error) {
	return p.CollectOne__1(false)
}

// -----------------------------------------------------------------------------
