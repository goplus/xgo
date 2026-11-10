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
	"errors"
	"path/filepath"

	"github.com/goplus/mod"
	"github.com/goplus/mod/goxmod"
)

// -----------------------------------------------------------------------------

// Workspace is a snapshot of the module layout that owns the sources being
// compiled: a go.work with its main modules, a single go.mod, or neither (the
// default workspace, which resolves the standard library and the built-in
// classfiles only). It is the unit that replaces the per-module *xgomod.Module
// the tool package used before: a single module is just a workspace of one.
//
// It bundles the read-only mod.Workspace with the directory it was discovered
// from and the go.work file that governs it (if any), two pieces of information
// mod.Workspace does not itself expose but that the tool package needs for the
// import root, the cache file name, and the `go mod tidy` decision.
type Workspace struct {
	*mod.Workspace

	// dir is the base directory the workspace resolves imports from: the module
	// directory for a go.mod-only workspace, or the go.work directory for a
	// go.work workspace. It is what packages.NewImporter uses as its work dir.
	dir string

	// gowork is the path of the go.work file in effect, or "" when there is
	// none (a go.mod-only or default workspace). A non-empty value means the
	// module being generated may depend on a sibling with no require line, so
	// `go mod tidy` must not run for it.
	gowork string
}

// LoadWorkspace finds the workspace that owns dir the way the go command does
// (the nearest go.work, else the nearest go.mod, else a default workspace) and
// loads it. It never fails with "no module": a loose directory yields a default
// workspace. A broken go.work or go.mod is still reported as an error.
func LoadWorkspace(dir string) (*Workspace, error) {
	gowork, gomod, derr := mod.Discover(dir)
	if derr != nil && !errors.Is(derr, mod.ErrNoModule) {
		return nil, derr
	}
	w, err := mod.LoadFromDir(dir)
	if err != nil {
		return nil, err
	}
	return newWorkspace(w, dir, gowork, gomod), nil
}

// newWorkspace wraps a loaded mod.Workspace with the base directory it resolves
// from and the go.work file (if any) that governs it. gomod is the go.mod found
// by Discover for a go.mod-only workspace (empty for go.work or no module).
func newWorkspace(w *mod.Workspace, dir, gowork, gomod string) *Workspace {
	var base string
	switch {
	case gowork != "":
		base = filepath.Dir(gowork)
	case gomod != "":
		base = filepath.Dir(gomod)
	default:
		base = absOr(dir)
	}
	return &Workspace{Workspace: w, dir: base, gowork: gowork}
}

// absOr returns dir made absolute, or dir unchanged if that fails.
func absOr(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// Dir returns the base directory the workspace resolves imports from.
func (w *Workspace) Dir() string {
	return w.dir
}

// HasGoWork reports whether a go.work governs the workspace.
func (w *Workspace) HasGoWork() bool {
	return w.gowork != ""
}

// -----------------------------------------------------------------------------

// classesOf returns the classfiles that apply to sources in dir: the classfile
// set of the main module that owns dir. A loose directory, or a nested module
// outside the workspace, gets regular sources plus the built-in classfiles.
func classesOf(w *Workspace, dir string) (*goxmod.Classes, error) {
	if w == nil {
		return goxmod.DefaultClasses(), nil
	}
	cls, err := w.Classes(dir)
	if errors.Is(err, mod.ErrDirNotInWorkspace) {
		return goxmod.DefaultClasses(), nil
	}
	return cls, err
}

// ClassesOf returns the classfiles that apply to sources in dir, discovering
// the workspace that owns dir first. A loose directory gets the built-in
// classfiles. It is a convenience for callers that only need the classfile set
// of a directory (such as the formatter).
func ClassesOf(dir string) (*goxmod.Classes, error) {
	w, err := LoadWorkspace(dir)
	if err != nil {
		return nil, err
	}
	return classesOf(w, dir)
}

// -----------------------------------------------------------------------------
