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

package printer_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/format"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/parser/parsertest"
	"github.com/goplus/xgo/printer"
	"github.com/goplus/xgo/token"
)

func init() {
	printer.SetDebug(printer.DbgFlagAll)
}

func TestNoPkgDecl(t *testing.T) {
	var dst bytes.Buffer
	fset := token.NewFileSet()
	if err := format.Node(&dst, fset, &ast.File{
		Name:      &ast.Ident{Name: "main"},
		NoPkgDecl: true,
	}); err != nil {
		t.Fatal("format.Node failed:", err)
	}
	if dst.String() != "\n" {
		t.Fatal("TestNoPkgDecl:", dst.String())
	}
}

func TestFuncs(t *testing.T) {
	var dst bytes.Buffer
	fset := token.NewFileSet()
	if err := format.Node(&dst, fset, &ast.File{
		Name: &ast.Ident{Name: "main"},
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Name: &ast.Ident{Name: "foo"},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&printer.NewlineStmt{}},
				},
			},
			&ast.FuncDecl{
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Name: &ast.Ident{Name: "bar"},
				Body: &ast.BlockStmt{},
			},
			&ast.FuncDecl{
				Type:   &ast.FuncType{Params: &ast.FieldList{}},
				Name:   &ast.Ident{Name: "Classname"},
				Body:   &ast.BlockStmt{},
				Shadow: true,
			},
		},
		NoPkgDecl: true,
	}); err != nil {
		t.Fatal("format.Node failed:", err)
	}
	if dst.String() != `func foo() {

}

func bar() {
}
` {
		t.Fatal("TestNoPkgDecl:", dst.String())
	}
}

func TestForPhraseStmt(t *testing.T) {
	for _, tt := range []struct{ name, src, want string }{
		{"Arrow", "for v <- [1] { println v }", "for v in [1] {\n\tprintln v\n}"},
		{"In", "for v in [1] { println v }", "for v in [1] {\n\tprintln v\n}"},
		{"Filtered", "for i, v <- [1], v > 0 { println i, v }", "for i, v in [1] if v > 0 {\n\tprintln i, v\n}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseEntry(fset, "main.xgo", tt.src, parser.Config{})
			if err != nil {
				t.Fatal(err)
			}
			stmt := file.ShadowEntry.Body.List[0]
			var dst bytes.Buffer
			if err := format.Node(&dst, fset, stmt); err != nil {
				t.Fatal(err)
			}
			diffBytes(t, dst.Bytes(), []byte(tt.want))
		})
	}
}

func TestForPhraseStmtInFunc(t *testing.T) {
	const want = "func run() {\n\tfor v in [1] {\n\t\tprintln v\n\t}\n}\n"
	for _, tt := range []struct{ name, src string }{
		{"OneLine", "func run() { for v <- [1] { println v } }\n"},
		{"MultiLine", "func run() {\nfor v <- [1] {\nprintln v\n}\n}\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, err := format.Source([]byte(tt.src), nil, "main.xgo")
			if err != nil {
				t.Fatal(err)
			}
			diffBytes(t, res, []byte(want))

			fset := token.NewFileSet()
			file, err := parser.ParseEntry(fset, "main.xgo", tt.src, parser.Config{})
			if err != nil {
				t.Fatal(err)
			}
			var dst bytes.Buffer
			if err := format.Node(&dst, fset, file); err != nil {
				t.Fatal(err)
			}
			diffBytes(t, dst.Bytes(), []byte(want))

			res, err = format.Source(res, nil, "main.xgo")
			if err != nil {
				t.Fatal(err)
			}
			diffBytes(t, res, []byte(want))
		})
	}
}

func diffBytes(t *testing.T, dst, src []byte) {
	line := 1
	offs := 0 // line offset
	for i := 0; i < len(dst) && i < len(src); i++ {
		d := dst[i]
		s := src[i]
		if d != s {
			t.Errorf("dst:%d: %s\n", line, dst[offs:])
			t.Errorf("src:%d: %s\n", line, src[offs:])
			return
		}
		if s == '\n' {
			line++
			offs = i + 1
		}
	}
	if len(dst) != len(src) {
		t.Errorf("len(dst) = %d, len(src) = %d\ndst = %q\nsrc = %q", len(dst), len(src), dst, src)
	}
}

const (
	excludeFormatSource = 1
	excludeFormatNode   = 2
)

func testFrom(t *testing.T, fpath, sel string, mode int) {
	if sel != "" && !strings.Contains(fpath, sel) {
		return
	}
	src, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatal(err)
	}

	if (mode & excludeFormatSource) == 0 {
		t.Run("format.Source "+fpath, func(t *testing.T) {
			res, err := format.Source(src, nil, fpath)
			if err != nil {
				t.Fatal("Source failed:", err)
			}
			diffBytes(t, res, src)
		})
	}

	if (mode & excludeFormatNode) == 0 {
		t.Run("format.Node "+fpath, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseEntry(fset, fpath, src, parser.Config{
				ClassInfo: parsertest.ClassInfo,
				Mode:      parser.ParseComments,
			})
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			err = format.Node(&buf, fset, f)
			if err != nil {
				t.Fatal(err)
			}
			diffBytes(t, buf.Bytes(), src)
		})
	}
}

func TestFromGopPrinter(t *testing.T) {
	testFrom(t, "nodes.go", "", 0)
	testFrom(t, "printer.go", "", 0)
	testFrom(t, "printer_test.go", "", 0)
}

func TestFromTestdata(t *testing.T) {
	sel := ""
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = filepath.Join(dir, "./_testdata")
	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".xgo" {
			testFrom(t, path, sel, 0)
		}
		return nil
	})
}

func TestFromParse(t *testing.T) {
	sel := ""
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = filepath.Join(dir, "../parser/_testdata")
	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		ext := filepath.Ext(name)
		if !info.IsDir() && (ext == ".xgo" || ext == ".gox" || ext == ".gsh" || ext == ".spx") {
			testFrom(t, path, sel, excludeFormatSource)
		}
		return nil
	})
}
