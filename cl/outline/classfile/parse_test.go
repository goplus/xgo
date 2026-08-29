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

package classfile

import (
	goast "go/ast"
	"testing"

	xast "github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

func TestParseDirectives(t *testing.T) {
	t.Run("ResourceDirectives", func(t *testing.T) {
		doc := &xast.CommentGroup{
			List: []*xast.Comment{
				{Text: "//xgo:class:resource sprite"},
				{Text: "//xgo:class:resource-discovery sprites.*"},
				{Text: "//xgo:class:resource-name-discovery id"},
			},
		}

		dirs := parseDirectives(doc)
		if len(dirs.resource) != 1 || dirs.resource[0].arg != "sprite" {
			t.Fatalf("unexpected resource directives: %#v", dirs.resource)
		}
		if len(dirs.discovery) != 1 || dirs.discovery[0].arg != "sprites.*" {
			t.Fatalf("unexpected discovery directives: %#v", dirs.discovery)
		}
		if len(dirs.nameDiscovery) != 1 || dirs.nameDiscovery[0].arg != "id" {
			t.Fatalf("unexpected name-discovery directives: %#v", dirs.nameDiscovery)
		}
	})

	t.Run("InvalidDirectiveSyntax", func(t *testing.T) {
		doc := &xast.CommentGroup{
			List: []*xast.Comment{
				{Text: "//xgo:class:resource"},
				{Text: "//xgo:class:resource-api-scope-binding param.x receiver"},
			},
		}

		dirs := parseDirectives(doc)
		if len(dirs.invalid) != 2 {
			t.Fatalf("unexpected invalid directives: %#v", dirs.invalid)
		}
	})
}

func TestParseResourceKind(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		ret, ok := parseResourceKind("sprite.costume_frame2")
		if !ok || ret != "sprite.costume_frame2" {
			t.Fatalf("unexpected parse result: %q, %v", ret, ok)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		for _, kind := range []string{"", "Sprite", "sprite..costume", "sprite.Costume"} {
			if _, ok := parseResourceKind(kind); ok {
				t.Fatalf("expected invalid kind: %q", kind)
			}
		}
	})
}

func TestParseScopeBinding(t *testing.T) {
	t.Run("ReceiverSource", func(t *testing.T) {
		target, source, ok := parseScopeBinding("param.0 receiver")
		if !ok || target != 0 || !source.Receiver || source.Param != 0 {
			t.Fatalf("unexpected scope binding: %d, %#v, %v", target, source, ok)
		}
	})

	t.Run("ParamSource", func(t *testing.T) {
		target, source, ok := parseScopeBinding("param.1 param.0")
		if !ok || target != 1 || source.Receiver || source.Param != 0 {
			t.Fatalf("unexpected scope binding: %d, %#v, %v", target, source, ok)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		for _, arg := range []string{"receiver param.0", "param.x receiver", "param.0"} {
			if _, _, ok := parseScopeBinding(arg); ok {
				t.Fatalf("expected invalid scope binding: %q", arg)
			}
		}
	})
}

func TestParseParam(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		ret, ok := parseParam("param.12")
		if !ok || ret != 12 {
			t.Fatalf("unexpected param parse result: %d, %v", ret, ok)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		for _, v := range []string{"param.", "param.01", "param.-1", "arg.0"} {
			if _, ok := parseParam(v); ok {
				t.Fatalf("expected invalid param: %q", v)
			}
		}
	})
}

func TestDeclDoc(t *testing.T) {
	t.Run("XGoDeclDoc", func(t *testing.T) {
		declDoc := &xast.CommentGroup{List: []*xast.Comment{{Text: "// decl", Slash: token.Pos(1)}}}
		specDoc := &xast.CommentGroup{List: []*xast.Comment{{Text: "// spec", Slash: token.Pos(2)}}}
		decl := &xast.GenDecl{Doc: declDoc}
		spec := &xast.TypeSpec{Doc: specDoc}
		if got := xgoDeclDoc(decl, spec); got != declDoc {
			t.Fatal("expected declaration doc to take precedence")
		}
	})

	t.Run("GoDeclDoc", func(t *testing.T) {
		declDoc := &goast.CommentGroup{List: []*goast.Comment{{Text: "// decl", Slash: 1}}}
		specDoc := &goast.CommentGroup{List: []*goast.Comment{{Text: "// spec", Slash: 2}}}
		decl := &goast.GenDecl{Doc: declDoc}
		spec := &goast.TypeSpec{Doc: specDoc}
		if got := goDeclDoc(decl, spec); got != (*xast.CommentGroup)(declDoc) {
			t.Fatal("expected declaration doc to take precedence")
		}
	})
}

func TestRecvBaseName(t *testing.T) {
	t.Run("XGo", func(t *testing.T) {
		recv := &xast.FieldList{List: []*xast.Field{{
			Type: &xast.StarExpr{X: &xast.ParenExpr{X: &xast.Ident{Name: "SpriteImpl"}}},
		}}}
		if got := xgoRecvBaseName(recv); got != "SpriteImpl" {
			t.Fatalf("unexpected XGo receiver base name: %q", got)
		}
	})

	t.Run("Go", func(t *testing.T) {
		recv := &goast.FieldList{List: []*goast.Field{{
			Type: &goast.StarExpr{X: &goast.ParenExpr{X: &goast.Ident{Name: "SpriteImpl"}}},
		}}}
		if got := goRecvBaseName(recv); got != "SpriteImpl" {
			t.Fatalf("unexpected Go receiver base name: %q", got)
		}
	})
}
