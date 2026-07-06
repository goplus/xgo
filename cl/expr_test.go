//go:build !genjs

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

package cl

import (
	"go/types"
	"testing"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

func TestTryProjectWorkClassMemberFallbackGuards(t *testing.T) {
	for _, tt := range []struct {
		name string
		load func(scope *types.Scope, pkg *types.Package) loader
	}{
		{name: "MissingSymbol"},
		{
			name: "LoadedSymbolIsNotTypeName",
			load: func(scope *types.Scope, pkg *types.Package) loader {
				return &typeLoader{typ: func() {
					scope.Insert(types.NewVar(token.NoPos, pkg, "Beta", types.Typ[types.Int]))
				}}
			},
		},
		{
			name: "LoadedTypeNameIsNotNamed",
			load: func(scope *types.Scope, pkg *types.Package) loader {
				return &typeLoader{typ: func() {
					scope.Insert(types.NewTypeName(token.NoPos, pkg, "Beta", types.Typ[types.Int]))
				}}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pkg := gogen.NewPackage("", "foo", goxConf)
			scope := pkg.Types.Scope()
			syms := make(map[string]loader)
			if tt.load != nil {
				syms["Beta"] = tt.load(scope, pkg.Types)
			}
			ctx := &blockCtx{
				pkgCtx: &pkgCtx{syms: syms},
				proj: &classProject{
					gameClass_: "Game",
					works: []*workClass{{
						feats: workClassEmbedded,
						types: []string{"Beta"},
					}},
				},
				pkg: pkg,
				cb:  pkg.CB(),
			}
			recvType := types.NewNamed(
				types.NewTypeName(token.NoPos, pkg.Types, "Game", nil),
				types.NewStruct(nil, nil),
				nil,
			)
			recv := types.NewVar(token.NoPos, pkg.Types, "this", types.NewPointer(recvType))
			if tryProjectWorkClassMember(ctx, recv, ast.NewIdent("Beta"), 0) {
				t.Fatal("tryProjectWorkClassMember succeeded unexpectedly")
			}
		})
	}
}

func TestTryProjectWorkClassMemberRecordsMember(t *testing.T) {
	pkg := gogen.NewPackage("", "foo", goxConf)
	betaNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg.Types, "Beta", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	rec := newExprRecorder()
	ctx := &blockCtx{
		pkgCtx: &pkgCtx{syms: map[string]loader{
			"Beta": &typeLoader{typ: func() {
				pkg.Types.Scope().Insert(betaNamed.Obj())
			}},
		}},
		proj: &classProject{
			gameClass_: "Game",
			works: []*workClass{{
				feats: workClassEmbedded,
				types: []string{"Beta"},
			}},
		},
		pkg:       pkg,
		cb:        pkg.CB(),
		rec:       newRecorder(rec),
		isXgoFile: true,
	}
	recvType := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg.Types, "Game", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	recv := types.NewVar(token.NoPos, pkg.Types, "this", types.NewPointer(recvType))
	ident := ast.NewIdent("Beta")
	if !tryProjectWorkClassMember(ctx, recv, ident, 0) {
		t.Fatal("tryProjectWorkClassMember failed")
	}
	obj, ok := rec.uses[ident]
	if !ok {
		t.Fatal("missing recorded use")
	}
	field, ok := obj.(*types.Var)
	if !ok {
		t.Fatalf("recorded use = %T, want *types.Var", obj)
	}
	if field.Name() != "Beta" {
		t.Fatalf("recorded use name = %q, want Beta", field.Name())
	}
	wantType := types.NewPointer(betaNamed)
	if !types.Identical(field.Type(), wantType) {
		t.Fatalf("recorded use type = %v, want %v", field.Type(), wantType)
	}
	tv, ok := rec.types[ident]
	if !ok {
		t.Fatal("missing recorded type")
	}
	if !types.Identical(tv.Type, wantType) {
		t.Fatalf("recorded type = %v, want %v", tv.Type, wantType)
	}
	if !tv.IsValue() || !tv.Addressable() {
		t.Fatalf("recorded type = %v, want addressable value", tv)
	}
}

type exprRecorder struct {
	types map[ast.Expr]types.TypeAndValue
	uses  map[*ast.Ident]types.Object
}

func newExprRecorder() *exprRecorder {
	return &exprRecorder{
		types: make(map[ast.Expr]types.TypeAndValue),
		uses:  make(map[*ast.Ident]types.Object),
	}
}

func (rec *exprRecorder) Type(e ast.Expr, tv types.TypeAndValue) {
	rec.types[e] = tv
}

func (rec *exprRecorder) Instantiate(*ast.Ident, types.Instance) {
}

func (rec *exprRecorder) Def(*ast.Ident, types.Object) {
}

func (rec *exprRecorder) Use(id *ast.Ident, obj types.Object) {
	rec.uses[id] = obj
}

func (rec *exprRecorder) Implicit(ast.Node, types.Object) {
}

func (rec *exprRecorder) Select(*ast.SelectorExpr, *types.Selection) {
}

func (rec *exprRecorder) Scope(ast.Node, *types.Scope) {
}
