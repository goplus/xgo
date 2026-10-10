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

	"github.com/goplus/mod/xgomod"
)

func writeGoMod(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(body), 0644); err != nil {
		t.Fatal("write go.mod:", err)
	}
}

// TestClassfileRequires checks that classfile requires (declared via a
// `//gop:class` or `//xgo:class` marker) are detected, while ordinary requires
// are ignored.
func TestClassfileRequires(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, `module hello

go 1.24

require (
	github.com/goplus/yap v0.8.1 //gop:class
	github.com/goplus/spx v1.0.0 //xgo:class
	github.com/qiniu/x v1.13.10 // indirect
)
`)
	modObj, err := xgomod.Load(dir)
	if err != nil {
		t.Fatal("xgomod.Load:", err)
	}
	reqs := classfileRequires(modObj)
	got := map[string]string{}
	for _, r := range reqs {
		got[r.Path] = r.Version
	}
	if v := got["github.com/goplus/yap"]; v != "v0.8.1" {
		t.Fatalf("yap version = %q, want v0.8.1", v)
	}
	if v := got["github.com/goplus/spx"]; v != "v1.0.0" {
		t.Fatalf("spx version = %q, want v1.0.0", v)
	}
	if _, ok := got["github.com/qiniu/x"]; ok {
		t.Fatal("github.com/qiniu/x should not be treated as a classfile require")
	}
	if len(reqs) != 2 {
		t.Fatalf("len(reqs) = %d, want 2", len(reqs))
	}
}

// TestClassfileRequiresNone ensures that a module with no classfile requires
// yields no entries.
func TestClassfileRequiresNone(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, `module plainhello

go 1.24

require github.com/qiniu/x v1.13.10 // indirect
`)
	modObj, err := xgomod.Load(dir)
	if err != nil {
		t.Fatal("xgomod.Load:", err)
	}
	if reqs := classfileRequires(modObj); len(reqs) != 0 {
		t.Fatalf("classfileRequires = %v, want none", reqs)
	}
}

// TestKeepClassfileRequiresReAdds reproduces the core of issue #2175: `go mod
// tidy` strips a classfile require that no Go source imports. keepClassfileRequires
// must re-add it (with the classfile marker) so the declaration survives.
func TestKeepClassfileRequiresReAdds(t *testing.T) {
	dir := t.TempDir()
	// Simulate the state right after `go mod tidy` dropped the classfile require.
	writeGoMod(t, dir, `module hello

go 1.24
`)
	classMods := []classfileRequire{{Path: "github.com/goplus/yap", Version: "v0.8.1"}}
	if err := keepClassfileRequires(dir, classMods); err != nil {
		t.Fatal("keepClassfileRequires:", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal("read go.mod:", err)
	}
	out := string(data)
	if !strings.Contains(out, "github.com/goplus/yap v0.8.1") {
		t.Fatalf("classfile require was not re-added:\n%s", out)
	}
	if !strings.Contains(out, "xgo:class") && !strings.Contains(out, "gop:class") {
		t.Fatalf("classfile marker missing:\n%s", out)
	}

	// Reloading must report it as a classfile require again.
	modObj, err := xgomod.Load(dir)
	if err != nil {
		t.Fatal("xgomod.Load:", err)
	}
	reqs := classfileRequires(modObj)
	if len(reqs) != 1 || reqs[0].Path != "github.com/goplus/yap" {
		t.Fatalf("reloaded classfile requires = %v, want github.com/goplus/yap", reqs)
	}
}

// TestKeepClassfileRequiresNoChange ensures that when the classfile require is
// still present, go.mod is left byte-for-byte unchanged.
func TestKeepClassfileRequiresNoChange(t *testing.T) {
	dir := t.TempDir()
	body := `module hello

go 1.24

require github.com/goplus/yap v0.8.1 //gop:class
`
	writeGoMod(t, dir, body)
	classMods := []classfileRequire{{Path: "github.com/goplus/yap", Version: "v0.8.1"}}
	if err := keepClassfileRequires(dir, classMods); err != nil {
		t.Fatal("keepClassfileRequires:", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal("read go.mod:", err)
	}
	if string(data) != body {
		t.Fatalf("go.mod changed unexpectedly:\n%s", string(data))
	}
}

// TestKeepClassfileRequiresEmpty is a no-op fast path when there are no
// classfile requires to preserve.
func TestKeepClassfileRequiresEmpty(t *testing.T) {
	if err := keepClassfileRequires(t.TempDir(), nil); err != nil {
		t.Fatal("keepClassfileRequires(nil):", err)
	}
}
