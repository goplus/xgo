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
	"fmt"
	"strings"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

const wrapFuncResultPrefix = "__gop_wrap_ret"

func wrapFuncBody(ctx *blockCtx, decl *ast.FuncDecl) (*ast.BlockStmt, error) {
	body := decl.Body
	modifier := decl.Wrap
	if modifier == nil || body == nil {
		return body, nil
	}

	wrapCall, err := wrapFuncCallName(ctx, modifier)
	if err != nil {
		return nil, err
	}
	if resultCount(decl.Type.Results) == 0 {
		return wrapFuncBodyNoResults(modifier, wrapCall, body), nil
	}
	return wrapFuncBodyWithResults(decl, modifier, wrapCall, body), nil
}

func wrapFuncCallName(ctx *blockCtx, modifier *ast.Ident) (string, error) {
	if ctx == nil || ctx.proj == nil {
		return "", fmt.Errorf("%s func is only supported in classfiles with %s", modifier.Name, classConstWrapCall)
	}

	wrapCall := ctx.proj.wrapFuncCall
	if wrapCall == "" {
		return "", ctx.newCodeErrorf(
			modifier.Pos(), modifier.End(), "%s func requires class package const %s", modifier.Name, classConstWrapCall,
		)
	}
	if !strings.EqualFold(modifier.Name, wrapCall) {
		return "", ctx.newCodeErrorf(
			modifier.Pos(), modifier.End(), "%s func doesn't match class package %s = %q", modifier.Name, classConstWrapCall, wrapCall,
		)
	}
	return wrapCall, nil
}

func wrapFuncBodyNoResults(modifier *ast.Ident, wrapCall string, body *ast.BlockStmt) *ast.BlockStmt {
	call := wrapFuncCallStmt(modifier, wrapCall, wrapFuncClosure(modifier.Pos(), body))
	return wrapFuncBlock(body, []ast.Stmt{call})
}

func wrapFuncBodyWithResults(decl *ast.FuncDecl, modifier *ast.Ident, wrapCall string, body *ast.BlockStmt) *ast.BlockStmt {
	decls, lhs, ret := wrapFuncResultVars(decl.Type.Results, wrapFuncSignatureNames(decl))
	assign := &ast.AssignStmt{
		Lhs: lhs,
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{
					Func:    decl.Type.Func,
					Params:  &ast.FieldList{},
					Results: decl.Type.Results,
				},
				Body: body,
			},
		}},
	}
	call := wrapFuncCallStmt(modifier, wrapCall, &ast.FuncLit{
		Type: wrapFuncClosureType(modifier.Pos()),
		Body: &ast.BlockStmt{
			List: []ast.Stmt{assign},
		},
	})

	stmts := make([]ast.Stmt, 0, len(decls)+2)
	for _, spec := range decls {
		stmts = append(stmts, &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok:   token.VAR,
			Specs: []ast.Spec{spec},
		}})
	}
	stmts = append(stmts, call, &ast.ReturnStmt{Results: ret})
	return wrapFuncBlock(body, stmts)
}

func wrapFuncCallStmt(modifier *ast.Ident, wrapCall string, closure *ast.FuncLit) *ast.ExprStmt {
	return &ast.ExprStmt{X: &ast.CallExpr{
		Fun:  cloneIdentWithName(modifier, wrapCall),
		Args: []ast.Expr{closure},
	}}
}

func wrapFuncBlock(src *ast.BlockStmt, list []ast.Stmt) *ast.BlockStmt {
	return &ast.BlockStmt{
		Lbrace: src.Lbrace,
		List:   list,
		Rbrace: src.Rbrace,
	}
}

func wrapFuncClosure(pos token.Pos, body *ast.BlockStmt) *ast.FuncLit {
	return &ast.FuncLit{
		Type: wrapFuncClosureType(pos),
		Body: body,
	}
}

func wrapFuncClosureType(pos token.Pos) *ast.FuncType {
	return &ast.FuncType{
		Func:   pos,
		Params: &ast.FieldList{},
	}
}

func wrapFuncResultVars(results *ast.FieldList, used map[string]bool) (decls []*ast.ValueSpec, lhs []ast.Expr, ret []ast.Expr) {
	for _, field := range results.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		spec := &ast.ValueSpec{Type: field.Type}
		for i := 0; i < count; i++ {
			name := wrapFuncResultName(used)
			spec.Names = append(spec.Names, ast.NewIdent(name))
			lhs = append(lhs, ast.NewIdent(name))
			ret = append(ret, ast.NewIdent(name))
		}
		decls = append(decls, spec)
	}
	return
}

func wrapFuncResultName(used map[string]bool) string {
	for i := 0; ; i++ {
		name := fmt.Sprintf("%s%d", wrapFuncResultPrefix, i)
		if !used[name] {
			used[name] = true
			return name
		}
	}
}

func wrapFuncSignatureNames(decl *ast.FuncDecl) map[string]bool {
	used := make(map[string]bool)
	collectWrapFuncNames(used, decl.Recv)
	collectWrapFuncNames(used, decl.Type.Params)
	collectWrapFuncNames(used, decl.Type.Results)
	return used
}

func collectWrapFuncNames(used map[string]bool, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			if name.Name != "_" {
				used[name.Name] = true
			}
		}
	}
}

func resultCount(results *ast.FieldList) int {
	if results == nil {
		return 0
	}
	count := 0
	for _, field := range results.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func cloneIdentWithName(v *ast.Ident, name string) *ast.Ident {
	if v == nil {
		return nil
	}
	if name == "" {
		name = v.Name
	}
	return &ast.Ident{NamePos: v.NamePos, Name: name}
}
