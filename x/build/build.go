/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package build

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/types"
	"path/filepath"

	"github.com/goplus/gogen"
	"github.com/goplus/gogen/packages"
	"github.com/runsys/xmod/modfile"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/parser/fsx/memfs"
	"github.com/goplus/xgo/token"
)

type Class = cl.Class

var (
	projects = make(map[string]*cl.Project)
)

func RegisterClassFileType(ext string, class string, works []*Class, pkgPaths ...string) {
	cls := &cl.Project{
		Ext:      ext,
		Class:    class,
		Works:    works,
		PkgPaths: pkgPaths,
	}
	if ext != "" {
		projects[ext] = cls
	}
	for _, w := range works {
		projects[w.Ext] = cls
	}
}

func init() {
	RegisterClassFileType(".gmx", "Game", []*Class{{Ext: ".spx", Class: "Sprite"}}, "github.com/goplus/spx", "math")
}

type Package struct {
	Fset *token.FileSet
	Pkg  *gogen.Package
}

func (p *Package) ToSource() ([]byte, error) {
	var buf bytes.Buffer
	if err := p.Pkg.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Package) ToAst() *goast.File {
	return p.Pkg.ASTFile()
}

func ClassKind(fname string) (isProj, ok bool) {
	ext := modfile.ClassExt(fname)
	switch ext {
	case ".gmx":
		return true, true
	case ".spx":
		return fname == "main.spx", true
	default:
		if c, ok := projects[ext]; ok {
			for _, w := range c.Works {
				if w.Ext == ext {
					if ext != c.Ext || fname != "main"+ext {
						return false, true
					}
					break
				}
			}
			return true, true
		}
	}
	return
}

type Context struct {
	impl       types.Importer
	fset       *token.FileSet
	LoadConfig func(*cl.Config)
}

func Default() *Context {
	fset := token.NewFileSet()
	impl := packages.NewImporter(fset)
	return NewContext(impl, fset)
}

func NewContext(impl types.Importer, fset *token.FileSet) *Context {
	if fset == nil {
		fset = token.NewFileSet()
	}
	return &Context{impl: impl, fset: fset}
}

func (c *Context) Import(path string) (*types.Package, error) {
	return c.impl.Import(path)
}

func (c *Context) ParseDir(dir string) (*Package, error) {
	pkgs, err := parser.ParseDirEx(c.fset, dir, parser.Config{
		ClassKind: ClassKind,
	})
	if err != nil {
		return nil, err
	}
	return c.loadPackage(dir, pkgs)
}

func (c *Context) ParseFSDir(fs parser.FileSystem, dir string) (*Package, error) {
	pkgs, err := parser.ParseFSDir(c.fset, fs, dir, parser.Config{
		ClassKind: ClassKind,
	})
	if err != nil {
		return nil, err
	}
	return c.loadPackage(dir, pkgs)
}

func (c *Context) ParseFile(file string, src any) (*Package, error) {
	fs, err := memfs.File(file, src)
	if err != nil {
		return nil, err
	}
	return c.ParseFSDir(fs, filepath.Dir(file))
}

func (c *Context) loadPackage(srcDir string, pkgs map[string]*ast.Package) (*Package, error) {
	mainPkg, ok := pkgs["main"]
	if !ok {
		for _, v := range pkgs {
			mainPkg = v
			break
		}
	}
	conf := &cl.Config{Fset: c.fset}
	conf.Importer = c
	conf.LookupClass = func(ext string) (c *cl.Project, ok bool) {
		c, ok = projects[ext]
		return
	}
	if c.LoadConfig != nil {
		c.LoadConfig(conf)
	}
	out, err := cl.NewPackage("", mainPkg, conf)
	if err != nil {
		return nil, err
	}
	return &Package{c.fset, out}, nil
}

func (ctx *Context) BuildFile(filename string, src any) (data []byte, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("compile %v failed. %v", filename, r)
		}
	}()
	pkg, err := ctx.ParseFile(filename, src)
	if err != nil {
		return nil, err
	}
	return pkg.ToSource()
}

func (ctx *Context) BuildFSDir(fs parser.FileSystem, dir string) (data []byte, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("compile %v failed. %v", dir, err)
		}
	}()
	pkg, err := ctx.ParseFSDir(fs, dir)
	if err != nil {
		return nil, err
	}
	return pkg.ToSource()
}

func (ctx *Context) BuildDir(dir string) (data []byte, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("compile %v failed. %v", dir, err)
		}
	}()
	pkg, err := ctx.ParseDir(dir)
	if err != nil {
		return nil, err
	}
	return pkg.ToSource()
}
