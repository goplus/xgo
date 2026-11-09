/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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
	"path/filepath"
	"strings"

	"github.com/goplus/mod/env"
	"github.com/goplus/mod/goxmod"
	"golang.org/x/mod/modfile"
)

// This file replaces the go.mod / gox.mod writing operations that used to live
// in github.com/goplus/mod/{modload,xgomod}, which are gone in the Workspace
// model (mod.Workspace is a read-only snapshot). The edits are done directly on
// golang.org/x/mod/modfile, which is the same writer those packages used.

// -----------------------------------------------------------------------------

const (
	defaultGoVer  = "1.18"
	defaultXgoVer = "1.5"
)

// GoMod is an editable go.mod for the module-management commands (xgo mod init,
// xgo get, xgo mod tidy) and for the go.mod auto-update that follows a
// successful compile. It is a thin editor over golang.org/x/mod/modfile;
// nothing here reads the module graph. The compiler directive (llgo/tinygo)
// and classfile markers are recorded in go.mod itself, as the go command and
// XGo toolchain read them, so no separate gox.mod is written.
type GoMod struct {
	file  *modfile.File // parsed go.mod
	gomod string        // path of go.mod
}

// CreateGoMod creates a new module in dir with the given module path, writing
// nothing to disk yet (call Save). goVer defaults when empty. xgoVer is accepted
// for signature compatibility with the previous modload.Create; the XGo version
// is not recorded in go.mod. It errors if a go.mod already exists in dir.
func CreateGoMod(dir, modPath, goVer, xgoVer string) (*GoMod, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	gomod := filepath.Join(abs, "go.mod")
	if _, err := os.Stat(gomod); err == nil {
		return nil, &os.PathError{Op: "create", Path: gomod, Err: os.ErrExist}
	}
	if goVer == "" {
		goVer = defaultGoVer
	}
	_ = xgoVer // accepted for API compatibility; see doc above
	f := new(modfile.File)
	if err := f.AddModuleStmt(modPath); err != nil {
		return nil, err
	}
	if err := f.AddGoStmt(goVer); err != nil {
		return nil, err
	}
	f.Syntax.Name = gomod
	return &GoMod{file: f, gomod: gomod}, nil
}

// LoadGoMod reads and parses the go.mod of the module rooted at dir.
func LoadGoMod(dir string) (*GoMod, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	gomod := filepath.Join(abs, "go.mod")
	data, err := os.ReadFile(gomod)
	if err != nil {
		return nil, err
	}
	f, err := modfile.Parse(gomod, data, nil)
	if err != nil {
		return nil, err
	}
	return &GoMod{file: f, gomod: gomod}, nil
}

// Path returns the module path.
func (p *GoMod) Path() string {
	if p.file.Module != nil {
		return p.file.Module.Mod.Path
	}
	return ""
}

// AddCompiler records a compiler directive (llgo/tinygo) as a comment on the
// go directive in go.mod, the way the XGo toolchain reads it.
func (p *GoMod) AddCompiler(compiler, ver string) {
	f := p.file
	if f.Go == nil {
		f.AddGoStmt(defaultGoVer)
	}
	if line := f.Go.Syntax; line != nil {
		line.Suffix = []modfile.Comment{{
			Token:  "// " + compiler + " " + ver,
			Suffix: true,
		}}
	}
}

// AddRequire adds a require line, marking it as a classfile require
// (//xgo:class) when hasProj is set.
func (p *GoMod) AddRequire(path, vers string, hasProj bool) error {
	f := p.file
	if err := f.AddRequire(path, vers); err != nil {
		return err
	}
	if hasProj {
		for _, r := range f.Require {
			if r.Mod.Path == path {
				if !isClassRequire(r) && r.Syntax != nil {
					r.Syntax.Suffix = append(r.Syntax.Suffix, modfile.Comment{
						Token:  "//xgo:class",
						Suffix: true,
					})
				}
				break
			}
		}
	}
	return nil
}

// Save writes go.mod to disk. The compiler directive (llgo/tinygo) lives in the
// go.mod `go` line suffix, so a plain module init writes no gox.mod — matching
// the old modload.Create/Save, which only wrote gox.mod when the module
// declared classfile projects.
func (p *GoMod) Save() error {
	p.file.Cleanup()
	data, err := p.file.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(p.gomod, data, 0644)
}

// -----------------------------------------------------------------------------

// saveXGoMod adds `require github.com/goplus/xgo` (and github.com/qiniu/x)
// to the go.mod at gomod when the compiled package depends on them, matching
// the old (*xgomod.Module).SaveWithXGoMod. flags come from gogen's
// CheckXGoDeps (FlagDepModXGo / FlagDepModX).
func saveXGoMod(gomod string, xgo *env.XGo, flags int) error {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return err
	}
	f, err := modfile.Parse(gomod, data, nil)
	if err != nil {
		return err
	}
	old := checkXGoDeps(f)
	if (flags &^ old) == 0 { // nothing to add
		return nil
	}
	changed := false
	if (flags&flagDepModXGo) != 0 && (old&flagDepModXGo) == 0 {
		if err := f.AddRequire(xgoMod, xgoVerOf(xgo)); err == nil {
			changed = true
		}
	}
	if (flags&flagDepModX) != 0 && (old&flagDepModX) == 0 {
		if ver := xModVerOf(xgo); ver != "" {
			if err := f.AddRequire(xMod, ver); err == nil {
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	f.Cleanup()
	out, err := f.Format()
	if err != nil {
		return err
	}
	return os.WriteFile(gomod, out, 0644)
}

const (
	flagDepModXGo = 1 << iota // depends on github.com/goplus/xgo
	flagDepModX               // depends on github.com/qiniu/x
)

// checkXGoDeps reports which of the XGo toolchain modules the go.mod already
// requires (so they are not re-added).
func checkXGoDeps(f *modfile.File) (flags int) {
	if f.Module != nil {
		switch f.Module.Mod.Path {
		case xgoMod:
			return flagDepModXGo | flagDepModX
		case xMod:
			return flagDepModX
		}
	}
	for _, r := range f.Require {
		switch r.Mod.Path {
		case xgoMod:
			flags |= flagDepModXGo
		case xMod:
			flags |= flagDepModX
		}
	}
	return
}

// xgoVerOf returns the XGo version to require, trimming a trailing " devel".
func xgoVerOf(xgo *env.XGo) string {
	ver := xgo.Version
	if pos := strings.IndexByte(ver, ' '); pos > 0 {
		ver = ver[:pos]
	}
	if ver == "" {
		return "v" + defaultXgoVer + ".0"
	}
	return ver
}

// xModVerOf returns the github.com/qiniu/x version required by the XGo
// installation's own go.mod, or "" if it cannot be determined.
func xModVerOf(xgo *env.XGo) string {
	data, err := os.ReadFile(filepath.Join(xgo.Root, "go.mod"))
	if err != nil {
		return ""
	}
	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return ""
	}
	for _, r := range f.Require {
		if r.Mod.Path == xMod {
			return r.Mod.Version
		}
	}
	return ""
}

// -----------------------------------------------------------------------------

// isClassRequire reports whether a require line carries a //xgo:class or
// //gop:class marker.
func isClassRequire(r *modfile.Require) bool {
	if r.Syntax == nil {
		return false
	}
	for _, c := range r.Syntax.Suffix {
		if len(c.Token) < 2 {
			continue
		}
		text := strings.TrimLeft(c.Token[2:], " \t")
		if strings.HasPrefix(text, "xgo:class") || strings.HasPrefix(text, "gop:class") {
			return true
		}
	}
	return false
}

// modPathOf returns the module path declared in the go.mod at the given path,
// or "" if it cannot be read.
func modPathOf(gomod string) string {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return ""
	}
	f, err := modfile.ParseLax(gomod, data, nil)
	if err != nil || f.Module == nil {
		return ""
	}
	return f.Module.Mod.Path
}

// HasProject reports whether the module rooted at dir declares any classfile
// project (in its gox.mod, legacy gop.mod, or a //xgo:class require). It is
// used by `xgo get` to decide whether a fetched dependency is a classfile
// module.
func HasProject(dir string) bool {
	gomodData, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	gomod, err := modfile.Parse("go.mod", gomodData, nil)
	if err != nil {
		return false
	}
	for _, r := range gomod.Require {
		if isClassRequire(r) {
			return true
		}
	}
	gox, err := goxmod.LoadModfile(gomod, filepath.Join(dir, "gox.mod"))
	if err != nil {
		return false
	}
	return len(gox.Projects) > 0
}

// -----------------------------------------------------------------------------
