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

package format

import (
	"go/types"
	"slices"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

func newFormatTestCtx() *formatCtx {
	return &formatCtx{
		imports: map[string]*importCtx{
			"fmt": {pkgPath: "fmt"},
		},
		scope: types.NewScope(nil, token.NoPos, token.NoPos, ""),
	}
}

func fmtCallExpr(name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("fmt"),
			Sel: ast.NewIdent(name),
		},
		Args: args,
	}
}

func formattedCallName(t *testing.T, expr ast.Expr) string {
	t.Helper()
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expression = %T, want *ast.CallExpr", expr)
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		t.Fatalf("call function = %T, want *ast.Ident", call.Fun)
	}
	return name.Name
}

func formattedLambdaCallName(t *testing.T, expr ast.Expr) string {
	t.Helper()
	lambda, ok := expr.(*ast.LambdaExpr)
	if !ok {
		t.Fatalf("expression = %T, want *ast.LambdaExpr", expr)
	}
	if len(lambda.Rhs) != 1 {
		t.Fatalf("lambda rhs length = %d, want 1", len(lambda.Rhs))
	}
	return formattedCallName(t, lambda.Rhs[0])
}

func TestFormatExtensionExprBranches(t *testing.T) {
	for _, tt := range []struct {
		name string
		expr ast.Expr
		got  func(*testing.T, ast.Expr) []string
		want []string
	}{
		{
			name: "ForPhrase",
			expr: &ast.ForPhrase{
				X: fmtCallExpr("Sprint", &ast.BasicLit{Kind: token.INT, Value: "1"}),
				Cond: fmtCallExpr(
					"Sprintf",
					&ast.BasicLit{Kind: token.STRING, Value: `"%d"`},
					&ast.BasicLit{Kind: token.INT, Value: "2"},
				),
			},
			got: func(t *testing.T, expr ast.Expr) []string {
				forPhrase := expr.(*ast.ForPhrase)
				return []string{
					formattedCallName(t, forPhrase.X),
					formattedCallName(t, forPhrase.Cond),
				}
			},
			want: []string{"sprint", "sprintf"},
		},
		{
			name: "KwargExpr",
			expr: &ast.KwargExpr{
				Name:  ast.NewIdent("msg"),
				Value: fmtCallExpr("Sprint", &ast.BasicLit{Kind: token.INT, Value: "1"}),
			},
			got: func(t *testing.T, expr ast.Expr) []string {
				return []string{formattedCallName(t, expr.(*ast.KwargExpr).Value)}
			},
			want: []string{"sprint"},
		},
		{
			name: "KwargExprFuncLit",
			expr: &ast.KwargExpr{
				Name: ast.NewIdent("cb"),
				Value: &ast.FuncLit{
					Type: &ast.FuncType{
						Params: &ast.FieldList{},
						Results: &ast.FieldList{List: []*ast.Field{
							{Type: ast.NewIdent("string")},
						}},
					},
					Body: &ast.BlockStmt{List: []ast.Stmt{
						&ast.ReturnStmt{Results: []ast.Expr{
							fmtCallExpr("Sprint", &ast.BasicLit{Kind: token.INT, Value: "1"}),
						}},
					}},
				},
			},
			got: func(t *testing.T, expr ast.Expr) []string {
				return []string{formattedLambdaCallName(t, expr.(*ast.KwargExpr).Value)}
			},
			want: []string{"sprint"},
		},
		{
			name: "ElemEllipsis",
			expr: &ast.ElemEllipsis{
				Elt: fmtCallExpr("Sprint", &ast.BasicLit{Kind: token.INT, Value: "1"}),
			},
			got: func(t *testing.T, expr ast.Expr) []string {
				return []string{formattedCallName(t, expr.(*ast.ElemEllipsis).Elt)}
			},
			want: []string{"sprint"},
		},
		{
			name: "AnySelectorExpr",
			expr: &ast.AnySelectorExpr{
				X:   fmtCallExpr("Sprint", &ast.BasicLit{Kind: token.INT, Value: "1"}),
				Sel: ast.NewIdent("name"),
			},
			got: func(t *testing.T, expr ast.Expr) []string {
				return []string{formattedCallName(t, expr.(*ast.AnySelectorExpr).X)}
			},
			want: []string{"sprint"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newFormatTestCtx()
			expr := tt.expr
			formatExpr(ctx, expr, &expr)
			if got := tt.got(t, expr); !slices.Equal(got, tt.want) {
				t.Fatalf("formatted call names = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFuncTypeTypeParams(t *testing.T) {
	ctx := newFormatTestCtx()
	formatFuncType(ctx, &ast.FuncType{
		TypeParams: &ast.FieldList{List: []*ast.Field{
			{
				Names: []*ast.Ident{ast.NewIdent("T")},
				Type: &ast.SelectorExpr{
					X:   ast.NewIdent("fmt"),
					Sel: ast.NewIdent("Stringer"),
				},
			},
		}},
		Params: &ast.FieldList{},
	})
	if !ctx.imports["fmt"].isUsed {
		t.Fatal("type parameter constraint didn't mark fmt import as used")
	}
}

func TestMarkExprImports(t *testing.T) {
	t.Run("NilExpr", func(t *testing.T) {
		ctx := newFormatTestCtx()
		markExprImports(ctx, nil)
		if ctx.imports["fmt"].isUsed {
			t.Fatal("nil expression marked fmt import as used")
		}
	})

	t.Run("NonIdentifierSelector", func(t *testing.T) {
		ctx := newFormatTestCtx()
		markExprImports(ctx, &ast.SelectorExpr{
			X:   &ast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
			Sel: ast.NewIdent("Stringer"),
		})
		if ctx.imports["fmt"].isUsed {
			t.Fatal("non-identifier selector marked fmt import as used")
		}
	})

	t.Run("ShadowedImportName", func(t *testing.T) {
		ctx := newFormatTestCtx()
		ctx.insert("fmt")
		markExprImports(ctx, &ast.SelectorExpr{
			X:   ast.NewIdent("fmt"),
			Sel: ast.NewIdent("Stringer"),
		})
		if ctx.imports["fmt"].isUsed {
			t.Fatal("scoped fmt identifier marked fmt import as used")
		}
	})

	t.Run("ImportedSelector", func(t *testing.T) {
		ctx := newFormatTestCtx()
		markExprImports(ctx, &ast.SelectorExpr{
			X:   ast.NewIdent("fmt"),
			Sel: ast.NewIdent("Stringer"),
		})
		if !ctx.imports["fmt"].isUsed {
			t.Fatal("fmt selector didn't mark fmt import as used")
		}
	})

	t.Run("ParenthesizedImportedSelector", func(t *testing.T) {
		ctx := newFormatTestCtx()
		markExprImports(ctx, &ast.SelectorExpr{
			X:   &ast.ParenExpr{X: ast.NewIdent("fmt")},
			Sel: ast.NewIdent("Stringer"),
		})
		if !ctx.imports["fmt"].isUsed {
			t.Fatal("parenthesized fmt selector didn't mark fmt import as used")
		}
	})
}
