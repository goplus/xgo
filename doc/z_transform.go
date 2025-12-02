/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package doc

import (
	"go/ast"
	"go/doc"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

type mthd struct {
	typ  string
	name string
}

type omthd struct {
	mthd
	idx int
}

type typExtra struct {
	t       *doc.Type
	funcs   []*doc.Func
	methods []*doc.Func
}

type transformCtx struct {
	overloadFuncs map[mthd][]omthd // realName => []overloadName
	typs          map[string]*typExtra
	orders        map[*doc.Func]int
}

func (p *transformCtx) finish(in *doc.Package) {
	for _, ex := range p.typs {
		if t := ex.t; t != nil {
			t.Funcs = p.mergeFuncs(t.Funcs, ex.funcs)
			t.Methods = p.mergeFuncs(t.Methods, ex.methods)
		} else {
			in.Funcs = p.mergeFuncs(in.Funcs, ex.funcs)
		}
	}
}

func (p *transformCtx) mergeFuncs(a, b []*doc.Func) []*doc.Func {
	if len(b) == 0 {
		return a
	}
	a = append(a, b...)
	sort.Slice(a, func(i, j int) bool {
		fa, fb := a[i], a[j]
		aName, bName := fa.Name, fb.Name
		if aName == bName {
			return p.orders[fa] < p.orders[fb]
		}
		return aName < bName
	})
	return a
}

func newCtx(in *doc.Package) *transformCtx {
	typs := make(map[string]*typExtra, len(in.Types)+1)
	typs[""] = &typExtra{} // global functions
	for _, t := range in.Types {
		typs[t.Name] = &typExtra{t: t}
	}
	return &transformCtx{
		overloadFuncs: make(map[mthd][]omthd),
		typs:          typs,
		orders:        make(map[*doc.Func]int),
	}
}

func newIdent(name string, in *ast.Ident) *ast.Ident {
	ret := *in
	ret.Name = name
	return &ret
}

func newFuncDecl(name string, in *ast.FuncDecl) *ast.FuncDecl {
	ret := *in
	ret.Name = newIdent(name, ret.Name)
	return &ret
}

func newMethodDecl(name string, in *ast.FuncDecl) *ast.FuncDecl {
	ret := *in
	if ret.Recv == nil {
		ft := *ret.Type
		params := *ft.Params
		ret.Recv = &ast.FieldList{List: params.List[:1]}
		params.List = params.List[1:]
		ft.Params = &params
		ret.Type = &ft
	}
	ret.Name = newIdent(name, ret.Name)
	return &ret
}

func docRecv(recv *ast.Field) (_ string, ok bool) {
	switch v := recv.Type.(type) {
	case *ast.Ident:
		return v.Name, true
	case *ast.StarExpr:
		if t, ok := v.X.(*ast.Ident); ok {
			return "*" + t.Name, true
		}
	}
	return
}

func newMethod(name string, in *doc.Func) *doc.Func {
	ret := *in
	ret.Name = name
	ret.Decl = newMethodDecl(name, in.Decl)
	if recv, ok := docRecv(ret.Decl.Recv.List[0]); ok {
		ret.Recv = recv
	}
	// TODO(xsw): alias doc - ret.Doc
	return &ret
}

func newFunc(name string, in *doc.Func) *doc.Func {
	ret := *in
	ret.Name = name
	ret.Decl = newFuncDecl(name, in.Decl)
	// TODO(xsw): alias doc - ret.Doc
	return &ret
}

func setOrder(ctx *transformCtx, in *doc.Func, order int) *doc.Func {
	ctx.orders[in] = order
	return in
}

func buildFunc(ctx *transformCtx, overload omthd, in *doc.Func) {
	if ex, ok := ctx.typs[overload.typ]; ok {
		if ex.t != nil { // method
			ex.methods = append(ex.methods, setOrder(ctx, newMethod(overload.name, in), overload.idx))
		} else {
			ex.funcs = append(ex.funcs, setOrder(ctx, newFunc(overload.name, in), overload.idx))
		}
	}
}

func toIndex(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return int(c - ('a' - 10))
	}
	panic("invalid character out of [0-9,a-z]")
}

func transformFunc(ctx *transformCtx, t *doc.Type, in *doc.Func, method bool) {
	var m mthd
	if method {
		m.typ = t.Name
	}
	m.name = in.Name
	if overloads, ok := ctx.overloadFuncs[m]; ok {
		for _, overload := range overloads {
			buildFunc(ctx, overload, in)
		}
	}
	if isOverload(in.Name) {
		order := toIndex(in.Name[len(in.Name)-1])
		in.Name = in.Name[:len(in.Name)-3]
		in.Decl.Name.Name = in.Name
		ctx.orders[in] = order
	}
}

func transformFuncs(ctx *transformCtx, t *doc.Type, in []*doc.Func, method bool) {
	for _, f := range in {
		transformFunc(ctx, t, f, method)
	}
}

func transformTypes(ctx *transformCtx, in []*doc.Type) {
	for _, t := range in {
		transformFuncs(ctx, t, t.Funcs, false)
		transformFuncs(ctx, t, t.Methods, true)
	}
}

func transformGopo(ctx *transformCtx, name, val string) {
	overload := checkTypeMethod(name[len(gopoPrefix):])
	parts := strings.Split(val, ",")
	for idx, part := range parts {
		if part == "" {
			continue
		}
		var real mthd
		if part[0] == '.' {
			real = mthd{overload.typ, part[1:]}
		} else {
			real = mthd{"", part}
		}
		ctx.overloadFuncs[real] = append(ctx.overloadFuncs[real], omthd{overload, idx})
	}
}

func transformConstSpec(ctx *transformCtx, vspec *ast.ValueSpec) {
	name := vspec.Names[0].Name
	if isGopoConst(name) {
		if lit, ok := vspec.Values[0].(*ast.BasicLit); ok {
			if lit.Kind == token.STRING {
				if val, e := strconv.Unquote(lit.Value); e == nil {
					transformGopo(ctx, name, val)
				}
			}
		}
	}
}

func transformConst(ctx *transformCtx, in *doc.Value) {
	if hasGopoConst(in) {
		for _, spec := range in.Decl.Specs {
			vspec := spec.(*ast.ValueSpec)
			transformConstSpec(ctx, vspec)
		}
	}
}

func transformConsts(ctx *transformCtx, in []*doc.Value) {
	for _, v := range in {
		transformConst(ctx, v)
	}
}

// Transform converts a Go doc package to an XGo doc package.
func Transform(in *doc.Package) *doc.Package {
	if isXGoPackage(in) {
		ctx := newCtx(in)
		transformConsts(ctx, in.Consts)
		transformFuncs(ctx, nil, in.Funcs, false)
		transformTypes(ctx, in.Types)
		ctx.finish(in)
	}
	return in
}
