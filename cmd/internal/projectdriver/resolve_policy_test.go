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

package projectdriver

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/x/xgoprojs"
)

func TestLoadDriverModulePropagatesOptionalMetadataReadErrors(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	goxMod := filepath.Join(dir, "gox.mod")
	mustWriteFile(t, goMod, "module example.test/app\n\ngo 1.25\n")
	mustMkdirAll(t, goxMod)
	// A valid legacy fallback must not mask a non-NotExist error from gox.mod.
	mustWriteFile(t, filepath.Join(dir, "gop.mod"), "gop 1.8\nproject main.foo Game example.test/app\n")
	if _, err := loadDriverModule(goMod, goxMod); err == nil || !strings.Contains(err.Error(), "gox.mod") {
		t.Fatalf("loadDriverModule() = %v, want explicit gox.mod read error", err)
	}
}

func TestLoadDriverModuleFallsBackToGopModWhenGoxModIsAbsent(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	gopMod := filepath.Join(dir, "gop.mod")
	mustWriteFile(t, goMod, "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, gopMod, "gop 1.1\nproject main.foo Game example.test/app\n")
	loaded, err := loadDriverModule(goMod, filepath.Join(dir, "gox.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.HasProject() || loaded.GoxModIdentity().Path != gopMod {
		t.Fatalf("gop.mod fallback identity = %#v", loaded.GoxModIdentity())
	}
}

func TestResolvePropagatesOptionalMetadataReadErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustMkdirAll(t, filepath.Join(dir, "gox.mod"))
	mustWriteFile(t, filepath.Join(dir, "main.foo"), "// project source\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: dir})
	if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "gox.mod") {
		t.Fatalf("Resolve() = %v, want explicit gox.mod read error", err)
	}
}

func TestResolvePackageTargetRejectsUnmarkedDependencyDriver(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	dependencyProject := filepath.Join(fixture.framework, "example")
	mustMkdirAll(t, filepath.Join(dependencyProject, "pack"))
	mustWriteFile(t, filepath.Join(dependencyProject, "main.foo"), "// dependency project\n")
	mustWriteFile(t, filepath.Join(dependencyProject, "pack", "index.data"), "{}\n")

	goModPath := filepath.Join(fixture.app, "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = bytes.Replace(goMod, []byte(" //xgo:class"), nil, 1)
	if err := os.WriteFile(goModPath, goMod, 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := fixture.resolver(t)
	_, err = resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/framework/example"})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("unmarked dependency driver = %v, want ErrNotHandled", err)
	}
}

func TestResolveOrdinaryGoSubpackageInsideDriverModuleUsesLegacyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	ordinary := filepath.Join(fixture.framework, "ordinary")
	mustMkdirAll(t, ordinary)
	mustWriteFile(t, filepath.Join(ordinary, "main.go"), "package ordinary\n")
	resolver := fixture.resolver(t)
	_, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/framework/ordinary"})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("ordinary Go subpackage = %v", err)
	}
}

func TestUnsupportedPatternOnlyRejectsMatchedDriverTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	legacy := filepath.Join(fixture.app, "legacy")
	mustMkdirAll(t, legacy)
	mustWriteFile(t, filepath.Join(legacy, "main.go"), "package main\nfunc main() {}\n")
	nested := filepath.Join(legacy, "nested")
	mustMkdirAll(t, nested)
	mustWriteFile(t, filepath.Join(nested, "go.mod"), "module example.test/nested\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(nested, "main.foo"), "// nested driver-backed project must be outside the pattern\n")
	resolver := fixture.resolver(t)
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: filepath.Join(legacy, "...")}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("legacy pattern = %v, want ErrNotHandled", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: filepath.Join(fixture.project, "...")}); err == nil || !strings.Contains(err.Error(), "directory pattern") {
		t.Fatalf("driver pattern error = %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: filepath.Join(fixture.app, "...")}); err == nil || !strings.Contains(err.Error(), "directory pattern") {
		t.Fatalf("parent pattern containing driver-backed project error = %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/legacy/..."}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("legacy package pattern = %v, want ErrNotHandled", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/..."}); err == nil || !strings.Contains(err.Error(), "package pattern") {
		t.Fatalf("package pattern containing driver-backed project error = %v", err)
	}
}

func TestResolveDriverWithoutPack(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	goxmod := filepath.Join(fixture.framework, "gox.mod")
	data, err := os.ReadFile(goxmod)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte("pack pack index.data\n"), nil)
	if err := os.WriteFile(goxmod, data, 0644); err != nil {
		t.Fatal(err)
	}
	drv := resolveDriver(t, fixture.resolver(t), &xgoprojs.DirProj{Dir: fixture.project})
	if drv.PackDir != "" || drv.PackIndex != "" {
		t.Fatalf("pack = %q, %q", drv.PackDir, drv.PackIndex)
	}
	args, err := driverArgs(drv, actionRun, BuildPolicy{}, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "--pack-") {
			t.Fatalf("optional pack leaked into argv: %#v", args)
		}
	}
}

func TestDeclaringMetadataRejectsChangeAfterDiscovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gox.mod")
	original := []byte("xgo 1.8\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := xgomod.FileIdentity{Path: path, SHA256: sha256Bytes(original)}
	origin := ResolvedModule{Selected: ModuleRef{Path: "example.test/driver", Dir: dir, GoMod: filepath.Join(dir, "go.mod")}, Main: true}
	if _, err := declaringMetadata(origin, snapshot); err != nil {
		t.Fatalf("unchanged declaration rejected: %v", err)
	}
	if err := os.WriteFile(path, []byte("xgo 1.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := declaringMetadata(origin, snapshot); err == nil || !strings.Contains(err.Error(), "changed after discovery") {
		t.Fatalf("changed declaration error = %v", err)
	}
}

func TestResolveDriverRejectsAmbiguousAndMultiFile(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t)
	other := filepath.Join(fixture.project, "other.foo")
	mustWriteFile(t, other, "// another project\n")
	_, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err == nil || !strings.Contains(err.Error(), "2 project files") {
		t.Fatalf("ambiguity error = %v", err)
	}
	if err := os.Remove(other); err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.FilesProj{Files: []string{fixture.mainFile, filepath.Join(fixture.project, "worker.bar")}})
	if err == nil || !strings.Contains(err.Error(), "multiple source-file") {
		t.Fatalf("multi-file error = %v", err)
	}
}

func TestResolveNonDriverNotHandled(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/plain\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {}\n")
	resolver, err := NewResolver(context.Background(), dir, []string{"-tags=legacy-only"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: dir})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("plain project = %v", err)
	}
}

func TestResolverRejectsAmbientPolicyFailureWithoutFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/plain\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {}\n")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "'unterminated")
	if resolver, err := NewResolver(context.Background(), dir, nil); err == nil || resolver != nil {
		t.Fatalf("NewResolver() = %#v, %v; want policy error without fallback graph", resolver, err)
	}
}

func TestResolverDefersMissingGraphFileForLegacy(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/plain\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {}\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), dir, []string{"-modfile=missing.mod"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: dir}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("legacy target = %v", err)
	}
}

func TestCheckRequiredXGo(t *testing.T) {
	tests := []struct {
		required string
		current  string
		ok       bool
	}{
		{"1.8", "v1.8.0", true},
		{"v1.8.1", "v1.9.0", true},
		{"1.8", "v1.7.5", false},
		{"1.9", "v1.8.9", false},
		{"1.8", "(devel)", true},
		{"1.8.1", "(devel)", false},
		{"1.8", "v1.7.5 devel", true},
		{"1.8.1", "v1.7.5 devel", false},
		{"1.8", "xgo v1.9.0-devel", true},
		{"", "(devel)", true},
	}
	for _, test := range tests {
		err := checkRequiredXGo(test.required, test.current)
		if (err == nil) != test.ok {
			t.Fatalf("checkRequiredXGo(%q, %q) = %v", test.required, test.current, err)
		}
	}
}

func TestCheckRequiredXGoReportsDevelopmentCapability(t *testing.T) {
	err := checkRequiredXGo("1.8.1", "(devel)")
	if err == nil {
		t.Fatal("development build unexpectedly satisfied a newer capability")
	}
	for _, want := range []string{"declaring module requires XGo 1.8.1", "(devel)", "driver capability 1.8.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("checkRequiredXGo() error = %q, want %q", err, want)
		}
	}
}

func TestResolveUsesDeclaringModuleXGoRequirement(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t)
	resolver.xgoVersion = "v1.8.0"
	resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})

	setFixtureRequiredXGo(t, fixture, "1.9.0")
	resolver = fixture.resolver(t)
	resolver.xgoVersion = "v1.8.9"
	_, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err == nil || !strings.Contains(err.Error(), "declaring module requires XGo 1.9.0") {
		t.Fatalf("Resolve() = %v, want declaring-module version error", err)
	}

	resolver = fixture.resolver(t)
	resolver.xgoVersion = "v1.9.0"
	resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})
}
