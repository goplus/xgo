/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package cltest

import (
	"go/types"

	"github.com/goplus/xgo/ast"
)

type gopRecorder struct {
}

// Type maps expressions to their types, and for constant
// expressions, also their values. Invalid expressions are
// omitted.
//
// For (possibly parenthesized) identifiers denoting built-in
// functions, the recorded signatures are call-site specific:
// if the call result is not a constant, the recorded type is
// an argument-specific signature. Otherwise, the recorded type
// is invalid.
//
// The Types map does not record the type of every identifier,
// only those that appear where an arbitrary expression is
// permitted. For instance, the identifier f in a selector
// expression x.f is found only in the Selections map, the
// identifier z in a variable declaration 'var z int' is found
// only in the Defs map, and identifiers denoting packages in
// qualified identifiers are collected in the Uses map.
func (info gopRecorder) Type(e ast.Expr, tv types.TypeAndValue) {
}

// Instantiate maps identifiers denoting generic types or functions to their
// type arguments and instantiated type.
//
// For example, Instantiate will map the identifier for 'T' in the type
// instantiation T[int, string] to the type arguments [int, string] and
// resulting instantiated *Named type. Given a generic function
// func F[A any](A), Instances will map the identifier for 'F' in the call
// expression F(int(1)) to the inferred type arguments [int], and resulting
// instantiated *Signature.
//
// Invariant: Instantiating Uses[id].Type() with Instances[id].TypeArgs
// results in an equivalent of Instances[id].Type.
func (info gopRecorder) Instantiate(id *ast.Ident, inst types.Instance) {
}

// Def maps identifiers to the objects they define (including
// package names, dots "." of dot-imports, and blank "_" identifiers).
// For identifiers that do not denote objects (e.g., the package name
// in package clauses, or symbolic variables t in t := x.(type) of
// type switch headers), the corresponding objects are nil.
//
// For an embedded field, Def maps the field *Var it defines.
//
// Invariant: Defs[id] == nil || Defs[id].Pos() == id.Pos()
func (info gopRecorder) Def(id *ast.Ident, obj types.Object) {
}

// Use maps identifiers to the objects they denote.
//
// For an embedded field, Use maps the *TypeName it denotes.
//
// Invariant: Uses[id].Pos() != id.Pos()
func (info gopRecorder) Use(id *ast.Ident, obj types.Object) {
}

// Implicit maps nodes to their implicitly declared objects, if any.
// The following node and object types may appear:
//
//	node               declared object
//
//	*ast.ImportSpec    *PkgName for imports without renames
//	*ast.CaseClause    type-specific *Var for each type switch case clause (incl. default)
//	*ast.Field         anonymous parameter *Var (incl. unnamed results)
func (info gopRecorder) Implicit(node ast.Node, obj types.Object) {
}

// Select maps selector expressions (excluding qualified identifiers)
// to their corresponding selections.
func (info gopRecorder) Select(e *ast.SelectorExpr, sel *types.Selection) {
}

// Scope maps ast.Nodes to the scopes they define. Package scopes are not
// associated with a specific node but with all files belonging to a package.
// Thus, the package scope can be found in the type-checked Package object.
// Scopes nest, with the Universe scope being the outermost scope, enclosing
// the package scope, which contains (one or more) files scopes, which enclose
// function scopes which in turn enclose statement and function literal scopes.
// Note that even though package-level functions are declared in the package
// scope, the function scopes are embedded in the file scope of the file
// containing the function declaration.
//
// The following node types may appear in Scopes:
//
//	*ast.File
//	*ast.FuncType
//	*ast.TypeSpec
//	*ast.BlockStmt
//	*ast.IfStmt
//	*ast.SwitchStmt
//	*ast.TypeSwitchStmt
//	*ast.CaseClause
//	*ast.CommClause
//	*ast.ForStmt
//	*ast.RangeStmt
func (info gopRecorder) Scope(n ast.Node, scope *types.Scope) {
}

func (info gopRecorder) Builtin(name string, obj types.Object) {
}
