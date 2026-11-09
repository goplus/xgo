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
	"path/filepath"

	"github.com/goplus/mod/env"
	"github.com/qiniu/x/errors"
	"golang.org/x/mod/modfile"
)

func Tidy(dir string, xgo *env.XGo) (err error) {
	w, err := LoadWorkspace(dir)
	if err != nil {
		return errors.NewWith(err, `LoadWorkspace(dir)`, -2, "tool.LoadWorkspace", dir)
	}
	if w.IsDefault() {
		return errors.NewWith(ErrNotFound, `no go.mod found`, -2, "tool.Tidy", dir)
	}

	modRoot := w.Dir()
	gomod := filepath.Join(modRoot, "go.mod")

	// Remember the classfile dependencies (declared via `//gop:class` or
	// `//xgo:class` require directives) before running `go mod tidy`. The native
	// `go mod tidy` only sees Go imports, so a classfile module that is referenced
	// implicitly (via a classfile import, or not yet referenced by any source
	// file) looks unused to it and gets dropped from go.mod. We re-add them
	// afterwards so that `xgo mod tidy` behaves like running `xgo get` for
	// classfile deps.
	classMods, err := classfileRequires(gomod)
	if err != nil {
		return errors.NewWith(err, `classfileRequires(gomod)`, -2, "tool.classfileRequires", gomod)
	}

	conf := &Config{XGo: xgo, Workspace: w}
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

// classfileRequires returns the classfile module requires declared in the
// go.mod at gomod, keeping the version recorded for each one.
func classfileRequires(gomod string) ([]classfileRequire, error) {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return nil, err
	}
	f, err := modfile.Parse(gomod, data, nil)
	if err != nil {
		return nil, err
	}
	var reqs []classfileRequire
	for _, r := range f.Require {
		if isClassRequire(r) {
			reqs = append(reqs, classfileRequire{Path: r.Mod.Path, Version: r.Mod.Version})
		}
	}
	return reqs, nil
}

// keepClassfileRequires re-adds classfile module requires that `go mod tidy`
// may have dropped, re-applying the `//xgo:class` marker (as `xgo get` does) so
// that the classfile dependency declaration survives. A dropped require is
// restored at the version recorded before `go mod tidy` ran, not re-resolved
// against the module graph. It does nothing when every classfile require is
// still present.
func keepClassfileRequires(modRoot string, classMods []classfileRequire) (err error) {
	if len(classMods) == 0 {
		return nil
	}
	modObj, err := LoadGoMod(modRoot)
	if err != nil {
		return errors.NewWith(err, `LoadGoMod(modRoot)`, -2, "tool.LoadGoMod", modRoot)
	}
	have := make(map[string]bool, len(modObj.file.Require))
	for _, r := range modObj.file.Require {
		have[r.Mod.Path] = true
	}
	changed := false
	for _, c := range classMods {
		if have[c.Path] {
			continue
		}
		if e := modObj.AddRequire(c.Path, c.Version, true); e != nil {
			return errors.NewWith(e, `modObj.AddRequire(path, vers, true)`, -2, "(*tool.GoMod).AddRequire", c.Path, c.Version, true)
		}
		changed = true
	}
	if !changed {
		return nil
	}
	if err = modObj.Save(); err != nil {
		return errors.NewWith(err, `modObj.Save()`, -2, "(*tool.GoMod).Save")
	}
	return nil
}
