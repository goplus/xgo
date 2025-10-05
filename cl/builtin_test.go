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
	"log"
	"testing"

	"github.com/goplus/gogen"
	"github.com/goplus/gogen/packages"
	"github.com/goplus/mod/modfile"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

var (
	goxConf = getGoxConf()
)

func getGoxConf() *gogen.Config {
	fset := token.NewFileSet()
	imp := packages.NewImporter(fset)
	return &gogen.Config{Fset: fset, Importer: imp}
}

func TestEmbeddedFieldCast(t *testing.T) {
	var o = new(types.Struct)
	embeddedFieldCast(o, nil, nil, visitedT{o: none{}})
	embeddedFieldCast(o, nil, nil, visitedT{})
}

func TestNonClosure(t *testing.T) {
	tn := types.NewTypeName(0, nil, "a", nil)
	if !nonClosure(types.NewNamed(tn, types.Typ[types.Int], nil)) {
		t.Fatal("nonClosure")
	}
}

func TestLoadExpr(t *testing.T) {
	var ni nodeInterp
	if v := ni.LoadExpr(&ast.Ident{Name: "x"}); v != "" {
		t.Fatal("LoadExpr:", v)
	}
}

func TestSpriteOf(t *testing.T) {
	proj := &gmxProject{}
	if getSpxObj(proj, "a") != nil {
		t.Fatal("spriteOf: not nil?")
	}
}

func TestGetGameClass(t *testing.T) {
	proj := &gmxProject{
		gameIsPtr:  true,
		hasMain_:   false,
		gameClass_: "",
		gt: &modfile.Project{
			Class:    "*App",
			PkgPaths: []string{"foo/bar"},
		},
	}
	ctx := &pkgCtx{
		projs: map[string]*gmxProject{".gmx": proj, ".spx": proj},
		nproj: 2,
	}
	if v := proj.getGameClass(ctx); v != "BarApp" {
		t.Fatal("getGameClass:", v)
	}
}

func TestSimplifyPkgPath(t *testing.T) {
	if simplifyPkgPath("c/lua") != "github.com/goplus/lib/c/lua" {
		t.Fatal("simplifyPkgPath: c/lua")
	}
	if simplifyPkgPath("cpp/std") != "github.com/goplus/lib/cpp/std" {
		t.Fatal("simplifyPkgPath: cpp/std")
	}
}

func TestCompileLambdaExpr(t *testing.T) {
	ctx := &blockCtx{
		pkgCtx: &pkgCtx{},
	}
	lhs := []*ast.Ident{ast.NewIdent("x")}
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	e := compileLambdaExpr(ctx, &ast.LambdaExpr{Lhs: lhs}, sig)
	if ce := e.(*gogen.CodeError); ce.Msg != `too many arguments in lambda expression
	have (x)
	want ()` {
		t.Fatal("compileLambdaExpr:", ce.Msg)
	}
}

func TestCompileLambda1(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			if ce := e.(*gogen.CodeError); ce.Msg != `too many arguments in lambda expression
	have (x)
	want ()` {
				t.Fatal("compileLambda:", ce.Msg)
			}
		}
	}()
	ctx := &blockCtx{
		pkgCtx: &pkgCtx{},
	}
	lhs := []*ast.Ident{ast.NewIdent("x")}
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	compileLambda(ctx, &ast.LambdaExpr{Lhs: lhs}, sig)
}

func TestCompileLambda2(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			if ce := e.(*gogen.CodeError); ce.Msg != `too many arguments in lambda expression
	have (x)
	want ()` {
				t.Fatal("compileLambda:", ce.Msg)
			}
		}
	}()
	ctx := &blockCtx{
		pkgCtx: &pkgCtx{},
	}
	lhs := []*ast.Ident{ast.NewIdent("x")}
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	compileLambda(ctx, &ast.LambdaExpr2{Lhs: lhs}, sig)
}

func TestCompileExpr(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			if ce := e.(*gogen.CodeError); ce.Msg != "compileExpr failed: unknown - *ast.Ellipsis" {
				t.Fatal("compileExpr:", ce.Msg)
			}
		}
	}()
	ctx := &blockCtx{pkgCtx: &pkgCtx{}}
	compileExpr(ctx, &ast.Ellipsis{})
}

func TestCompileStmt(t *testing.T) {
	old := enableRecover
	defer func() {
		enableRecover = old
		if e := recover(); e != "compileStmt failed: unknown - *ast.BadStmt\n" {
			t.Fatal("compileStmt:", e)
		}
	}()
	enableRecover = false
	ctx := &blockCtx{}
	compileStmt(ctx, &ast.BadStmt{})
}

func TestTryGopExec(t *testing.T) {
	pkg := gogen.NewPackage("", "foo", goxConf)
	if tryGopExec(pkg.CB(), nil) {
		t.Fatal("tryGopExec")
	}
}

func TestCompileFuncAlias(t *testing.T) {
	ctx := &blockCtx{
		pkgCtx: &pkgCtx{
			syms: map[string]loader{"Foo": &baseLoader{
				fn: func() {},
			}},
		},
	}
	scope := types.NewScope(nil, 0, 0, "")
	x := ast.NewIdent("foo")
	if compileFuncAlias(ctx, scope, x, 0) {
		t.Fatal("compileFuncAlias: ok?")
	}
}

func TestErrStringLit(t *testing.T) {
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("TestErrStringLit: no panic?")
		}
	}()
	compileStringLitEx(nil, nil, &ast.BasicLit{
		Value: "Hello",
		Extra: &ast.StringLitEx{
			Parts: []any{1},
		},
	})
}

func TestErrPreloadFile(t *testing.T) {
	pkg := gogen.NewPackage("", "foo", goxConf)
	ctx := &blockCtx{pkgCtx: &pkgCtx{}}
	t.Run("unknown decl", func(t *testing.T) {
		defer func() {
			if e := recover(); e == nil || e != "TODO - cl.preloadFile: unknown decl - *ast.BadDecl\n" {
				t.Fatal("TestErrPreloadFile:", e)
			}
		}()
		decls := []ast.Decl{
			&ast.BadDecl{},
		}
		preloadFile(pkg, ctx, &ast.File{Decls: decls}, "", true)
	})
}

func TestErrParseTypeEmbedName(t *testing.T) {
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("TestErrParseTypeEmbedName: no panic?")
		}
	}()
	parseTypeEmbedName(&ast.StructType{})
}

func TestGmxCheckProjs(t *testing.T) {
	_, multi := gmxCheckProjs(nil, &pkgCtx{
		projs: map[string]*gmxProject{
			".a": {hasMain_: true}, ".b": {hasMain_: true},
		},
	})
	if !multi {
		t.Fatal("gmxCheckProjs: not multi?")
	}
}

func TestGmxCheckProjs2(t *testing.T) {
	_, multi := gmxCheckProjs(nil, &pkgCtx{
		projs: map[string]*gmxProject{
			".a": {}, ".b": {},
		},
	})
	if !multi {
		t.Fatal("gmxCheckProjs: not multi?")
	}
}

func TestNodeInterp(t *testing.T) {
	ni := &nodeInterp{}
	if v := ni.Caller(&ast.Ident{}); v != "the function call" {
		t.Fatal("TestNodeInterp:", v)
	}
	defer func() {
		if e := recover(); e == nil {
			log.Fatal("TestNodeInterp: no error")
		}
	}()
	ni.Caller(&ast.CallExpr{})
}

func TestMarkAutogen(t *testing.T) {
	old := noMarkAutogen
	noMarkAutogen = false

	NewPackage("", &ast.Package{Files: map[string]*ast.File{
		"main.t2gmx": {IsProj: true},
	}}, &Config{
		LookupClass: lookupClassErr,
	})

	noMarkAutogen = old
}

func TestClassNameAndExt(t *testing.T) {
	name, clsfile, ext := ClassNameAndExt("/foo/bar.abc_yap.gox")
	if name != "bar_abc" || clsfile != "bar.abc" || ext != "_yap.gox" {
		t.Fatal("classNameAndExt:", name, ext)
	}
	name, clsfile, ext = ClassNameAndExt("/foo/get-bar_:id.yap")
	if name != "get_bar_id" || clsfile != "get-bar_:id" || ext != ".yap" {
		t.Fatal("classNameAndExt:", name, ext)
	}
}

func TestFileClassType(t *testing.T) {
	type testData struct {
		isClass     bool
		isNormalGox bool
		isProj      bool
		fileName    string
		classType   string
		isTest      bool
		found       bool
	}
	tests := []*testData{
		{false, false, false, "abc.gop", "", false, false},
		{false, false, false, "abc.xgo", "", false, false},
		{false, false, false, "abc_test.gop", "", true, false},
		{false, false, false, "abc_test.xgo", "", true, false},

		{true, true, false, "abc.gox", "abc", false, true},
		{true, true, false, "Abc.gox", "Abc", false, true},
		{true, true, false, "abc_demo.gox", "abc", false, true},
		{true, true, false, "Abc_demo.gox", "Abc", false, true},

		{true, true, false, "main.gox", "_main", false, true},
		{true, true, false, "main_demo.gox", "_main", false, true},
		{true, true, false, "abc_xtest.gox", "abc", false, true},
		{true, true, false, "main_xtest.gox", "_main", false, true},

		{true, true, false, "abc_test.gox", "case_abc", true, true},
		{true, true, false, "Abc_test.gox", "caseAbc", true, true},
		{true, true, false, "main_test.gox", "case_main", true, true},

		{true, false, false, "get.yap", "get", false, true},
		{true, false, false, "get_p_#id.yap", "get_p_id", false, true},
		{true, false, true, "main.yap", "AppV2", false, true},

		{true, false, false, "abc_yap.gox", "abc", false, true},
		{true, false, false, "Abc_yap.gox", "Abc", false, true},
		{true, false, true, "main_yap.gox", "App", false, true},

		{true, false, true, "abc_yap.gox", "abc", false, true},
		{true, false, true, "Abc_yap.gox", "Abc", false, true},
		{true, false, true, "main_yap.gox", "App", false, true},

		{true, false, false, "abc_ytest.gox", "case_abc", true, true},
		{true, false, false, "Abc_ytest.gox", "caseAbc", true, true},
		{true, false, true, "main_ytest.gox", "App", true, true},
	}
	lookupClass := func(ext string) (c *Project, ok bool) {
		switch ext {
		case ".yap":
			return &modfile.Project{
				Ext: ".yap", Class: "AppV2",
				Works:    []*modfile.Class{{Ext: ".yap", Class: "Handler"}},
				PkgPaths: []string{"github.com/goplus/yap"}}, true
		case "_yap.gox":
			return &modfile.Project{
				Ext: "_yap.gox", Class: "App",
				PkgPaths: []string{"github.com/goplus/yap"}}, true
		case "_ytest.gox":
			return &modfile.Project{
				Ext: "_ytest.gox", Class: "App",
				Works:    []*modfile.Class{{Ext: "_ytest.gox", Class: "Case"}},
				PkgPaths: []string{"github.com/goplus/yap/ytest", "testing"}}, true
		}
		return
	}
	for _, test := range tests {
		_ = test.found
		f := &ast.File{IsClass: test.isClass, IsNormalGox: test.isNormalGox, IsProj: test.isProj}
		classType, isTest := GetFileClassType(f, test.fileName, lookupClass)
		if isTest != test.isTest {
			t.Fatalf("%v check classType isTest want %v, got %v.", test.fileName, test.isTest, isTest)
		}
		if classType != test.classType {
			t.Fatalf("%v getClassType want %v, got %v.", test.fileName, test.classType, classType)
		}
	}
}

func TestErrMultiStarRecv(t *testing.T) {
	_, _, ok := getRecvType(&ast.StarExpr{
		X: &ast.StarExpr{},
	})
	if ok {
		t.Fatal("TestErrMultiStarRecv: no error?")
	}
}

func TestErrAssign(t *testing.T) {
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("TestErrAssign: no panic?")
		}
	}()
	ctx := &blockCtx{}
	compileAssignStmt(ctx, &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{
			&ast.SelectorExpr{
				X:   ast.NewIdent("foo"),
				Sel: ast.NewIdent("bar"),
			},
		},
	})
}

func TestErrPanicToRecv(t *testing.T) {
	ctx := &blockCtx{
		tlookup: &typeParamLookup{
			[]*types.TypeParam{
				types.NewTypeParam(types.NewTypeName(0, nil, "t", nil), nil),
			},
		},
	}
	recv := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.SelectorExpr{}},
		},
	}
	func() {
		defer func() {
			if e := recover(); e == nil {
				t.Fatal("TestErrPanicToRecv: no panic?")
			}
		}()
		toRecv(ctx, recv)
	}()
}

func TestCompileErrWrapExpr(t *testing.T) {
	defer func() {
		if e := recover(); e != "TODO: can't use expr? in global" {
			t.Fatal("TestCompileErrWrapExpr failed")
		}
	}()
	pkg := gogen.NewPackage("", "foo", goxConf)
	ctx := &blockCtx{pkg: pkg, cb: pkg.CB()}
	compileErrWrapExpr(ctx, &ast.ErrWrapExpr{Tok: token.QUESTION}, 0)
}

func TestToString(t *testing.T) {
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("toString: no error?")
		}
	}()
	toString(&ast.BasicLit{Kind: token.INT, Value: "1"})
}

func TestGetTypeName(t *testing.T) {
	if getTypeName(types.Typ[types.Int]) != "int" {
		t.Fatal("getTypeName int failed")
	}
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("getTypeName: no error?")
		}
	}()
	getTypeName(types.NewSlice(types.Typ[types.Int]))
}

func TestHandleRecover(t *testing.T) {
	var ctx pkgCtx
	ctx.handleRecover("hello", nil)
	if !(len(ctx.errs) == 1 && ctx.errs[0].Error() == "hello") {
		t.Fatal("TestHandleRecover failed:", ctx.errs)
	}
}

func TestCheckCommandWithoutArgs(t *testing.T) {
	if checkCommandWithoutArgs(
		&ast.SelectorExpr{
			X:   &ast.SelectorExpr{X: ast.NewIdent("foo"), Sel: ast.NewIdent("bar")},
			Sel: ast.NewIdent("val"),
		}) != clCommandWithoutArgs {
		t.Fatal("TestCanAutoCall failed")
	}
}

func TestClRangeStmt(t *testing.T) {
	ctx := &blockCtx{
		cb: &gogen.CodeBuilder{},
	}
	stmt := &ast.RangeStmt{
		Tok:  token.DEFINE,
		X:    &ast.SliceLit{},
		Body: &ast.BlockStmt{},
	}
	compileRangeStmt(ctx, stmt)
	stmt.Tok = token.ASSIGN
	stmt.Value = &ast.Ident{Name: "_"}
	compileRangeStmt(ctx, stmt)
}

// -----------------------------------------------------------------------------

func TestGetStringConst(t *testing.T) {
	spx := gogen.PkgRef{Types: types.NewPackage("", "foo")}
	if v := getStringConst(spx, "unknown"); v != "" {
		t.Fatal("getStringConst:", v)
	}
}

func TestSpxRef(t *testing.T) {
	defer func() {
		if e := recover(); !isError(e, "foo.bar not found") {
			t.Fatal("TestSpxRef:", e)
		}
	}()
	pkg := gogen.PkgRef{
		Types: types.NewPackage("foo", "foo"),
	}
	spxRef(pkg, "bar")
}

func isError(e any, msg string) bool {
	if e != nil {
		if err, ok := e.(error); ok {
			return err.Error() == msg
		}
		if err, ok := e.(string); ok {
			return err == msg
		}
	}
	return false
}

func TestGmxProject(t *testing.T) {
	pkg := gogen.NewPackage("", "foo", goxConf)
	ctx := &pkgCtx{
		projs:   make(map[string]*gmxProject),
		classes: make(map[*ast.File]*gmxClass),
	}
	gmx := loadClass(ctx, pkg, "main.t2gmx", &ast.File{IsProj: true}, &Config{
		LookupClass: lookupClass,
	})
	scheds := gmx.getScheds(pkg.CB())
	if len(scheds) != 2 || scheds[0] == nil || scheds[0] != scheds[1] {
		t.Fatal("TestGmxProject failed")
	}
	gmx.hasScheds = false
	if gmx.getScheds(nil) != nil {
		t.Fatal("TestGmxProject failed: hasScheds?")
	}

	func() {
		defer func() {
			if e := recover(); e != "class not found: .abcx" {
				t.Fatal("TestGmxProject failed:", e)
			}
		}()
		loadClass(nil, pkg, "main.abcx", &ast.File{IsProj: true}, &Config{
			LookupClass: lookupClass,
		})
	}()
	func() {
		defer func() {
			if e := recover(); e != "multiple project files found: main, main" {
				t.Fatal("TestGmxProject failed:", e)
			}
		}()
		loadClass(ctx, pkg, "main.t2gmx", &ast.File{IsProj: true}, &Config{
			LookupClass: lookupClass,
		})
	}()
}

func TestSpxLookup(t *testing.T) {
	defer func() {
		if e := recover(); e == nil {
			t.Fatal("TestSpxLookup failed: no error?")
		}
	}()
	spxLookup(nil, "foo")
}

func lookupClass(ext string) (c *modfile.Project, ok bool) {
	switch ext {
	case ".t2gmx", ".t2spx":
		return &modfile.Project{
			Ext: ".t2gmx", Class: "Game",
			Works:    []*modfile.Class{{Ext: ".t2spx", Class: "Sprite"}},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/spx2"}}, true
	}
	return
}

func lookupClassErr(ext string) (c *modfile.Project, ok bool) {
	switch ext {
	case ".t2gmx", ".t2spx":
		return &modfile.Project{
			Ext: ".t2gmx", Class: "Game",
			Works:    []*modfile.Class{{Ext: ".t2spx", Class: "Sprite"}},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/libc"}}, true
	}
	return
}

func TestGetGoFile(t *testing.T) {
	if f := genGoFile("a_test.gop", false); f != testingGoFile {
		t.Fatal("TestGetGoFile:", f)
	}
	if f := genGoFile("a_test.xgo", false); f != testingGoFile {
		t.Fatal("TestGetGoFile:", f)
	}
	if f := genGoFile("a_test.gox", true); f != testingGoFile {
		t.Fatal("TestGetGoFile:", f)
	}
	if f := genGoFile("a.gop", false); f != defaultGoFile {
		t.Fatal("TestGetGoFile:", f)
	}
	if f := genGoFile("a.xgo", false); f != defaultGoFile {
		t.Fatal("TestGetGoFile:", f)
	}
}

func TestErrNewType(t *testing.T) {
	testPanic(t, `bar redeclared in this block
	previous declaration at <TODO>
`, func() {
		pkg := types.NewPackage("", "foo")
		newType(pkg, token.NoPos, "bar")
		newType(pkg, token.NoPos, "bar")
	})
}

func TestErrCompileBasicLit(t *testing.T) {
	testPanic(t, "compileBasicLit: invalid syntax\n", func() {
		ctx := &blockCtx{cb: new(gogen.CodeBuilder)}
		compileBasicLit(ctx, &ast.BasicLit{Kind: token.CSTRING, Value: `\\x`})
	})
}

func testPanic(t *testing.T, panicMsg string, doPanic func()) {
	t.Run(panicMsg, func(t *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				t.Fatal("testPanic: no error?")
			} else if msg := e.(string); msg != panicMsg {
				t.Fatalf("\nResult:\n%s\nExpected Panic:\n%s\n", msg, panicMsg)
			}
		}()
		doPanic()
	})
}

func TestClassFileEnd(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "get.yap", `json {"id": ${id} }`, parser.ParseGoPlusClass)
	if err != nil {
		t.Fatal(err)
	}
	if f.End() != f.ShadowEntry.End() {
		t.Fatal("class file end not shadow entry")
	}
}

// -----------------------------------------------------------------------------
