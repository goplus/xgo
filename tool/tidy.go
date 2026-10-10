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

package tool

import (
	"os"
	"os/exec"

	"github.com/goplus/mod/env"
	"github.com/goplus/mod/xgomod"
	"github.com/qiniu/x/errors"
)

func Tidy(dir string, xgo *env.XGo) (err error) {
	modObj, err := xgomod.Load(dir)
	if err != nil {
		return errors.NewWith(err, `xgomod.Load(dir, mod.GopModOnly)`, -2, "xgomod.Load", dir)
	}

	modRoot := modObj.Root()

	// Remember the classfile dependencies (declared via `//gop:class` require
	// directives) before running `go mod tidy`. The native `go mod tidy` only
	// sees Go imports, so a classfile module that is referenced implicitly (via
	// a classfile import, or not yet referenced by any source file) looks unused
	// to it and gets dropped from go.mod. We re-add them afterwards so that
	// `xgo mod tidy` behaves like running `xgo get` for classfile deps.
	classMods := classfileRequires(modObj)

	/*
		depMods, err := GenDepMods(modObj, modRoot, true)
		if err != nil {
			return errors.NewWith(err, `GenDepMods(modObj, modRoot, true)`, -2, "tool.GenDepMods", modObj, modRoot, true)
		}

		old := modObj.DepMods()
		for modPath := range old {
			if _, ok := depMods[modPath]; !ok { // removed
				modObj.DropRequire(modPath)
			}
		}
		for modPath := range depMods {
			if _, ok := old[modPath]; !ok { // added
				if newMod, e := modfetch.Get(modPath); e != nil {
					return errors.NewWith(e, `modfetch.Get(modPath)`, -1, "modfetch.Get", modPath)
				} else {
					modObj.AddRequire(newMod.Path, newMod.Version)
				}
			}
		}

		modObj.Cleanup()
		err = modObj.Save()
		if err != nil {
			return errors.NewWith(err, `modObj.Save()`, -2, "(*xgomod.Module).Save")
		}
	*/
	conf := &Config{XGo: xgo}
	err = genGoDir(modRoot, conf, true, true, 0)
	if err != nil {
		return errors.NewWith(err, `genGoDir(modRoot, conf, true, true)`, -2, "tool.genGoDir", modRoot, conf, true, true)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = modRoot
	err = cmd.Run()
	if err != nil {
		return errors.NewWith(err, `cmd.Run()`, -2, "(*exec.Cmd).Run")
	}

	return keepClassfileRequires(modRoot, classMods)
}

// classfileRequire is a classfile module dependency declared in go.mod via a
// `//gop:class` (or `//xgo:class`) require directive.
type classfileRequire struct {
	Path    string
	Version string
}

// classfileRequires returns the classfile module requires of a module, keeping
// the version recorded in go.mod for each one.
func classfileRequires(modObj *xgomod.Module) []classfileRequire {
	classMods := modObj.Opt.ClassMods
	if len(classMods) == 0 {
		return nil
	}
	isClass := make(map[string]bool, len(classMods))
	for _, modPath := range classMods {
		isClass[modPath] = true
	}
	reqs := make([]classfileRequire, 0, len(classMods))
	for _, r := range modObj.File.Require {
		if isClass[r.Mod.Path] {
			reqs = append(reqs, classfileRequire{Path: r.Mod.Path, Version: r.Mod.Version})
		}
	}
	return reqs
}

// keepClassfileRequires re-adds classfile module requires that `go mod tidy`
// may have dropped, preserving the `//gop:class` marker so that the classfile
// dependency declaration survives. It does nothing when every classfile require
// is still present.
func keepClassfileRequires(modRoot string, classMods []classfileRequire) (err error) {
	if len(classMods) == 0 {
		return nil
	}
	modObj, err := xgomod.Load(modRoot)
	if err != nil {
		return errors.NewWith(err, `xgomod.Load(modRoot)`, -2, "xgomod.Load", modRoot)
	}
	have := make(map[string]bool)
	for _, r := range modObj.File.Require {
		have[r.Mod.Path] = true
	}
	changed := false
	for _, c := range classMods {
		if have[c.Path] {
			continue
		}
		if e := modObj.AddRequire(c.Path, c.Version, true); e != nil {
			return errors.NewWith(e, `modObj.AddRequire(path, vers, true)`, -2, "(*xgomod.Module).AddRequire", c.Path, c.Version, true)
		}
		changed = true
	}
	if !changed {
		return nil
	}
	if err = modObj.Save(); err != nil {
		return errors.NewWith(err, `modObj.Save()`, -2, "(*xgomod.Module).Save")
	}
	return nil
}
