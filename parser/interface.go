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

// This file contains the exported entry points for invoking the parser.

package parser

import (
	goparser "go/parser"
	"go/scanner"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser/iox"
	"github.com/goplus/xgo/token"
)

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
type Mode uint

const (
	// PackageClauseOnly - stop parsing after package clause
	PackageClauseOnly = Mode(goparser.PackageClauseOnly)
	// ImportsOnly - stop parsing after import declarations
	ImportsOnly = Mode(goparser.ImportsOnly)
	// ParseComments - parse comments and add them to AST
	ParseComments = Mode(goparser.ParseComments)
	// Trace - print a trace of parsed productions
	Trace = Mode(goparser.Trace)
	// DeclarationErrors - report declaration errors
	DeclarationErrors = Mode(goparser.DeclarationErrors)
	// AllErrors - report all errors (not just the first 10 on different lines)
	AllErrors = Mode(goparser.AllErrors)

	// ParseGoAsXGo - parse Go files by xgo/parser
	ParseGoAsXGo Mode = 1 << 16
	// ParseXGoClass - parse XGo classfile by xgo/parser
	ParseXGoClass Mode = 1 << 17
	// SaveAbsFile - parse and save absolute path to pkg.Files
	SaveAbsFile Mode = 1 << 18

	// for backward compatibility
	ParseGoAsGoPlus = ParseGoAsXGo
	// for backward compatibility
	ParseGoPlusClass = ParseXGoClass

	goReservedFlags Mode = ((1 << 16) - 1)
)

// -----------------------------------------------------------------------------

// ParseFile parses the source code of a single Go source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
//
// If src != nil, ParseFile parses the source from src and the filename is
// only used when recording position information. The type of the argument
// for the src parameter must be string, []byte, or io.Reader.
// If src == nil, ParseFile parses the file specified by filename.
//
// The mode parameter controls the amount of source text parsed and other
// optional parser functionality. Position information is recorded in the
// file set fset, which must not be nil.
//
// If the source couldn't be read, the returned AST is nil and the error
// indicates the specific failure. If the source was read but syntax
// errors were found, the result is a partial AST (with ast.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors
// are returned via a scanner.ErrorList which is sorted by source position.
func parseFile(fset *token.FileSet, filename string, src any, mode Mode) (f *ast.File, err error) {
	if fset == nil {
		panic("parser.ParseFile: no token.FileSet provided (fset == nil)")
	}

	// get source
	text, err := iox.ReadSourceLocal(filename, src)
	if err != nil {
		return
	}

	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		// set result values
		if f == nil {
			// source is not a valid Go source file - satisfy
			// ParseFile API and return a valid (but) empty *ast.File
			f = &ast.File{
				Name: new(ast.Ident),
			}
		}
		f.Code = text

		p.errors.Sort()
		err = p.errors.Err()
	}()

	// parse source
	p.init(fset, filename, text, mode)
	f = p.parseFile()

	return
}

// -----------------------------------------------------------------------------

// ParseExprFrom is a convenience function for parsing an expression.
// The arguments have the same meaning as for ParseFile, but the source must
// be a valid Go/XGo (type or value) expression. Specifically, fset must not
// be nil.
//
// If the source couldn't be read, the returned AST is nil and the error
// indicates the specific failure. If the source was read but syntax
// errors were found, the result is a partial AST (with ast.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors
// are returned via a scanner.ErrorList which is sorted by source position.
func ParseExprFrom(fset *token.FileSet, filename string, src any, mode Mode) (expr ast.Expr, err error) {
	// get source
	text, err := iox.ReadSourceLocal(filename, src)
	if err != nil {
		return
	}

	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}
		p.errors.Sort()
		err = p.errors.Err()
	}()

	// parse expr
	p.init(fset, filename, text, mode)
	expr = p.parseRHS()

	// If a semicolon was inserted, consume it;
	// report an error if there's more tokens.
	if p.tok == token.SEMICOLON && p.lit == "\n" {
		p.next()
	}
	p.expect(token.EOF)

	return
}

// ParseExprEx is a convenience function for parsing an expression.
// The arguments have the same meaning as for ParseFile, but the source must
// be a valid Go/XGo (type or value) expression. Specifically, fset must not
// be nil.
//
// If the source couldn't be read, the returned AST is nil and the error
// indicates the specific failure. If the source was read but syntax
// errors were found, the result is a partial AST (with ast.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors
// are returned via a scanner.ErrorList which is sorted by source position.
func ParseExprEx(file *token.File, src []byte, offset int, mode Mode) (expr ast.Expr, err scanner.ErrorList) {
	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}
		err = p.errors
	}()

	// parse expr
	p.initSub(file, src, offset, mode)
	expr = p.parseRHS()

	// If a semicolon was inserted, consume it;
	// report an error if there's more tokens.
	if p.tok == token.SEMICOLON && p.lit == "\n" {
		p.next()
	}
	p.expect(token.EOF)

	return
}

// ParseExpr is a convenience function for obtaining the AST of an expression x.
// The position information recorded in the AST is undefined. The filename used
// in error messages is the empty string.
//
// If syntax errors were found, the result is a partial AST (with ast.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors are
// returned via a scanner.ErrorList which is sorted by source position.
func ParseExpr(x string) (ast.Expr, error) {
	return ParseExprFrom(token.NewFileSet(), "", []byte(x), 0)
}

// -----------------------------------------------------------------------------
