/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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
	"path/filepath"
	"strings"

	"github.com/runsys/xmod/gopmod"
	"github.com/runsys/xmod/modfetch"
	"github.com/goplus/xgo/parser"

	astmod "github.com/goplus/xgo/ast/mod"
)

// -----------------------------------------------------------------------------

func GenDepMods(mod *gopmod.Module, dir string, recursively bool) (ret map[string]struct{}, err error) {
	modBase := mod.Path()
	ret = make(map[string]struct{})
	for _, r := range mod.Opt.Import {
		ret[r.ClassfileMod] = struct{}{}
	}
	err = HandleDeps(mod, dir, recursively, func(pkgPath string) {
		modPath, _ := modfetch.Split(pkgPath, modBase)
		if modPath != "" && modPath != modBase {
			ret[modPath] = struct{}{}
		}
	})
	return
}

func HandleDeps(mod *gopmod.Module, dir string, recursively bool, h func(pkgPath string)) (err error) {
	g := depsGen{
		deps: astmod.Deps{HandlePkg: h},
		mod:  mod,
		fset: token.NewFileSet(),
	}
	if recursively {
		err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				if strings.HasPrefix(d.Name(), "_") { // skip _
					return filepath.SkipDir
				}
				err = g.gen(path)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
			}
			return err
		})
	} else {
		err = g.gen(dir)
	}
	return
}

type depsGen struct {
	deps astmod.Deps
	mod  *gopmod.Module
	fset *token.FileSet
}

func (p depsGen) gen(dir string) (err error) {
	pkgs, err := parser.ParseDirEx(p.fset, dir, parser.Config{
		ClassKind: p.mod.ClassKind,
		Mode:      parser.ImportsOnly,
	})
	if err != nil {
		return
	}

	for _, pkg := range pkgs {
		p.deps.Load(pkg, false)
	}
	return
}

// -----------------------------------------------------------------------------
