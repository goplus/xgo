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

package parser

import (
	"io/fs"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
	fsx "github.com/qiniu/x/http/fs"
)

// -----------------------------------------------------------------------------

type fileInfo struct {
	*fsx.FileInfo
}

func (p fileInfo) Info() (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func TestFilter(t *testing.T) {
	d := fsx.NewFileInfo("foo.go", 10)
	if filter(d, func(fi fs.FileInfo) bool {
		return false
	}) {
		t.Fatal("TestFilter: true?")
	}
	if !filter(d, func(fi fs.FileInfo) bool {
		return true
	}) {
		t.Fatal("TestFilter: false?")
	}
	d2 := fileInfo{d}
	if filter(d2, func(fi fs.FileInfo) bool {
		return true
	}) {
		t.Fatal("TestFilter: true?")
	}
}

func TestAssert(t *testing.T) {
	defer func() {
		if e := recover(); e != "go/parser internal error: panic msg" {
			t.Fatal("TestAssert:", e)
		}
	}()
	assert(false, "panic msg")
}

func panicMsg(e any) string {
	switch v := e.(type) {
	case string:
		return v
	case error:
		return v.Error()
	}
	return ""
}

func testErrCode(t *testing.T, code string, errExp, panicExp string) {
	defer func() {
		if e := recover(); e != nil {
			if panicMsg(e) != panicExp {
				t.Fatal("testErrCode panic:", e)
			}
		}
	}()
	t.Helper()
	fset := token.NewFileSet()
	_, err := Parse(fset, "/foo/bar.xgo", code, 0)
	if err == nil || err.Error() != errExp {
		t.Fatal("testErrCode error:", err)
	}
}

func testShadowEntry(t *testing.T, code string, errExp string, decl *ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := ParseEntry(fset, "/foo/bar.xgo", code, Config{
		Mode: ParseXGoClass | ParseComments | AllErrors,
	})
	if err != nil && err.Error() != errExp {
		t.Fatal("testShadowEntry error:", err)
	}
	if err == nil && errExp != "" {
		t.Fatal("testShadowEntry: nil, errExp:", errExp)
	}

	if f.ShadowEntry == nil {
		t.Fatal("testShadowEntry: nil")
	}
	if f.ShadowEntry.Body.Lbrace != decl.Body.Lbrace || f.ShadowEntry.Body.Rbrace != decl.Body.Rbrace {
		t.Fatal("testShadowEntry: brace mismatch", f.ShadowEntry.Body.Lbrace, decl.Body.Lbrace, f.ShadowEntry.Body.Rbrace, decl.Body.Rbrace)
	}
}

func testErrCodeParseExpr(t *testing.T, code string, errExp, panicExp string) {
	defer func() {
		if e := recover(); e != nil {
			if panicMsg(e) != panicExp {
				t.Fatal("testErrCodeParseExpr panic:", e)
			}
		}
	}()
	t.Helper()
	_, err := ParseExpr(code)
	if err == nil || err.Error() != errExp {
		t.Fatal("testErrCodeParseExpr error:", err)
	}
}

func testClassErrCode(t *testing.T, code string, errExp, panicExp string) {
	defer func() {
		if e := recover(); e != nil {
			if panicMsg(e) != panicExp {
				t.Fatal("testErrCode panic:", e)
			}
		}
	}()
	t.Helper()
	fset := token.NewFileSet()
	_, err := Parse(fset, "/foo/bar.gox", code, ParseXGoClass)
	if err == nil || err.Error() != errExp {
		t.Fatal("testErrCode error:", err)
	}
}

func TestErrLabel(t *testing.T) {
	testErrCode(t, `a.x:`, `/foo/bar.xgo:1:4: illegal label declaration`, ``)
}

func TestErrTplLit(t *testing.T) {
	testErrCode(t, "tpl`a =`", `/foo/bar.xgo:1:8: expected ';', found 'EOF' (and 1 more errors)`, ``)
}

func TestErrTuple(t *testing.T) {
	testErrCode(t, `println (1,2)*2`, `/foo/bar.xgo:1:9: tuple is not supported`, ``)
	testErrCode(t, `println 2*(1,2)`, `/foo/bar.xgo:1:13: expected ')', found ','`, ``)
	testErrCode(t, `func test() (int,int) { return (100,100)`, `/foo/bar.xgo:1:32: tuple is not supported`, ``)
}

func TestErrOperand(t *testing.T) {
	testErrCode(t, `a :=`, `/foo/bar.xgo:1:5: expected operand, found 'EOF'`, ``)
}

func TestErrMissingComma(t *testing.T) {
	testErrCode(t, `func a(b int c)`, `/foo/bar.xgo:1:14: missing ',' in parameter list`, ``)
}

func TestErrLambda(t *testing.T) {
	testErrCode(t, `func test(v string, f func( int)) {
}
test "hello" => {
	println "lambda",x
}
`, `/foo/bar.xgo:3:6: expected 'IDENT', found "hello"`, ``)
	testErrCode(t, `func test(v string, f func( int)) {
}
test "hello", "x" => {
	println "lambda",x
}
`, `/foo/bar.xgo:3:15: expected 'IDENT', found "x"`, ``)
	testErrCode(t, `func test(v string, f func( int)) {
}
test "hello", ("x") => {
	println "lambda",x
}
`, `/foo/bar.xgo:3:16: expected 'IDENT', found "x"`, ``)
	testErrCode(t, `func test(v string, f func(int,int)) {
}
test "hello", (x, "y") => {
	println "lambda",x,y
}
`, `/foo/bar.xgo:3:19: expected 'IDENT', found "y"`, ``)
	testErrCode(t, `onTouchStart "someone" => {
	say "touched by someone"
}
`, `/foo/bar.xgo:1:14: expected 'IDENT', found "someone"`, ``)
}

func TestErrTooManyParseExpr(t *testing.T) {
	testErrCodeParseExpr(t, `func() int {
  var
  var
  var
  var
  var
  var
  var
  var
  var
  var
  var
  var
}()
`, `3:3: expected 'IDENT', found 'var' (and 10 more errors)`, ``)
}

func TestErrTooMany(t *testing.T) {
	testErrCode(t, `
func f() { var }
func g() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
func h() { var }
`, `/foo/bar.xgo:2:16: expected 'IDENT', found '}' (and 10 more errors)`, ``)
}

func TestErrInFunc(t *testing.T) {
	testErrCode(t, `func test() {
	a,
}`, `/foo/bar.xgo:2:2: expected 1 expression (and 2 more errors)`, ``)
	testErrCode(t, `func test() {
	a.test, => {
	}
}`, `/foo/bar.xgo:2:10: expected operand, found '=>' (and 1 more errors)`, ``)
	testErrCode(t, `func test() {
		,
	}
}`, `/foo/bar.xgo:2:3: expected statement, found ',' (and 1 more errors)`, ``)
}

// -----------------------------------------------------------------------------

func TestClassErrCode(t *testing.T) {
	testClassErrCode(t, `var (
	A,B
	v int
)
`, `/foo/bar.gox:2:2: missing variable type or initialization`, ``)
	testClassErrCode(t, `var (
	A.*B
	v int
)
`, `/foo/bar.gox:2:4: expected 'IDENT', found '*' (and 2 more errors)`, ``)
	testClassErrCode(t, `var (
	[]A
	v int
)
`, `/foo/bar.gox:2:2: expected 'IDENT', found '['`, ``)
	testClassErrCode(t, `var (
	*[]A
	v int
)
`, `/foo/bar.gox:2:3: expected 'IDENT', found '['`, ``)
	testClassErrCode(t, `
var (
)
const c = 100
const d
`, `/foo/bar.gox:5:7: missing constant value`, ``)
}

func TestErrGlobal(t *testing.T) {
	testErrCode(t, `func test() {}
}`, `/foo/bar.xgo:2:1: expected statement, found '}'`, ``)
}

func TestErrCompositeLiteral(t *testing.T) {
	testErrCode(t, `println (T[int]){a: 1, b: 2}
`, `/foo/bar.xgo:1:10: cannot parenthesize type in composite literal`, ``)
}

func TestErrSelectorExpr(t *testing.T) {
	testErrCode(t, `
x.
*p
`, `/foo/bar.xgo:3:1: expected selector or type assertion, found '*'`, ``)
}

func TestErrStringLitEx(t *testing.T) {
	testErrCode(t, `
println "${ ... }"
`, "/foo/bar.xgo:2:13: expected operand, found '...'", ``)
	testErrCode(t, `
println "${b"
`, "/foo/bar.xgo:2:11: invalid $ expression: ${ doesn't end with }", ``)
	testErrCode(t, `
println "$a${b}"
`, "/foo/bar.xgo:2:10: invalid $ expression: neither `${ ... }` nor `$$`", ``)
}

func TestErrStringLiteral(t *testing.T) {
	testErrCode(t, `run "
`, `/foo/bar.xgo:1:5: string literal not terminated`, ``)
}

func TestErrFieldDecl(t *testing.T) {
	testErrCode(t, `
type T struct {
	*(Foo)
}
`, `/foo/bar.xgo:3:3: cannot parenthesize embedded type`, ``)
	testErrCode(t, `
type T struct {
	(Foo)
}
`, `/foo/bar.xgo:3:2: cannot parenthesize embedded type`, ``)
	testErrCode(t, `
type T struct {
	(*Foo)
}
`, `/foo/bar.xgo:3:2: cannot parenthesize embedded type`, ``)
}

func TestParseFieldDecl(t *testing.T) {
	var p parser
	p.init(token.NewFileSet(), "/foo/bar.xgo", []byte(`type T struct {
}
`), 0)
	p.parseFieldDecl(nil)
}

func TestCheckExpr(t *testing.T) {
	var p parser
	p.init(token.NewFileSet(), "/foo/bar.xgo", []byte(``), 0)
	p.checkExpr(&ast.Ellipsis{})
	p.checkExpr(&ast.ElemEllipsis{})
	p.checkExpr(&ast.StarExpr{})
	p.checkExpr(&ast.IndexListExpr{})
	p.checkExpr(&ast.FuncType{})
	p.checkExpr(&ast.FuncLit{})
}

func TestErrTupleType(t *testing.T) {
	testErrCode(t, `var a (a float64, x int, chan int)
`, `/foo/bar.xgo:1:8: mixed named and unnamed fields in tuple type`, ``)
	testErrCode(t, `var a (a float64, x int, string)
`, `/foo/bar.xgo:1:8: mixed named and unnamed fields in tuple type`, ``)
}

func TestErrFuncDecl(t *testing.T) {
	testErrCode(t, `func test()
{
}
`, `/foo/bar.xgo:2:1: unexpected semicolon or newline before {`, ``)
	testErrCode(t, `func test() +1
`, `/foo/bar.xgo:1:13: expected ';', found '+'`, ``)
	testErrCode(t, `
func (a T) +{}
`, `/foo/bar.xgo:2:12: expected type, found '+'`, ``)
	testErrCode(t, `func +(a T, b T) {}
`, `/foo/bar.xgo:1:6: overload operator can only have one parameter`, ``)
}

func TestErrForIn(t *testing.T) {
	testErrCode(t, `x := [a for a i b]
`, `/foo/bar.xgo:1:15: expected 'in', found i`, ``)
}

func TestErrKwargExpr(t *testing.T) {
	testErrCode(t, `
f a=1, 13
`, `/foo/bar.xgo:2:8: positional argument follows keyword argument`, ``)
}

func TestNumberUnitLit(t *testing.T) {
	var p parser
	p.checkExpr(&ast.NumberUnitLit{})
	p.toIdent(&ast.NumberUnitLit{})
}

func TestImplicitIdent(t *testing.T) {
	if ast.NewIdentEx(100, "foo", ast.ImplicitPkg).End() != 100 {
		t.Fatal("TestImplicitPkg: not 100")
	}
	if ast.NewIdentEx(100, "foo", ast.ImplicitFun).End() != 100 {
		t.Fatal("TestImplicitFun: not 100")
	}
	if ast.NewIdentEx(100, "foo", ast.Fun).End() != 103 {
		t.Fatal("TestFun: not 103")
	}
}

func TestErrGlobalVarWithSyntaxError(t *testing.T) {
	// Parse the code
	testShadowEntry(t, `var (
	foo int.=2
	sprites []Sprite
)

func reset() {
	foo = 10
	sprites = make([]Sprite, 0)
}

onStart => {
	reset()
}
`, `/foo/bar.xgo:2:10: expected 'IDENT', found '=' (and 19 more errors)`, &ast.FuncDecl{
		Body: &ast.BlockStmt{
			Lbrace: 119,
			Rbrace: 121,
		},
	})

	testShadowEntry(t, `var (
	foo int
	sprites []Sprite
)

func reset() {
	foo = 10
	sprites = make([]Sprite, 0)
}

onStart => {
	reset()
}`, "", &ast.FuncDecl{
		Body: &ast.BlockStmt{
			Lbrace: 94,
			Rbrace: 117,
		},
	})
}

// -----------------------------------------------------------------------------
