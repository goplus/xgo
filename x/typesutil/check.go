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

package typesutil

import (
	goast "go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/token"
	"github.com/goplus/xgo/x/typesutil/internal/typesutil"
	"github.com/qiniu/x/errors"
	"github.com/qiniu/x/log"
)

// -----------------------------------------------------------------------------

type dbgFlags int

const (
	DbgFlagVerbose dbgFlags = 1 << iota
	DbgFlagPrintError
	DbgFlagDisableRecover
	DbgFlagDefault = DbgFlagVerbose | DbgFlagPrintError
	DbgFlagAll     = DbgFlagDefault | DbgFlagDisableRecover
)

var (
	debugVerbose  bool
	debugPrintErr bool
)

func SetDebug(flags dbgFlags) {
	debugVerbose = (flags & DbgFlagVerbose) != 0
	debugPrintErr = (flags & DbgFlagPrintError) != 0
	if (flags & DbgFlagDisableRecover) != 0 {
		cl.SetDisableRecover(true)
	}
}

// -----------------------------------------------------------------------------

type Project = cl.Project

type Config struct {
	// Types provides type information for the package (required).
	Types *types.Package

	// Fset provides source position information for syntax trees and types (required).
	Fset *token.FileSet

	// WorkingDir is the directory in which to run xgo compiler (optional).
	// If WorkingDir is not set, os.Getwd() is used.
	WorkingDir string

	// Mod represents an XGo module (optional).
	Mod *xgomod.Module

	// If IgnoreFuncBodies is set, skip compiling function bodies (optional).
	IgnoreFuncBodies bool

	// If UpdateGoTypesOverload is set, update go types overload data (optional).
	UpdateGoTypesOverload bool
}

// A Checker maintains the state of the type checker.
// It must be created with NewChecker.
type Checker struct {
	conf    *types.Config
	opts    *Config
	goInfo  *types.Info
	xgoInfo *Info
}

// NewChecker returns a new Checker instance for a given package.
// Package files may be added incrementally via checker.Files.
func NewChecker(conf *types.Config, opts *Config, goInfo *types.Info, xgoInfo *Info) *Checker {
	return &Checker{conf, opts, goInfo, xgoInfo}
}

// Files checks the provided files as part of the checker's package.
func (p *Checker) Files(goFiles []*goast.File, xgoFiles []*ast.File) (err error) {
	opts := p.opts
	pkgTypes := opts.Types
	fset := opts.Fset
	conf := p.conf

	// Save original Error handler and restore it on function exit
	origError := conf.Error
	defer func() {
		conf.Error = origError
	}()

	if len(xgoFiles) == 0 {
		onErr := conf.Error
		if onErr != nil {
			conf.Error = func(err error) {
				if e, ok := convGoErr(err); ok {
					onErr(e)
				}
			}
		}
		checker := types.NewChecker(conf, fset, pkgTypes, p.goInfo)
		return checker.Files(goFiles)
	}
	files := make([]*goast.File, 0, len(goFiles))
	gofs := make(map[string]*goast.File)
	xgofs := make(map[string]*ast.File)
	for _, goFile := range goFiles {
		f := fset.File(goFile.Pos())
		if f == nil {
			continue
		}
		file := f.Name()
		fname := filepath.Base(file)
		if strings.HasPrefix(fname, "xgo_autogen") {
			continue
		}
		gofs[file] = goFile
		files = append(files, goFile)
	}
	for _, xgoFile := range xgoFiles {
		f := fset.File(xgoFile.Pos())
		if f == nil {
			continue
		}
		xgofs[f.Name()] = xgoFile
	}
	if debugVerbose {
		log.Println("typesutil.Check:", pkgTypes.Path(), "xgoFiles =", len(xgofs), "goFiles =", len(gofs))
	}
	pkg := &ast.Package{
		Name:    pkgTypes.Name(),
		Files:   xgofs,
		GoFiles: gofs,
	}
	mod := opts.Mod
	if mod == nil {
		mod = xgomod.Default
	}
	_, err = cl.NewPackage(pkgTypes.Path(), pkg, &cl.Config{
		Types:          pkgTypes,
		Fset:           fset,
		LookupClass:    mod.LookupClass,
		Importer:       conf.Importer,
		Recorder:       NewRecorder(p.xgoInfo),
		NoFileLine:     true,
		NoAutoGenMain:  true,
		NoSkipConstant: true,
		Outline:        opts.IgnoreFuncBodies,
	})
	if err != nil {
		if onErr := conf.Error; onErr != nil {
			if list, ok := err.(errors.List); ok {
				for _, e := range list {
					if ce, ok := convErr(fset, e); ok {
						onErr(ce)
					}
				}
			} else if ce, ok := convErr(fset, err); ok {
				onErr(ce)
			} else {
				onErr(err)
			}
		}
		if debugPrintErr {
			log.Println("typesutil.Check err:", err)
			log.SingleStack()
		}
		// don't return even if err != nil
	}
	if len(files) > 0 {
		onErr := conf.Error
		if onErr != nil {
			conf.Error = func(err error) {
				if e, ok := convGoErr(err); ok {
					onErr(e)
				}
			}
		}
		scope := pkgTypes.Scope()
		objMap := DeleteObjects(scope, files)
		checker := types.NewChecker(conf, fset, pkgTypes, p.goInfo)
		err = checker.Files(files)
		// TODO(xsw): how to process error?
		CorrectTypesInfo(scope, objMap, p.xgoInfo.Uses)
		if opts.UpdateGoTypesOverload {
			gogen.InitThisGopPkg(pkgTypes)
		}
	}
	return
}

type astIdent interface {
	comparable
	ast.Node
}

type objMapT = map[types.Object]types.Object

// CorrectTypesInfo corrects types info to avoid there are two instances for the same Go object.
func CorrectTypesInfo[Ident astIdent](scope *types.Scope, objMap objMapT, uses map[Ident]types.Object) {
	for o := range objMap {
		objMap[o] = scope.Lookup(o.Name())
	}
	for id, old := range uses {
		if new := objMap[old]; new != nil {
			uses[id] = new
		}
	}
}

// DeleteObjects deletes all objects defined in Go files and returns deleted objects.
func DeleteObjects(scope *types.Scope, files []*goast.File) objMapT {
	objMap := make(objMapT)
	for _, f := range files {
		for _, decl := range f.Decls {
			switch v := decl.(type) {
			case *goast.GenDecl:
				for _, spec := range v.Specs {
					switch v := spec.(type) {
					case *goast.ValueSpec:
						for _, name := range v.Names {
							scopeDelete(objMap, scope, name.Name)
						}
					case *goast.TypeSpec:
						scopeDelete(objMap, scope, v.Name.Name)
					}
				}
			case *goast.FuncDecl:
				if v.Recv == nil {
					scopeDelete(objMap, scope, v.Name.Name)
				}
			}
		}
	}
	return objMap
}

func convErr(fset *token.FileSet, e error) (ret Error, ok bool) {
	switch v := e.(type) {
	case *gogen.CodeError:
		ret.Pos, ret.End, ret.Msg = v.Pos, v.End, v.Msg
	case *gogen.MatchError:
		if v.Src != nil {
			ret.Pos, ret.End = v.Src.Pos(), v.Src.End()
		}
		ret.Msg = v.Message("")
	case *gogen.ImportError:
		ret.Pos, ret.End, ret.Msg = v.Pos, v.End, v.Err.Error()
	case *gogen.BoundTypeError:
		ret.Pos, ret.End, ret.Msg = v.Pos, v.End, v.Error()
	default:
		return
	}
	ret.Fset, ok = fset, true
	return
}

func convGoErr(e error) (ret Error, ok bool) {
	if v, ok := e.(types.Error); ok {
		ret.Pos, ret.Msg, ret.Soft = v.Pos, v.Msg, v.Soft
		code, _, end, ok := typesutil.GetErrorGo116(&v)
		if ok {
			ret.Code = Code(code)
			ret.End = end
		}
	}
	return ret, true
}

func scopeDelete(objMap map[types.Object]types.Object, scope *types.Scope, name string) {
	if o := typesutil.ScopeDelete(scope, name); o != nil {
		objMap[o] = nil
	}
}
