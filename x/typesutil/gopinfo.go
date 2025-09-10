/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package typesutil

import (
	"fmt"
	"go/types"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/token"
	"github.com/qiniu/x/log"
)

// -----------------------------------------------------------------------------

// Info holds result type information for a type-checked package.
// Only the information for which a map is provided is collected.
// If the package has type errors, the collected information may
// be incomplete.
type Info struct {
	// Types maps expressions to their types, and for constant
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
	Types map[ast.Expr]types.TypeAndValue

	// Instances maps identifiers denoting generic types or functions to their
	// type arguments and instantiated type.
	//
	// For example, Instances will map the identifier for 'T' in the type
	// instantiation T[int, string] to the type arguments [int, string] and
	// resulting instantiated *Named type. Given a generic function
	// func F[A any](A), Instances will map the identifier for 'F' in the call
	// expression F(int(1)) to the inferred type arguments [int], and resulting
	// instantiated *Signature.
	//
	// Invariant: Instantiating Uses[id].Type() with Instances[id].TypeArgs
	// results in an equivalent of Instances[id].Type.
	Instances map[*ast.Ident]types.Instance

	// Defs maps identifiers to the objects they define (including
	// package names, dots "." of dot-imports, and blank "_" identifiers).
	// For identifiers that do not denote objects (e.g., the package name
	// in package clauses, or symbolic variables t in t := x.(type) of
	// type switch headers), the corresponding objects are nil.
	//
	// For an embedded field, Defs returns the field *Var it defines.
	//
	// Invariant: Defs[id] == nil || Defs[id].Pos() == id.Pos()
	Defs map[*ast.Ident]types.Object

	// Uses maps identifiers to the objects they denote.
	//
	// For an embedded field, Uses returns the *TypeName it denotes.
	//
	// Invariant: Uses[id].Pos() != id.Pos()
	Uses map[*ast.Ident]types.Object

	// Implicits maps nodes to their implicitly declared objects, if any.
	// The following node and object types may appear:
	//
	//     node               declared object
	//
	//     *ast.ImportSpec    *PkgName for imports without renames
	//     *ast.CaseClause    type-specific *Var for each type switch case clause (incl. default)
	//     *ast.Field         anonymous parameter *Var (incl. unnamed results)
	//     *ast.FunLit        function literal in *ast.OverloadFuncDecl
	//
	Implicits map[ast.Node]types.Object

	// Selections maps selector expressions (excluding qualified identifiers)
	// to their corresponding selections.
	Selections map[*ast.SelectorExpr]*types.Selection

	// Scopes maps ast.Nodes to the scopes they define. Package scopes are not
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
	//     *ast.File
	//     *ast.FuncType
	//     *ast.TypeSpec
	//     *ast.BlockStmt
	//     *ast.IfStmt
	//     *ast.SwitchStmt
	//     *ast.TypeSwitchStmt
	//     *ast.CaseClause
	//     *ast.CommClause
	//     *ast.ForStmt
	//     *ast.RangeStmt
	//     *ast.ForPhraseStmt
	//     *ast.ForPhrase
	//     *ast.LambdaExpr
	//     *ast.LambdaExpr2
	//
	Scopes map[ast.Node]*types.Scope

	// InitOrder is the list of package-level initializers in the order in which
	// they must be executed. Initializers referring to variables related by an
	// initialization dependency appear in topological order, the others appear
	// in source order. Variables without an initialization expression do not
	// appear in this list.
	// InitOrder []*Initializer

	// Overloads maps identifiers to the overload decl object.
	Overloads map[*ast.Ident]types.Object

	// Builtins maps identifiers the Go+ builtin name object.
	Builtins map[string]types.Object
}

// ObjectOf returns the object denoted by the specified id,
// or nil if not found.
//
// If id is an embedded struct field, ObjectOf returns the field (*Var)
// it defines, not the type (*TypeName) it uses.
//
// Precondition: the Uses and Defs maps are populated.
func (info *Info) ObjectOf(id *ast.Ident) types.Object {
	if obj := info.Defs[id]; obj != nil {
		return obj
	}
	return info.Uses[id]
}

// TypeOf returns the type of expression e, or nil if not found.
// Precondition: the Types, Uses and Defs maps are populated.
func (info *Info) TypeOf(e ast.Expr) types.Type {
	if t, ok := info.Types[e]; ok {
		return t.Type
	}
	if id, _ := e.(*ast.Ident); id != nil {
		if obj := info.ObjectOf(id); obj != nil {
			return obj.Type()
		}
	}
	return nil
}

// Returns the overloaded function declaration corresponding to the ident and its overloaded function members
func (info *Info) OverloadOf(id *ast.Ident) (types.Object, []types.Object) {
	if obj := info.Overloads[id]; obj != nil {
		if sig, ok := obj.Type().(*types.Signature); ok {
			if _, objs := gogen.CheckSigFuncExObjects(sig); len(objs) > 1 {
				return obj, objs
			}
		}
	}
	return nil, nil
}

// -----------------------------------------------------------------------------

type xgoRecorder struct {
	*Info
}

// NewRecorder creates a new recorder for cl.NewPackage.
func NewRecorder(info *Info) cl.Recorder {
	if info == nil {
		return nil
	}
	return xgoRecorder{info}
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
func (info xgoRecorder) Type(e ast.Expr, tv types.TypeAndValue) {
	if debugVerbose {
		log.Println("==> Type:", e, tv.Type)
	}
	if info.Types != nil {
		info.Types[e] = tv
	}
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
func (info xgoRecorder) Instantiate(id *ast.Ident, inst types.Instance) {
	if info.Instances != nil {
		info.Instances[id] = inst
	}
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
func (info xgoRecorder) Def(id *ast.Ident, obj types.Object) {
	if debugVerbose {
		log.Println("==> Def:", id, obj)
	}
	if info.Defs != nil {
		info.Defs[id] = obj
	}
}

// Use maps identifiers to the objects they denote.
//
// For an embedded field, Use maps the *TypeName it denotes.
//
// Invariant: Uses[id].Pos() != id.Pos()
func (info xgoRecorder) Use(id *ast.Ident, obj types.Object) {
	if debugVerbose {
		log.Println("==> Use:", id, obj)
	}
	if info.Uses != nil {
		info.Uses[id] = obj
	}
	if info.Overloads != nil {
		if sig, ok := obj.Type().(*types.Signature); ok {
			if ext, ok := gogen.CheckSigFuncEx(sig); ok {
				if debugVerbose {
					log.Println("==> Overloads:", id, ext)
				}
				info.Overloads[id] = obj
			}
		}
	}
}

// Implicit maps nodes to their implicitly declared objects, if any.
// The following node and object types may appear:
//
//	node               declared object
//
//	*ast.ImportSpec    *PkgName for imports without renames
//	*ast.CaseClause    type-specific *Var for each type switch case clause (incl. default)
//	*ast.Field         anonymous parameter *Var (incl. unnamed results)
func (info xgoRecorder) Implicit(node ast.Node, obj types.Object) {
	if debugVerbose {
		log.Println("==> Implicit:", obj)
	}
	if info.Implicits != nil {
		info.Implicits[node] = obj
	}
}

// Select maps selector expressions (excluding qualified identifiers)
// to their corresponding selections.
func (info xgoRecorder) Select(e *ast.SelectorExpr, sel *types.Selection) {
	if info.Selections != nil {
		info.Selections[e] = sel
	}
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
func (info xgoRecorder) Scope(n ast.Node, scope *types.Scope) {
	if debugVerbose {
		log.Println("==> Scope:", scope)
	}
	if info.Scopes != nil {
		info.Scopes[n] = scope
	}
}

// Go+ builtin name object
func (info xgoRecorder) Builtin(name string, obj types.Object) {
	if info.Builtins != nil {
		info.Builtins[name] = obj
	}
}

// -----------------------------------------------------------------------------

// An Error describes a type-checking error; it implements the error interface.
// A "soft" error is an error that still permits a valid interpretation of a
// package (such as "unused variable"); "hard" errors may lead to unpredictable
// behavior if ignored.
type Error struct {
	Fset     *token.FileSet // file set for interpretation of Pos
	Pos, End token.Pos      // error position
	Msg      string         // error message
	Code     Code           // error code
	Soft     bool           // if set, error is "soft"
}

// Error returns an error string formatted as follows:
// filename:line:column: message
func (err Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Fset.Position(err.Pos), err.Msg)
}
