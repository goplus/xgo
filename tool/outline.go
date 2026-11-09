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

package tool

import (
	"fmt"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goplus/mod"
	"github.com/goplus/xgo/cl/outline"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/x/xgoenv"
	"github.com/qiniu/x/errors"
)

// -----------------------------------------------------------------------------

func Outline(dir string, conf *Config) (out outline.Package, err error) {
	if dir, err = filepath.Abs(dir); err != nil {
		return
	}

	if conf == nil {
		conf = new(Config)
	}

	w := conf.Workspace
	if w == nil {
		if w, err = LoadWorkspace(dir); err != nil {
			err = errors.NewWith(err, `LoadWorkspace(dir)`, -2, "tool.LoadWorkspace", dir)
			return
		}
	}
	cls, err := classesOf(w, dir)
	if err != nil {
		err = errors.NewWith(err, `classesOf(w, dir)`, -2, "tool.classesOf", w, dir)
		return
	}

	filterConf := conf.Filter
	filter := func(fi fs.FileInfo) bool {
		if filterConf != nil && !filterConf(fi) {
			return false
		}
		fname := fi.Name()
		if pos := strings.Index(fname, "."); pos > 0 {
			fname = fname[:pos]
		}
		return !strings.HasSuffix(fname, "_test")
	}
	fset := conf.Fset
	if fset == nil {
		fset = token.NewFileSet()
	}
	pkgs, err := parser.ParseDirEx(fset, dir, parser.Config{
		ClassInfo: cls.ClassInfo,
		Filter:    filter,
		Mode:      parser.ParseComments,
	})
	if err != nil {
		return
	}
	if len(pkgs) == 0 {
		err = ErrNotFound
		return
	}

	imp := conf.Importer
	if imp == nil {
		xgo := conf.XGo
		if xgo == nil {
			xgo = xgoenv.Get()
		}
		imp = NewImporter(w, xgo, fset)
	}

	modRoot, modPath := ownerModule(w, dir)
	for name, pkg := range pkgs {
		if out.Valid() {
			err = fmt.Errorf("%w: %s, %s", ErrMultiPackges, name, out.Pkg().Name())
			return
		}
		if len(pkg.Files)+len(pkg.GoFiles) == 0 { // no Go/XGo source files
			break
		}
		pkgPath := modPath
		if modRoot != "" {
			if relPart, e := filepath.Rel(modRoot, dir); e == nil {
				pkgPath = path.Join(modPath, filepath.ToSlash(relPart))
			}
		}
		out, err = outline.NewPackage(pkgPath, pkg, &outline.Config{
			Fset:        fset,
			Importer:    imp,
			LookupClass: cls.LookupClass,
		})
		if err != nil {
			return
		}
	}
	if !out.Valid() {
		err = ErrNotFound
	}
	return
}

// ownerModule returns the module root directory and module path of the
// workspace module that owns dir, or ("", "") for a loose directory.
func ownerModule(w *Workspace, dir string) (modRoot, modPath string) {
	gomod := ownerGoMod(w, dir)
	if gomod == "" {
		return "", ""
	}
	return filepath.Dir(gomod), modPathOf(gomod)
}

// -----------------------------------------------------------------------------

func OutlinePkgPath(workDir, pkgPath string, conf *Config, allowExtern bool) (out outline.Package, err error) {
	if conf == nil {
		conf = new(Config)
	}
	w := conf.Workspace
	if w == nil {
		if w, err = LoadWorkspace(workDir); err != nil {
			err = errors.NewWith(err, `LoadWorkspace(workDir)`, -2, "tool.LoadWorkspace", workDir)
			return
		}
	}

	if w.IsDefault() && allowExtern {
		remotePkgPathDo(pkgPath, func(pkgDir, modDir string) {
			modFile := chmodModfile(modDir)
			defer os.Chmod(modFile, modReadonly)
			out, err = Outline(pkgDir, conf)
		}, func(e error) {
			err = e
		})
		return
	}

	pkg, err := w.Lookup(pkgPath)
	if err != nil {
		return
	}
	if pkg.Type == mod.PkgtExtern {
		modFile := chmodModfile(pkg.ModDir)
		defer os.Chmod(modFile, modReadonly)
	}
	return Outline(pkg.Dir, conf)
}

func chmodModfile(modDir string) string {
	modFile := modDir + "/go.mod"
	os.Chmod(modFile, modWritable)
	return modFile
}

// -----------------------------------------------------------------------------
