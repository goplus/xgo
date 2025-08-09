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

package cltest

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	goparser "go/parser"

	"github.com/goplus/gogen"
	"github.com/goplus/mod"
	"github.com/runsys/xmod/env"
	"github.com/runsys/xmod/modfile"
	"github.com/runsys/xmod/xgomod"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/parser/fsx"
	"github.com/goplus/xgo/parser/fsx/memfs"
	"github.com/goplus/xgo/scanner"
	"github.com/goplus/xgo/token"
	"github.com/goplus/xgo/tool"
	"github.com/qiniu/x/test"
)

var (
	XGoRoot string
	XGo     *env.XGo
	Conf    *cl.Config
)

func init() {
	XGo = &env.XGo{Version: "1.0"}
	gogen.SetDebug(gogen.DbgFlagAll)
	cl.SetDebug(cl.DbgFlagAll | cl.FlagNoMarkAutogen)
	fset := token.NewFileSet()
	imp := tool.NewImporter(nil, XGo, fset)
	XGoRoot, _, _ = mod.FindGoMod("")
	Conf = &cl.Config{
		Fset:          fset,
		Importer:      imp,
		Recorder:      gopRecorder{},
		LookupClass:   LookupClass,
		NoFileLine:    true,
		NoAutoGenMain: true,
	}
}

// -----------------------------------------------------------------------------

func LookupClass(ext string) (c *modfile.Project, ok bool) {
	switch ext {
	case ".tgmx", ".tspx":
		return &modfile.Project{
			Ext: ".tgmx", Class: "*MyGame",
			Works:    []*modfile.Class{{Ext: ".tspx", Class: "Sprite"}},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/spx", "math"}}, true
	case ".t2gmx", ".t2spx":
		return &modfile.Project{
			Ext: ".t2gmx", Class: "Game",
			Works: []*modfile.Class{
				{Ext: ".t2spx", Class: "Sprite"},
			},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/spx2"}}, true
	case ".t4gmx", ".t4spx":
		return &modfile.Project{
			Ext: ".t4gmx", Class: "*MyGame",
			Works:    []*modfile.Class{{Ext: ".t4spx", Class: "Sprite"}},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/spx4", "math"}}, true
	case "_spx.gox":
		return &modfile.Project{
			Ext: "_spx.gox", Class: "Game",
			Works:    []*modfile.Class{{Ext: "_spx.gox", Class: "Sprite"}},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/spx3", "math"},
			Import:   []*modfile.Import{{Path: "github.com/goplus/xgo/cl/internal/spx3/jwt"}}}, true
	case "_xtest.gox":
		return &modfile.Project{
			Ext: "_xtest.gox", Class: "App",
			Works:    []*modfile.Class{{Ext: "_xtest.gox", Class: "Case"}},
			PkgPaths: []string{"github.com/goplus/xgo/test", "testing"}}, true
	case "_mcp.gox", "_tool.gox", "_prompt.gox":
		return &modfile.Project{
			Ext: "_mcp.gox", Class: "Game",
			Works: []*modfile.Class{
				{Ext: "_tool.gox", Class: "Tool", Proto: "ToolProto", Prefix: "Tool_"},
				{Ext: "_prompt.gox", Class: "Prompt", Proto: "PromptProto", Embedded: true},
				{Ext: "_res.gox", Class: "Resource", Proto: "ResourceProto"},
			},
			PkgPaths: []string{"github.com/goplus/xgo/cl/internal/mcp"}}, true
	}
	return
}

// -----------------------------------------------------------------------------

func Named(t *testing.T, name string, gopcode, expected string) {
	t.Run(name, func(t *testing.T) {
		Do(t, gopcode, expected)
	})
}

func Do(t *testing.T, gopcode, expected string) {
	DoExt(t, Conf, "main", gopcode, expected)
}

func DoWithFname(t *testing.T, gopcode, expected string, fname string) {
	fs := memfs.SingleFile("/foo", fname, gopcode)
	DoFS(t, Conf, fs, "/foo", nil, "main", expected)
}

func DoExt(t *testing.T, conf *cl.Config, pkgname, gopcode, expected string) {
	fs := memfs.SingleFile("/foo", "bar.xgo", gopcode)
	DoFS(t, conf, fs, "/foo", nil, pkgname, expected)
}

func Mixed(t *testing.T, pkgname, gocode, gopcode, expected string, outline ...bool) {
	conf := *Conf
	conf.Outline = (outline != nil && outline[0])
	fs := memfs.TwoFiles("/foo", "a.go", gocode, "b.xgo", gopcode)
	DoFS(t, &conf, fs, "/foo", nil, pkgname, expected)
}

// -----------------------------------------------------------------------------

func DoFS(
	t *testing.T, conf *cl.Config,
	fs parser.FileSystem, dir string, filter func(fs.FileInfo) bool, pkgname string, exp any) {
	cl.SetDisableRecover(true)
	defer cl.SetDisableRecover(false)

	fset := conf.Fset
	pkgs, err := parser.ParseFSDir(fset, fs, dir, parser.Config{
		Mode:   parser.ParseComments,
		Filter: filter,
	})
	if err != nil {
		scanner.PrintError(os.Stderr, err)
		t.Fatal("ParseFSDir:", err)
	}
	bar := pkgs[pkgname]
	pkg, err := cl.NewPackage("github.com/goplus/xgo/cl", bar, conf)
	if err != nil {
		t.Fatal("NewPackage:", err)
	}
	var b bytes.Buffer
	err = pkg.WriteTo(&b)
	if err != nil {
		t.Fatal("gogen.WriteTo failed:", err)
	}
	if expected, ok := exp.(string); ok {
		result := b.String()
		if result != expected {
			t.Fatalf("\nResult:\n%s\nExpected:\n%s\n", result, expected)
		}
	} else if test.Diff(t, dir+"/result.txt", b.Bytes(), exp.([]byte)) {
		t.Fatal(dir, ": unexpect result")
	}
}

// -----------------------------------------------------------------------------

func FromDir(t *testing.T, sel, relDir string) {
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
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testFrom(t, dir+"/"+name, sel)
		})
	}
}

func testFrom(t *testing.T, pkgDir, sel string) {
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}
	log.Println("Parsing", pkgDir)
	out := pkgDir + "/out.go"
	b, _ := os.ReadFile(out)
	filter := func(fi fs.FileInfo) bool {
		return fi.Name() == "in.xgo"
	}
	conf := Conf
	goMod := pkgDir + "/go.mod"
	if _, err := os.Stat(goMod); err == nil {
		if mod, err := xgomod.Load(pkgDir); err == nil {
			confCopy := *Conf
			confCopy.Importer = tool.NewImporter(mod, XGo, conf.Fset)
			conf = &confCopy
		}
	} else {
		confCopy := *Conf
		confCopy.RelativeBase = XGoRoot
		conf = &confCopy
	}
	DoFS(t, conf, fsx.Local, pkgDir, filter, "main", b)
}

func Go1Point() int {
	for i := len(build.Default.ReleaseTags) - 1; i >= 0; i-- {
		var version int
		if _, err := fmt.Sscanf(build.Default.ReleaseTags[i], "go1.%d", &version); err != nil {
			continue
		}
		return version
	}
	panic("bad release tags")
}

func EnableTypesalias() bool {
	ver := Go1Point()
	if ver < 22 {
		return false
	} else if ver > 22 {
		_, ok := types.Universe.Lookup("any").Type().(*types.Interface)
		return !ok
	} else {
		fset := token.NewFileSet()
		f, _ := goparser.ParseFile(fset, "a.go", "package p; type A = int", goparser.SkipObjectResolution)
		pkg, _ := new(types.Config).Check("p", fset, []*ast.File{f}, nil)
		_, ok := pkg.Scope().Lookup("A").Type().(*types.Basic)
		return !ok
	}
}

// -----------------------------------------------------------------------------
