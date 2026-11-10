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
	"path/filepath"
	"testing"
)

// TestLoadWorkspaceDefault: a directory with no go.work/go.mod yields a default
// workspace that answers the built-in classfiles only.
func TestLoadWorkspaceDefault(t *testing.T) {
	dir := t.TempDir()
	w, err := LoadWorkspace(dir)
	if err != nil {
		t.Fatal("LoadWorkspace:", err)
	}
	if !w.IsDefault() {
		t.Fatal("IsDefault = false, want true for a loose directory")
	}
	if w.HasGoWork() {
		t.Fatal("HasGoWork = true, want false")
	}
	if w.Dir() == "" {
		t.Fatal("Dir() is empty")
	}
	cls, err := classesOf(w, dir)
	if err != nil {
		t.Fatal("classesOf:", err)
	}
	if !cls.IsClass(".gsh") {
		t.Fatal(".gsh should be a built-in classfile")
	}
	if cls.IsClass(".spx") {
		t.Fatal(".spx should not be a classfile in a default workspace")
	}
}

// TestLoadWorkspaceGoMod: a go.mod-only project resolves its own module
// directory as the workspace base.
func TestLoadWorkspaceGoMod(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example.com/app\n\ngo 1.24\n")
	w, err := LoadWorkspace(dir)
	if err != nil {
		t.Fatal("LoadWorkspace:", err)
	}
	if w.IsDefault() {
		t.Fatal("IsDefault = true, want false for a go.mod project")
	}
	if w.HasGoWork() {
		t.Fatal("HasGoWork = true, want false for a go.mod-only project")
	}
	absDir, _ := filepath.Abs(dir)
	if got, want := filepath.Clean(w.Dir()), filepath.Clean(absDir); !samePathLike(got, want) {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
}

// TestLoadWorkspaceGoModWithReplace guards the regression where a go.mod-only
// project's workspace base was resolved to a locally-replaced dependency's
// directory (which sorts before the main module by path) instead of the main
// module's own directory.
func TestLoadWorkspaceGoModWithReplace(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, `module example.com/app

go 1.24

require zzz.org/bar v1.0.0

replace zzz.org/bar => ../bar
`)
	w, err := LoadWorkspace(dir)
	if err != nil {
		t.Fatal("LoadWorkspace:", err)
	}
	absDir, _ := filepath.Abs(dir)
	if got, want := filepath.Clean(w.Dir()), filepath.Clean(absDir); !samePathLike(got, want) {
		t.Fatalf("Dir() = %q, want the main module dir %q (not the replaced dependency's)", got, want)
	}
}

// TestClassesOf checks the ClassesOf convenience wrapper and the
// not-in-workspace fallback to built-ins.
func TestClassesOf(t *testing.T) {
	dir := t.TempDir()
	cls, err := ClassesOf(dir)
	if err != nil {
		t.Fatal("ClassesOf:", err)
	}
	if !cls.IsClass(".gsh") {
		t.Fatal("built-in .gsh expected")
	}
	// classesOf(nil, dir) falls back to the built-in classfiles.
	if c, err := classesOf(nil, dir); err != nil || !c.IsClass(".gsh") {
		t.Fatalf("classesOf(nil) = %v, %v", c, err)
	}
}

// TestCanCl checks the source-file predicate with and without a classfile set.
func TestCanCl(t *testing.T) {
	// Fixed extensions are always sources.
	for _, name := range []string{"a.go", "a.xgo", "a.gop", "a.gox"} {
		if !canCl(nil, name) {
			t.Fatalf("canCl(nil, %q) = false, want true", name)
		}
	}
	// A classfile extension depends on the class set.
	if canCl(nil, "Sprite.spx") {
		t.Fatal("canCl(nil, Sprite.spx) = true, want false")
	}
}

// TestIsDirArg distinguishes directory arguments from import paths.
func TestIsDirArg(t *testing.T) {
	dirs := []string{".", "..", "./x", "../x"}
	for _, d := range dirs {
		if !isDirArg(d) {
			t.Fatalf("isDirArg(%q) = false, want true", d)
		}
	}
	imports := []string{"fmt", "example.com/app", "github.com/goplus/xgo/tool"}
	for _, p := range imports {
		if isDirArg(p) {
			t.Fatalf("isDirArg(%q) = true, want false", p)
		}
	}
	if !isDirArg(mustAbs(t, ".")) {
		t.Fatal("isDirArg(absolute) = false, want true")
	}
}

// TestAbsJoin resolves directory arguments against workDir.
func TestAbsJoin(t *testing.T) {
	work := mustAbs(t, ".")
	if got := absJoin(work, "./x"); !filepath.IsAbs(got) {
		t.Fatalf("absJoin not absolute: %q", got)
	}
	abs := mustAbs(t, ".")
	if got := absJoin(work, abs); filepath.Clean(got) != filepath.Clean(abs) {
		t.Fatalf("absJoin(absolute) = %q, want %q", got, abs)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal("filepath.Abs:", err)
	}
	return abs
}

// samePathLike compares two paths loosely (cleaned), tolerating symlinked temp
// roots that differ only by a /private or /tmp prefix on some platforms.
func samePathLike(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	return filepath.Base(a) == filepath.Base(b)
}
