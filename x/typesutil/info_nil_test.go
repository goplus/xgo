//go:build !genjs

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

package typesutil_test

import (
	"go/types"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
	"github.com/goplus/xgo/x/typesutil"
)

func TestInfoTypeOf(t *testing.T) {
	info := &typesutil.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	for range 2000 {
		info.Types[&ast.Ident{Name: "value"}] = types.TypeAndValue{Type: types.Typ[types.Int]}
	}
	if got := info.TypeOf(nil); got != nil {
		t.Fatalf("type of absent expression = %v, want nil", got)
	}
	nilLiteral := &ast.Ident{Name: "nil"}
	info.Types[nilLiteral] = types.TypeAndValue{Type: types.Typ[types.UntypedNil]}
	if got := info.TypeOf(nilLiteral); got != types.Typ[types.UntypedNil] {
		t.Fatalf("type of nil literal = %v, want untyped nil", got)
	}
}

func TestRecorderType(t *testing.T) {
	for _, tt := range []struct {
		name   string
		record bool
	}{
		{name: "Disabled"},
		{name: "GrownMap", record: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &typesutil.Info{}
			tv := types.TypeAndValue{Type: types.Typ[types.Int]}
			if tt.record {
				info.Types = make(map[ast.Expr]types.TypeAndValue)
				// Grow beyond one map table to exercise nil interface hashing on wasm.
				for range 2000 {
					info.Types[&ast.Ident{Name: "value"}] = tv
				}
			}
			rec := typesutil.NewRecorder(info)
			rec.Type(nil, tv)
			want := 0
			if tt.record {
				want = 2000
			}
			if len(info.Types) != want {
				t.Fatalf("type record count = %d, want %d", len(info.Types), want)
			}
			for expr := range info.Types {
				if expr == nil {
					t.Fatal("recorded an absent expression")
				}
			}
			nilLiteral := &ast.Ident{Name: "nil"}
			tv = types.TypeAndValue{Type: types.Typ[types.UntypedNil]}
			rec.Type(nilLiteral, tv)
			if tt.record && info.Types[nilLiteral] != tv {
				t.Fatal("nil literal type was not recorded")
			}
		})
	}
}

func TestInfoInferredCompositeLiterals(t *testing.T) {
	pkg, info, err := parseSource(token.NewFileSet(), "main.xgo", `type Point struct { X int }
var points map[string]Point = {"origin": {X: 1}}
var nullable map[string]*Point = {"empty": nil, "typed": (*Point)(nil)}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	point := pkg.Scope().Lookup("Point").Type()
	var inferred, structs, nilLiterals int
	for expr, tv := range info.Types {
		if expr == nil {
			t.Fatal("inferred literal type was recorded under an absent expression")
		}
		if literal, ok := expr.(*ast.CompositeLit); ok && literal.Type == nil {
			inferred++
			if tv.Type == nil || !tv.IsValue() {
				t.Fatalf("inferred literal has no value type: %v", tv)
			}
			if types.Identical(tv.Type, point) {
				structs++
			}
		}
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
			nilLiterals++
			if tv.Type == nil {
				t.Fatal("nil literal lost its type")
			}
		}
	}
	if inferred < 3 || structs != 1 || nilLiterals != 2 {
		t.Fatalf("recorded %d inferred literals, %d Point literals, and %d nil literals", inferred, structs, nilLiterals)
	}
}
