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

package cl_test

import (
	"go/types"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl/cltest"
	"github.com/goplus/xgo/parser/fsx/memfs"
	"github.com/goplus/xgo/token"
)

type staticMemberUse struct {
	ident string
	obj   string
	line  int
}

type staticMemberRecorder struct {
	fset *token.FileSet
	uses []staticMemberUse
}

func (rec *staticMemberRecorder) Type(ast.Expr, types.TypeAndValue) {}

func (rec *staticMemberRecorder) Instantiate(*ast.Ident, types.Instance) {}

func (rec *staticMemberRecorder) Def(*ast.Ident, types.Object) {}

func (rec *staticMemberRecorder) Use(id *ast.Ident, obj types.Object) {
	if obj == nil {
		return
	}
	rec.uses = append(rec.uses, staticMemberUse{
		ident: id.Name,
		obj:   obj.Name(),
		line:  rec.fset.Position(id.Pos()).Line,
	})
}

func (rec *staticMemberRecorder) Implicit(ast.Node, types.Object) {}

func (rec *staticMemberRecorder) Select(*ast.SelectorExpr, *types.Selection) {}

func (rec *staticMemberRecorder) Scope(ast.Node, *types.Scope) {}

func (rec *staticMemberRecorder) hasUse(ident, obj string, line int) bool {
	for _, use := range rec.uses {
		if use.ident == ident && use.obj == obj && use.line == line {
			return true
		}
	}
	return false
}

func TestStaticMemberSelectorRecorderUsesReceiver(t *testing.T) {
	rec := &staticMemberRecorder{fset: cltest.Conf.Fset}
	conf := *cltest.Conf
	conf.Recorder = rec

	fs := memfs.SingleFile("/foo", "bar.xgo", `
type foo int

const foo.name = "xgo"
var foo.count int = 100

a := foo.name
foo.count++
`)
	cltest.DoFS(t, &conf, fs, "/foo", nil, "main", nil)

	for _, want := range []staticMemberUse{
		{ident: "foo", obj: "foo", line: 7},
		{ident: "name", obj: "XGos_foo_Name", line: 7},
		{ident: "foo", obj: "foo", line: 8},
		{ident: "count", obj: "XGos_foo_Count", line: 8},
	} {
		if !rec.hasUse(want.ident, want.obj, want.line) {
			t.Fatalf("missing recorder use ident=%q obj=%q line=%d in %#v", want.ident, want.obj, want.line, rec.uses)
		}
	}
}

func TestClassfileStaticMemberRecorderUsesBareMember(t *testing.T) {
	rec := &staticMemberRecorder{fset: cltest.Conf.Fset}
	conf := *cltest.Conf
	conf.Recorder = rec

	fs := memfs.SingleFile("/foo", "Rect.gox", `
const .name = "rect"

func Get() string {
	return name
}
`)
	cltest.DoFS(t, &conf, fs, "/foo", nil, "main", nil)

	if rec.hasUse("Rect", "Rect", 4) {
		t.Fatalf("bare static member should not record explicit receiver use: %#v", rec.uses)
	}
	if !rec.hasUse("name", "XGos_Rect_Name", 5) {
		t.Fatalf("missing static member use in %#v", rec.uses)
	}
}
