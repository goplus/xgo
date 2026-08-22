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
				{Text: "//xgo:class:resource-api-scope-binding param.1 param.0"},
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
		if len(dirs.apiScopeBindings) != 1 || dirs.apiScopeBindings[0].target != 1 || dirs.apiScopeBindings[0].source.Param != 0 {
			t.Fatalf("unexpected API scope-binding directives: %#v", dirs.apiScopeBindings)
		}
	})

	t.Run("ArgumentWhitespace", func(t *testing.T) {
		doc := &xast.CommentGroup{
			List: []*xast.Comment{{Text: "//xgo:class:resource\tsprite"}},
		}

		dirs := parseDirectives(doc)
		if len(dirs.resource) != 1 || dirs.resource[0].arg != "sprite" {
			t.Fatalf("unexpected resource directives: %#v", dirs.resource)
		}
	})

	t.Run("WhitespaceAfterNamespace", func(t *testing.T) {
		doc := &xast.CommentGroup{
			List: []*xast.Comment{
				{Text: "//xgo:class: resource sprite"},
				{Text: "//xgo:class:\tresource sound"},
			},
		}

		dirs := parseDirectives(doc)
		if len(dirs.resource) != 0 || len(dirs.invalid) != 0 {
			t.Fatalf("unexpected directives: %#v", dirs)
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

func TestValidResourceKind(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		if !validResourceKind("sprite.costume_frame2") {
			t.Fatal("expected valid resource kind")
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		for _, kind := range []string{"", "Sprite", "sprite..costume", "sprite.Costume"} {
			if validResourceKind(kind) {
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

func TestXGoDeclDoc(t *testing.T) {
	declDoc := &xast.CommentGroup{List: []*xast.Comment{{Text: "// decl", Slash: token.Pos(1)}}}
	specDoc := &xast.CommentGroup{List: []*xast.Comment{{Text: "// spec", Slash: token.Pos(2)}}}
	spec := &xast.TypeSpec{Doc: specDoc}

	t.Run("Ungrouped", func(t *testing.T) {
		decl := &xast.GenDecl{Doc: declDoc}
		if got := xgoDeclDoc(decl, spec); got != declDoc {
			t.Fatal("expected declaration doc to take precedence")
		}
	})

	t.Run("Grouped", func(t *testing.T) {
		decl := &xast.GenDecl{Doc: declDoc, Lparen: token.Pos(3)}
		if got := xgoDeclDoc(decl, spec); got != specDoc {
			t.Fatal("expected spec doc for grouped declaration")
		}
	})
}

func TestGoDeclDoc(t *testing.T) {
	declDoc := &goast.CommentGroup{List: []*goast.Comment{{Text: "// decl", Slash: 1}}}
	specDoc := &goast.CommentGroup{List: []*goast.Comment{{Text: "// spec", Slash: 2}}}
	spec := &goast.TypeSpec{Doc: specDoc}

	t.Run("Ungrouped", func(t *testing.T) {
		decl := &goast.GenDecl{Doc: declDoc}
		if got := goDeclDoc(decl, spec); got != (*xast.CommentGroup)(declDoc) {
			t.Fatal("expected declaration doc to take precedence")
		}
	})

	t.Run("Grouped", func(t *testing.T) {
		decl := &goast.GenDecl{Doc: declDoc, Lparen: 3}
		if got := goDeclDoc(decl, spec); got != (*xast.CommentGroup)(specDoc) {
			t.Fatal("expected spec doc for grouped declaration")
		}
	})
}

func TestXGoRecvBaseName(t *testing.T) {
	t.Run("Named", func(t *testing.T) {
		recv := &xast.FieldList{List: []*xast.Field{{Type: &xast.Ident{Name: "SpriteImpl"}}}}
		if got := xgoRecvBaseName(recv); got != "SpriteImpl" {
			t.Fatalf("unexpected receiver base name: %q", got)
		}
	})

	t.Run("Nil", func(t *testing.T) {
		if got := xgoRecvBaseName(nil); got != "" {
			t.Fatalf("unexpected receiver base name: %q", got)
		}
	})
}

func TestXGoExprBaseName(t *testing.T) {
	for _, tc := range []struct {
		name string
		expr xast.Expr
		want string
	}{
		{name: "Named", expr: &xast.Ident{Name: "SpriteImpl"}, want: "SpriteImpl"},
		{name: "Pointer", expr: &xast.StarExpr{X: &xast.Ident{Name: "SpriteImpl"}}, want: "SpriteImpl"},
		{name: "Parenthesized", expr: &xast.ParenExpr{X: &xast.Ident{Name: "SpriteImpl"}}, want: "SpriteImpl"},
		{name: "Unsupported", expr: &xast.ArrayType{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := xgoExprBaseName(tc.expr); got != tc.want {
				t.Fatalf("xgoExprBaseName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGoRecvBaseName(t *testing.T) {
	t.Run("Named", func(t *testing.T) {
		recv := &goast.FieldList{List: []*goast.Field{{Type: &goast.Ident{Name: "SpriteImpl"}}}}
		if got := goRecvBaseName(recv); got != "SpriteImpl" {
			t.Fatalf("unexpected receiver base name: %q", got)
		}
	})

	t.Run("Nil", func(t *testing.T) {
		if got := goRecvBaseName(nil); got != "" {
			t.Fatalf("unexpected receiver base name: %q", got)
		}
	})
}

func TestGoExprBaseName(t *testing.T) {
	for _, tc := range []struct {
		name string
		expr goast.Expr
		want string
	}{
		{name: "Named", expr: &goast.Ident{Name: "SpriteImpl"}, want: "SpriteImpl"},
		{name: "Pointer", expr: &goast.StarExpr{X: &goast.Ident{Name: "SpriteImpl"}}, want: "SpriteImpl"},
		{name: "Parenthesized", expr: &goast.ParenExpr{X: &goast.Ident{Name: "SpriteImpl"}}, want: "SpriteImpl"},
		{name: "Generic", expr: &goast.IndexExpr{X: &goast.Ident{Name: "SpriteImpl"}}, want: "SpriteImpl"},
		{name: "GenericMultiple", expr: &goast.IndexListExpr{X: &goast.Ident{Name: "SpriteImpl"}}, want: "SpriteImpl"},
		{name: "Unsupported", expr: &goast.ArrayType{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := goExprBaseName(tc.expr); got != tc.want {
				t.Fatalf("goExprBaseName() = %q, want %q", got, tc.want)
			}
		})
	}
}
