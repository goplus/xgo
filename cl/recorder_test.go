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

package cl

import (
	"go/types"
	"testing"

	"github.com/goplus/xgo/ast"
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
	sink := &typeRecordingRecorder{}
	rec := newRecorder(sink)
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
