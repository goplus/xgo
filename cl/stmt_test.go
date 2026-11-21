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
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

func TestCompileReturnStmt(t *testing.T) {
	const filename = "return.xgo"
	const src = `func f() int {
	return "prefix" + missing
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	decl, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("got declaration %T, want *ast.FuncDecl", file.Decls[0])
	}
	interp := &nodeInterp{fset: fset, files: map[string]*ast.File{filename: file}}
	state := &pkgCtx{nodeInterp: interp, fset: fset}
	pkg := gogen.NewPackage("main", "main", &gogen.Config{
		Fset:            fset,
		HandleErr:       state.handleErr,
		NodeInterpreter: interp,
	})
	ctx := &blockCtx{pkgCtx: state, pkg: pkg, cb: pkg.CB()}
	result := types.NewParam(token.NoPos, pkg.Types, "", types.Typ[types.Int])
	fn := pkg.NewFunc(nil, decl.Name.Name, nil, types.NewTuple(result), false)
	loadFuncBody(ctx, fn, decl.Body, nil, decl, false)

	const want = "return.xgo:2:20: undefined: missing"
	if err := state.complete(); err == nil || err.Error() != want {
		t.Fatalf("compiler error = %v, want %q", err, want)
	}
}
