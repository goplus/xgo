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

package cl

import (
	"go/types"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

// inferEnumUnderlyingType infers the underlying Go type for an enum declaration
// by statically analyzing the first value expression in the enum block.
//
// Rules (matching Go's constant default type rules):
//
//	iota or integer literal  → int
//	string literal           → string
//	bool literal             → bool
//	float literal            → float64
//	char literal             → rune (int32)
//	T(expr) type conversion  → the named type T (if it resolves to a basic type)
//	binary/unary/paren expr  → delegate to first operand
//
// Returns nil when the type cannot be determined statically; the caller should
// report an error in that case.
func inferEnumUnderlyingType(ctx *blockCtx, specs []ast.Spec) types.Type {
	// Find the first spec that carries an explicit value.
	for _, spec := range specs {
		vs := spec.(*ast.ValueSpec)
		if len(vs.Values) == 0 {
			// No explicit value on this spec — continue to the next.
			continue
		}
		return inferConstExprType(ctx, vs.Values[0])
	}
	// All specs omit values (only possible when iota is implied), so the
	// default type is int.
	return types.Typ[types.Int]
}

// inferConstExprType recursively inspects expr to determine its Go default type.
func inferConstExprType(ctx *blockCtx, expr ast.Expr) types.Type {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return basicLitDefaultType(e.Kind)
	case *ast.Ident:
		switch e.Name {
		case "iota":
			return types.Typ[types.Int]
		case "true", "false":
			return types.Typ[types.Bool]
		}
		// Named constant — we cannot resolve it statically here; fall through.
		return nil
	case *ast.CallExpr:
		// Explicit type conversion: T(expr)
		if ident, ok := e.Fun.(*ast.Ident); ok {
			if t := lookupBasicType(ident.Name); t != nil {
				return t
			}
		}
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			// e.g. pkg.Type(expr) — not a basic type, but may be a named type in scope.
			// We handle only the simple pkg.Type form here.
			_ = sel
		}
		return nil
	case *ast.BinaryExpr:
		// For binary expressions, the default type is determined by the left operand.
		return inferConstExprType(ctx, e.X)
	case *ast.UnaryExpr:
		return inferConstExprType(ctx, e.X)
	case *ast.ParenExpr:
		return inferConstExprType(ctx, e.X)
	}
	return nil
}

// basicLitDefaultType maps a literal token kind to its Go default type.
func basicLitDefaultType(kind token.Token) types.Type {
	switch kind {
	case token.INT:
		return types.Typ[types.Int]
	case token.FLOAT:
		return types.Typ[types.Float64]
	case token.IMAG:
		return types.Typ[types.Complex128]
	case token.CHAR:
		return types.Typ[types.Rune]
	case token.STRING:
		return types.Typ[types.String]
	}
	return nil
}

// lookupBasicType resolves a type name to a *types.Basic if it is one of the
// standard Go predeclared integer/float/string/bool types.
var lookupBasicType = func(name string) types.Type {
	switch name {
	case "int":
		return types.Typ[types.Int]
	case "int8":
		return types.Typ[types.Int8]
	case "int16":
		return types.Typ[types.Int16]
	case "int32":
		return types.Typ[types.Int32]
	case "int64":
		return types.Typ[types.Int64]
	case "uint":
		return types.Typ[types.Uint]
	case "uint8":
		return types.Typ[types.Uint8]
	case "uint16":
		return types.Typ[types.Uint16]
	case "uint32":
		return types.Typ[types.Uint32]
	case "uint64":
		return types.Typ[types.Uint64]
	case "uintptr":
		return types.Typ[types.Uintptr]
	case "float32":
		return types.Typ[types.Float32]
	case "float64":
		return types.Typ[types.Float64]
	case "complex64":
		return types.Typ[types.Complex64]
	case "complex128":
		return types.Typ[types.Complex128]
	case "string":
		return types.Typ[types.String]
	case "bool":
		return types.Typ[types.Bool]
	case "byte":
		return types.Typ[types.Byte]
	case "rune":
		return types.Typ[types.Rune]
	}
	return nil
}

// loadEnumConsts loads all constants declared in an EnumType block with the
// given named enum type. It must be called after the named type has been fully
// initialized via InitType.
func loadEnumConsts(ctx *blockCtx, enumType *ast.EnumType, namedTyp types.Type) {
	pkg := ctx.pkg
	cdecl := pkg.NewConstDefs(pkg.Types.Scope())
	for iotav, spec := range enumType.Specs {
		vs := spec.(*ast.ValueSpec)
		loadEnumConst(ctx, cdecl, vs, iotav, namedTyp)
	}
}

func loadEnumConst(ctx *blockCtx, cdecl *gogen.ConstDefs, v *ast.ValueSpec, iotav int, namedTyp types.Type) {
	vNames := v.Names
	names := makeNames(vNames)
	if v.Values == nil {
		if debugLoad {
			_ = names // silence unused warning
		}
		cdecl.Next(iotav, v.Pos(), names...)
		defNames(ctx, vNames, nil)
		return
	}
	fn := func(cb *gogen.CodeBuilder) int {
		for _, val := range v.Values {
			compileExpr(ctx, 1, val)
		}
		return len(v.Values)
	}
	cdecl.New(fn, iotav, v.Pos(), namedTyp, names...)
	defNames(ctx, v.Names, nil)
}
