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
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl/internal/typesutil"
)

type recordedType struct {
	expr ast.Expr
	tv   types.TypeAndValue
}

type typeRecordingRecorder struct {
	Recorder
	records []recordedType
}

func (r *typeRecordingRecorder) Type(expr ast.Expr, tv types.TypeAndValue) {
	r.records = append(r.records, recordedType{expr, tv})
}

func TestGoxRecorderType(t *testing.T) {
	for _, tt := range []struct {
		name   string
		shared bool
	}{
		{name: "Private"},
		{name: "Shared", shared: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &typeRecordingRecorder{}
			var output Recorder = sink
			if tt.shared {
				shared := &sharedTypeRecordingRecorder{types: make(map[ast.Expr]types.TypeAndValue)}
				sink = &shared.typeRecordingRecorder
				output = shared
			}
			rec := newRecorder(output)
			if rec.sharesTypes != tt.shared {
				t.Fatalf("shared types = %t, want %t", rec.sharesTypes, tt.shared)
			}
			tv := types.TypeAndValue{Type: types.Typ[types.Int]}
			for range 2000 {
				rec.Type(&ast.Ident{Name: "value"}, tv)
			}

			rec.Type(nil, tv)
			if len(rec.types) != 2000 || len(sink.records) != 2000 {
				t.Fatalf("recording an absent expression changed the records: %d internal, %d forwarded", len(rec.types), len(sink.records))
			}
			for expr := range rec.types {
				if expr == nil {
					t.Fatal("recorded an absent expression")
				}
			}

			expr := &ast.Ident{Name: "nil"}
			tv = types.TypeAndValue{Type: types.Typ[types.UntypedNil]}
			rec.Type(expr, tv)
			if got := rec.types[expr]; got != tv {
				t.Fatalf("nil literal type = %v, want %v", got, tv)
			}
			if got := sink.records[len(sink.records)-1]; got.expr != expr || got.tv != tv {
				t.Fatalf("forwarded nil literal = %v, want %v", got, recordedType{expr, tv})
			}
		})
	}
}

func TestGoxRecorderRecordCompositeLit(t *testing.T) {
	for _, tt := range []struct {
		name string
		typ  types.Type
		node ast.Expr
	}{
		{name: "InferredMap", typ: types.NewMap(types.Typ[types.String], types.Typ[types.Int])},
		{name: "InferredSlice", typ: types.NewSlice(types.Typ[types.Int])},
		{name: "InferredStruct", typ: types.NewStruct(nil, nil)},
		{
			name: "ExplicitMap",
			typ:  types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
			node: &ast.MapType{Key: &ast.Ident{Name: "string"}, Value: &ast.Ident{Name: "int"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &typeRecordingRecorder{}
			rec := newRecorder(sink)
			for range 2000 {
				rec.Type(&ast.Ident{Name: "value"}, types.TypeAndValue{Type: types.Typ[types.Int]})
			}
			literal := &ast.CompositeLit{Type: tt.node}
			rec.recordCompositeLit(literal, tt.typ)
			if got := rec.types[literal]; got.Type != tt.typ || !got.IsValue() {
				t.Fatalf("literal type = %v, want a value of type %v", got, tt.typ)
			}
			want := 2001
			if tt.node != nil {
				want++
				if got := rec.types[tt.node]; got.Type != tt.typ || !got.IsType() {
					t.Fatalf("explicit type = %v, want type %v", got, tt.typ)
				}
			}
			if len(rec.types) != want || len(sink.records) != want {
				t.Fatalf("record count = %d internal, %d forwarded, want %d", len(rec.types), len(sink.records), want)
			}
			for expr := range rec.types {
				if expr == nil {
					t.Fatal("recorded the missing type expression of an inferred literal")
				}
			}
		})
	}
}

func TestGoxRecorderParenthesizedLHS(t *testing.T) {
	sink := &typeRecordingRecorder{}
	rec := newRecorder(sink)

	index := &ast.IndexExpr{}
	tv := typesutil.NewTypeAndValueForValue(types.Typ[types.Int], nil, typesutil.MapIndex)
	rec.Type(index, tv)
	paren := &ast.ParenExpr{X: index}
	rec.recordExpr(nil, paren, true)
	if got := rec.types[paren]; got != tv {
		t.Fatalf("parenthesized map index = %v, want %v", got, tv)
	}
	if len(sink.records) != 2 {
		t.Fatalf("forwarded %d types, want 2", len(sink.records))
	}
	if got := sink.records[1]; got.expr != paren || got.tv != tv {
		t.Fatalf("forwarded parenthesized map index = %v, want %v", got, recordedType{paren, tv})
	}

	blank := &ast.ParenExpr{X: &ast.Ident{Name: "_"}}
	rec.recordExpr(nil, blank, true)
	if _, ok := rec.types[blank]; ok || len(sink.records) != 2 {
		t.Fatal("recorded a type for a parenthesized blank identifier")
	}
}

type sharedTypeRecordingRecorder struct {
	typeRecordingRecorder
	types    map[ast.Expr]types.TypeAndValue
	previous types.TypeAndValue
}

func (r *sharedTypeRecordingRecorder) RecordedTypes() map[ast.Expr]types.TypeAndValue {
	return r.types
}

func (r *sharedTypeRecordingRecorder) Type(expr ast.Expr, tv types.TypeAndValue) {
	if r.types != nil {
		r.previous = r.types[expr]
		r.types[expr] = tv
	}
	r.typeRecordingRecorder.Type(expr, tv)
}

func TestGoxRecorderSharedTypes(t *testing.T) {
	t.Run("InstantiatedTypes", func(t *testing.T) {
		expr := &ast.Ident{Name: "Box"}
		sink := &sharedTypeRecordingRecorder{types: make(map[ast.Expr]types.TypeAndValue)}
		rec := newRecorder(sink)
		if !rec.sharesTypes {
			t.Fatal("an empty output map was not shared")
		}
		assertCompilerType := func(want types.TypeAndValue) {
			t.Helper()
			if got, ok := rec.recordedType(expr); !ok || got != want {
				t.Fatalf("compiler type = %v, want %v", got, want)
			}
		}
		original := typesutil.NewTypeAndValueForType(types.Typ[types.Int])
		rec.Type(expr, original)
		if sink.previous.Type != nil || sink.types[expr] != original {
			t.Fatal("the initial callback did not see an empty entry")
		}

		published := original
		for _, instance := range []types.Type{types.Typ[types.String], types.Typ[types.Bool]} {
			rec.instantiate(expr, original.Type, instance)
			if sink.previous != published {
				t.Fatalf("previous published type = %v, want %v", sink.previous, published)
			}
			published.Type = instance
			if got := sink.types[expr]; got != published {
				t.Fatalf("published instance = %v, want %v", got, published)
			}
			assertCompilerType(original)
		}

		replacement := typesutil.NewTypeAndValueForValue(types.Typ[types.Int], nil, typesutil.Variable)
		rec.Type(expr, replacement)
		if sink.previous != published {
			t.Fatalf("previous published type = %v, want %v", sink.previous, published)
		}
		assertCompilerType(replacement)
		rec.Complete(types.NewScope(nil, 0, 0, "test"))
		if rec.types != nil || rec.uninstantiatedTypes != nil {
			t.Fatal("retained compiler type maps after completion")
		}
		if got := sink.types[expr]; got != replacement {
			t.Fatalf("published type after completion = %v, want %v", got, replacement)
		}
	})

	for _, tt := range []struct {
		name      string
		populated bool
	}{
		{name: "DisabledOutput"},
		{name: "PopulatedOutput", populated: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			expr := &ast.Ident{Name: "value"}
			tv := typesutil.NewTypeAndValueForValue(types.Typ[types.Int], nil, typesutil.Variable)
			sink := &sharedTypeRecordingRecorder{}
			if tt.populated {
				sink.types = map[ast.Expr]types.TypeAndValue{expr: tv}
			}
			rec := newRecorder(sink)
			if _, ok := rec.recordedType(expr); ok {
				t.Fatal("the compiler read a type from before the check")
			}
			rec.Type(expr, tv)
			if got, ok := rec.recordedType(expr); !ok || got != tv {
				t.Fatalf("compiler type = %v, want %v", got, tv)
			}
		})
	}
}
