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
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
	"testing"
)

func TestToIndex(t *testing.T) {
	if ret := toIndex('a'); ret != 10 {
		t.Fatal("toIndex:", ret)
	}
	defer func() {
		if e := recover(); e != "invalid character out of [0-9,a-z]" {
			t.Fatal("panic:", e)
		}
	}()
	toIndex('A')
}

func TestCheckTypeMethod(t *testing.T) {
	if ret := checkTypeMethod("_Foo_a"); ret.typ != "" || ret.name != "Foo_a" {
		t.Fatal("checkTypeMethod:", ret)
	}
	if ret := checkTypeMethod("Foo_a"); ret.typ != "Foo" || ret.name != "a" {
		t.Fatal("checkTypeMethod:", ret)
	}
}

func TestIsXGoPackage(t *testing.T) {
	if isXGoPackage(&doc.Package{}) {
		t.Fatal("isXGoPackage: true?")
	}
}

func TestDocRecv(t *testing.T) {
	if _, ok := docRecv(&ast.Field{}); ok {
		t.Fatal("docRecv: ok?")
	}
}

func printVal(parts []string, format string, val any) []string {
	return append(parts, fmt.Sprintf(format, val))
}

func printFuncDecl(parts []string, fset *token.FileSet, decl *ast.FuncDecl) []string {
	var b bytes.Buffer
	if e := format.Node(&b, fset, decl); e != nil {
		panic(e)
	}
	return append(parts, b.String())
}

func printFunc(parts []string, fset *token.FileSet, format string, fn *doc.Func) []string {
	parts = printVal(parts, format, fn.Name)
	if fn.Recv != "" {
		parts = printVal(parts, "Recv: %s", fn.Recv)
	}
	parts = printVal(parts, "Doc: %s", fn.Doc)
	parts = printFuncDecl(parts, fset, fn.Decl)
	return parts
}

func printFuncs(parts []string, fset *token.FileSet, fns []*doc.Func) []string {
	for _, fn := range fns {
		parts = printFunc(parts, fset, "== Func %s ==", fn)
	}
	return parts
}

func printType(parts []string, fset *token.FileSet, typ *doc.Type) []string {
	parts = append(parts, fmt.Sprintf("== Type %s ==", typ.Name))
	for _, fn := range typ.Funcs {
		parts = printFunc(parts, fset, "- Func %s -", fn)
	}
	for _, fn := range typ.Methods {
		parts = printFunc(parts, fset, "- Method %s -", fn)
	}
	return parts
}

func printTypes(parts []string, fset *token.FileSet, types []*doc.Type) []string {
	for _, typ := range types {
		parts = printType(parts, fset, typ)
	}
	return parts
}

func printPkg(fset *token.FileSet, in *doc.Package) string {
	var parts []string
	parts = printFuncs(parts, fset, in.Funcs)
	parts = printTypes(parts, fset, in.Types)
	return strings.Join(append(parts, ""), "\n")
}

func testPkg(t *testing.T, filename string, src any, expected string) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := doc.NewFromFiles(fset, []*ast.File{f}, "foo")
	if err != nil {
		t.Fatal(err)
	}
	pkg = Transform(pkg)
	if ret := printPkg(fset, pkg); ret != expected {
		t.Fatalf("got:\n%s\nexpected:\n%s\n", ret, expected)
	}
}

func testFromDir(t *testing.T, sel, relDir string) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = path.Join(dir, relDir)
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if strings.HasPrefix(name, "_") || !fi.IsDir() || (sel != "" && !strings.Contains(name, sel)) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testDir := dir + "/" + name
			out, e := os.ReadFile(testDir + "/out.expect")
			if e != nil {
				t.Fatal(e)
			}
			testPkg(t, testDir+"/in.go", nil, string(out))
		})
	}
}

func TestFromTestdata(t *testing.T) {
	testFromDir(t, "", "./_testdata")
}
