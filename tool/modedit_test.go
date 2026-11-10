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
	"testing"

	"github.com/goplus/mod/env"
)

// TestCreateGoMod exercises the module-init path (go.mod + gox.mod writer) that
// replaces the removed modload.Create / AddCompiler / AddRequire / Save.
func TestCreateGoMod(t *testing.T) {
	dir := t.TempDir()
	m, err := CreateGoMod(dir, "example.com/app", "1.24", "1.5")
	if err != nil {
		t.Fatal("CreateGoMod:", err)
	}
	if m.Path() != "example.com/app" {
		t.Fatalf("Path = %q, want example.com/app", m.Path())
	}
	m.AddCompiler("llgo", "1.0")
	if err := m.AddRequire("github.com/goplus/lib", "v0.2.0", false); err != nil {
		t.Fatal("AddRequire:", err)
	}
	if err := m.Save(); err != nil {
		t.Fatal("Save:", err)
	}
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal("read go.mod:", err)
	}
	out := string(gomod)
	if !strings.Contains(out, "module example.com/app") {
		t.Fatalf("module line missing:\n%s", out)
	}
	if !strings.Contains(out, "// llgo 1.0") {
		t.Fatalf("compiler directive missing:\n%s", out)
	}
	if !strings.Contains(out, "github.com/goplus/lib v0.2.0") {
		t.Fatalf("require missing:\n%s", out)
	}
	// The compiler lives in go.mod's go directive, so a plain init writes no
	// gox.mod (matching the old modload.Create/Save).
	if _, err := os.Stat(filepath.Join(dir, "gox.mod")); err == nil {
		t.Fatal("gox.mod should not be written for a module with no projects")
	}
}

// TestCreateGoModExists ensures CreateGoMod refuses to overwrite an existing
// go.mod.
func TestCreateGoModExists(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module there\n\ngo 1.24\n")
	if _, err := CreateGoMod(dir, "there", "", ""); err == nil {
		t.Fatal("CreateGoMod should fail when go.mod already exists")
	}
}

// TestModPathOf reads the module path from a go.mod.
func TestModPathOf(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example.com/x\n\ngo 1.24\n")
	if got := modPathOf(filepath.Join(dir, "go.mod")); got != "example.com/x" {
		t.Fatalf("modPathOf = %q, want example.com/x", got)
	}
	if got := modPathOf(filepath.Join(dir, "missing.mod")); got != "" {
		t.Fatalf("modPathOf(missing) = %q, want empty", got)
	}
}

// TestHasProject reports whether a module declares a classfile project.
func TestHasProject(t *testing.T) {
	// A plain module has no project.
	plain := t.TempDir()
	writeGoMod(t, plain, "module plain\n\ngo 1.24\n")
	if HasProject(plain) {
		t.Fatal("HasProject(plain) = true, want false")
	}
	// A module with a //xgo:class require is a classfile module.
	cls := t.TempDir()
	writeGoMod(t, cls, "module cls\n\ngo 1.24\n\nrequire github.com/goplus/spx v1.0.0 //xgo:class\n")
	if !HasProject(cls) {
		t.Fatal("HasProject(cls) = false, want true")
	}
	// A directory with no module is not a project.
	if HasProject(t.TempDir()) {
		t.Fatal("HasProject(empty) = true, want false")
	}
}

// TestCheckXGoDeps classifies which XGo toolchain modules a go.mod requires.
func TestCheckXGoDeps(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, `module example.com/app

go 1.24

require (
	github.com/goplus/xgo v1.5.0
	github.com/qiniu/x v1.19.3
)
`)
	m, err := LoadGoMod(dir)
	if err != nil {
		t.Fatal("LoadGoMod:", err)
	}
	if flags := checkXGoDeps(m.file); flags != (flagDepModXGo | flagDepModX) {
		t.Fatalf("checkXGoDeps = %d, want both", flags)
	}
}

// TestSaveXGoMod adds the xgo/qiniu-x requires only when missing, and leaves the
// go.mod untouched when they are already there.
func TestSaveXGoMod(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example.com/app\n\ngo 1.24\n")
	gomod := filepath.Join(dir, "go.mod")
	xgo := &env.XGo{Version: "v1.5.0", Root: dir}

	// Depends on github.com/goplus/xgo: it should be added.
	if err := saveXGoMod(gomod, xgo, flagDepModXGo); err != nil {
		t.Fatal("saveXGoMod:", err)
	}
	data, _ := os.ReadFile(gomod)
	if !strings.Contains(string(data), "github.com/goplus/xgo v1.5.0") {
		t.Fatalf("xgo require not added:\n%s", string(data))
	}

	// A second call with the same flag is a no-op (already present).
	before, _ := os.ReadFile(gomod)
	if err := saveXGoMod(gomod, xgo, flagDepModXGo); err != nil {
		t.Fatal("saveXGoMod (2):", err)
	}
	after, _ := os.ReadFile(gomod)
	if string(before) != string(after) {
		t.Fatalf("go.mod changed on no-op call:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestXgoVerOf trims a trailing devel suffix and defaults when empty.
func TestXgoVerOf(t *testing.T) {
	if got := xgoVerOf(&env.XGo{Version: "v1.2.0 devel"}); got != "v1.2.0" {
		t.Fatalf("xgoVerOf(devel) = %q, want v1.2.0", got)
	}
	if got := xgoVerOf(&env.XGo{Version: "v1.3.0"}); got != "v1.3.0" {
		t.Fatalf("xgoVerOf = %q, want v1.3.0", got)
	}
	if got := xgoVerOf(&env.XGo{}); !strings.HasPrefix(got, "v") {
		t.Fatalf("xgoVerOf(empty) = %q, want a default version", got)
	}
}
