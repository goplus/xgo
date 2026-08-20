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

package runtimeprovider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/x/xgoprojs"
	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"
)

type runtimeFixture struct {
	root      string
	app       string
	project   string
	framework string
	mainFile  string
}

func newRuntimeFixture(t *testing.T) runtimeFixture {
	t.Helper()
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	project := filepath.Join(app, "game")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, filepath.Join(project, "pack"))
	mustMkdirAll(t, filepath.Join(framework, "cmd", "provider"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), `xgo 1.8

project main.foo Game example.test/framework
class *.bar Worker
pack pack index.data
runtime v1 example.test/framework/cmd/provider
`)
	mustWriteFile(t, filepath.Join(framework, "cmd", "provider", "main.go"), fakeProviderSource)
	mainFile := filepath.Join(project, "main.foo")
	mustWriteFile(t, mainFile, "// fake project source\n")
	mustWriteFile(t, filepath.Join(project, "pack", "index.data"), "{}\n")
	return runtimeFixture{root: root, app: app, project: project, framework: framework, mainFile: mainFile}
}

func (f runtimeFixture) resolver(t *testing.T, flags ...string) *Resolver {
	t.Helper()
	resolver, err := NewResolver(context.Background(), f.app, flags)
	if err != nil {
		t.Fatal(err)
	}
	resolver.xgoVersion = "v99.0.0"
	return resolver
}

func resolveRuntime(t *testing.T, resolver *Resolver, target xgoprojs.Proj) *Runtime {
	t.Helper()
	runtime, err := resolver.Resolve(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	return runtime
}

func TestLoadResolvedClassesWithoutClassMetadata(t *testing.T) {
	root := t.TempDir()
	goMod := filepath.Join(root, "go.mod")
	mustModuleFile(t, goMod, "example.test/plain")
	target := ResolvedModule{
		Selected: ModuleRef{Path: "example.test/plain", Dir: root, GoMod: goMod},
		Main:     true,
	}
	module, hasClass, err := loadResolvedClasses(&effectiveGraph{
		Target:        target,
		TargetModFile: fileIdentity{Path: goMod},
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasClass || module != nil {
		t.Fatalf("loadResolvedClasses() = %#v, %t; want nil, false", module, hasClass)
	}
}

func TestResolveRuntimeTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	for name, target := range map[string]xgoprojs.Proj{
		"directory": &xgoprojs.DirProj{Dir: fixture.project},
		"file":      &xgoprojs.FilesProj{Files: []string{fixture.mainFile}},
		"package":   &xgoprojs.PkgPathProj{Path: "example.test/app/game"},
	} {
		t.Run(name, func(t *testing.T) {
			rt := resolveRuntime(t, fixture.resolver(t), target)
			if rt.ProjectDir != canonicalDir(t, fixture.project) || rt.ProjectFile != canonicalFile(t, fixture.mainFile) {
				t.Fatalf("project identity = %q, %q", rt.ProjectDir, rt.ProjectFile)
			}
			if rt.ProviderPackage != "example.test/framework/cmd/provider" || rt.Protocol != "v1" {
				t.Fatalf("provider = %#v", rt)
			}
			if rt.Origin.Selected.Path != "example.test/framework" || rt.Origin.Selected.Version != "v1.2.3" || rt.Origin.Replace == nil {
				t.Fatalf("origin = %#v", rt.Origin)
			}
			if rt.Origin.Selected.Dir != "" || rt.Origin.Replace.Path != canonicalDir(t, fixture.framework) {
				t.Fatalf("replacement identity was flattened: %#v", rt.Origin)
			}
			if rt.PackDir != "pack" || rt.PackIndex != "index.data" || len(rt.GoxModSHA256) != 64 {
				t.Fatalf("metadata = %#v", rt)
			}
		})
	}
}

func TestResolveRejectsAnyNestedRuntimeDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	resolver := fixture.resolver(t)
	target := &xgoprojs.DirProj{Dir: fixture.project}
	guards := map[string]string{
		"same provider":      runtimeGuard(canonicalDir(t, fixture.project), "example.test/framework/cmd/provider"),
		"different provider": runtimeGuard(canonicalDir(t, fixture.app), "example.test/other/cmd/provider"),
	}
	for name, guard := range guards {
		t.Run(name, func(t *testing.T) {
			t.Setenv(runtimeGuardEnv, guard)
			if _, err := resolver.Resolve(context.Background(), target); !errors.Is(err, ErrRuntimeRecursive) {
				t.Fatalf("Resolve() = %v, want ErrRuntimeRecursive", err)
			}
		})
	}
}

func TestResolvePackageTargetKeepsCallerGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	dependencyProject := filepath.Join(fixture.framework, "example")
	mustMkdirAll(t, filepath.Join(dependencyProject, "pack"))
	mustWriteFile(t, filepath.Join(dependencyProject, "main.foo"), "// dependency project\n")
	mustWriteFile(t, filepath.Join(dependencyProject, "pack", "index.data"), "{}\n")

	resolver := fixture.resolver(t)
	rt := resolveRuntime(t, resolver, &xgoprojs.PkgPathProj{Path: "example.test/framework/example"})
	if rt.ModuleRoot != canonicalDir(t, fixture.framework) || rt.Graph.WorkDir != canonicalDir(t, fixture.app) {
		t.Fatalf("graph roots = module %q, work %q", rt.ModuleRoot, rt.Graph.WorkDir)
	}
	if rt.Origin.Main || rt.Origin.Replace == nil || rt.Origin.Selected.Version != "v1.2.3" {
		t.Fatalf("caller replacement graph was lost: %#v", rt.Origin)
	}
	if err := validateProvider(context.Background(), rt); err != nil {
		t.Fatalf("provider validation switched graphs: %v", err)
	}
}

func TestResolveXGoOnlyPackageTargetWithModfile(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	data, err := os.ReadFile(filepath.Join(fixture.app, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(fixture.app, "runtime.mod"), string(data))
	resolver := fixture.resolver(t, "-modfile=runtime.mod")
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/game"}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveVersionedPackageUsesRequestedVersionMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	proxy := t.TempDir()
	writeModuleProxy(t, proxy, "example.test/framework", map[string]map[string]string{
		"v1.0.0": {
			"go.mod":                   "module example.test/framework\n\ngo 1.25\n",
			"gox.mod":                  "xgo 1.8\nproject main.foo Game example.test/framework\nruntime v1 example.test/framework/cmd/provider\n",
			"cmd/provider/provider.go": "package provider\n",
		},
	})
	writeModuleProxy(t, proxy, "example.test/app/game", map[string]map[string]string{
		"v1.3.0": {
			"go.mod":     "module example.test/app/game\n\ngo 1.25\n",
			"legacy.foo": "// nested legacy project\n",
		},
	})
	writeModuleProxy(t, proxy, "example.test/app", map[string]map[string]string{
		"v1.0.0": {
			"go.mod":  "module example.test/app\n\ngo 1.25\n",
			"main.go": "package main\nfunc main() {}\n",
		},
		"v1.1.0": {
			"go.mod":           "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n",
			"main.go":          "package main\nfunc main() {}\n",
			"main.foo":         "// runtime project\n",
			"ordinary/main.go": "package ordinary\n",
		},
		"v1.2.0": {
			"go.mod":        "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n",
			"main.foo":      "// XGo-only runtime project\n",
			"game/main.foo": "// XGo-only subdirectory runtime project\n",
		},
		"v1.3.0": {
			"go.mod":        "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n",
			"main.foo":      "// latest runtime project\n",
			"game/main.foo": "// parent project hidden by nested module\n",
		},
	})
	cache := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(cache, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0700)
			}
			return os.Chmod(path, 0600)
		})
	})
	proxyURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}).String()
	t.Setenv("GOMODCACHE", cache)
	t.Setenv("GOPROXY", proxyURL)
	t.Setenv("GOSUMDB", "off")
	bin := t.TempDir()
	t.Setenv("GOBIN", bin)
	t.Setenv("GOWORK", "off")

	for _, test := range []struct {
		name          string
		callerVersion string
		callerClass   bool
		target        string
		wantRuntime   bool
	}{
		{name: "legacy requested version ignores runtime caller graph", callerVersion: "v1.1.0", callerClass: true, target: "example.test/app@v1.0.0"},
		{name: "runtime requested version ignores legacy caller graph", callerVersion: "v1.0.0", target: "example.test/app@v1.1.0", wantRuntime: true},
		{name: "latest query uses selected version metadata", callerVersion: "v1.0.0", target: "example.test/app@latest", wantRuntime: true},
		{name: "XGo-only runtime version is classified", callerVersion: "v1.0.0", target: "example.test/app@v1.2.0", wantRuntime: true},
		{name: "XGo-only subdirectory uses parent module", callerVersion: "v1.0.0", target: "example.test/app/game@v1.2.0", wantRuntime: true},
		{name: "nested module wins over parent prefix", callerVersion: "v1.0.0", target: "example.test/app/game@v1.3.0"},
		{name: "ordinary package in runtime version remains legacy", callerVersion: "v1.0.0", target: "example.test/app/ordinary@v1.1.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := t.TempDir()
			marker := ""
			if test.callerClass {
				marker = " //xgo:class"
			}
			mustWriteFile(t, filepath.Join(caller, "go.mod"), fmt.Sprintf("module example.test/caller\n\ngo 1.25\n\nrequire example.test/app %s%s\n", test.callerVersion, marker))
			resolver, err := NewResolver(context.Background(), caller, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: test.target})
			if test.wantRuntime {
				if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "@version") {
					t.Fatalf("Resolve(%q) = %v, want explicit @version rejection", test.target, err)
				}
				return
			}
			if !errors.Is(err, ErrNotHandled) {
				t.Fatalf("Resolve(%q) = %v, want ErrNotHandled", test.target, err)
			}
		})
	}
	entries, err := os.ReadDir(bin)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("versioned package probe installed files into GOBIN: %v", entries)
	}
}

func TestResolveVersionedPackageCanceledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/caller\n\ngo 1.25\n")
	t.Setenv("GOWORK", "off")
	bin := t.TempDir()
	t.Setenv("GOBIN", bin)
	resolver, err := NewResolver(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = resolver.Resolve(ctx, &xgoprojs.PkgPathProj{Path: "example.test/app@latest"})
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrNotHandled) {
		t.Fatalf("canceled versioned Resolve() = %v, want context.Canceled", err)
	}
	entries, readErr := os.ReadDir(bin)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("canceled versioned probe installed files into GOBIN: %v", entries)
	}
}

func TestLoadRuntimeModulePropagatesOptionalMetadataReadErrors(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	goxMod := filepath.Join(dir, "gox.mod")
	mustWriteFile(t, goMod, "module example.test/app\n\ngo 1.25\n")
	mustMkdirAll(t, goxMod)
	// A valid legacy fallback must not mask a non-NotExist error from gox.mod.
	mustWriteFile(t, filepath.Join(dir, "gop.mod"), "gop 1.8\nproject main.foo Game example.test/app\n")
	if _, err := loadRuntimeModule(goMod, goxMod); err == nil || !strings.Contains(err.Error(), "gox.mod") {
		t.Fatalf("loadRuntimeModule() = %v, want explicit gox.mod read error", err)
	}
}

func TestLoadRuntimeModuleFallsBackToGopModWhenGoxModIsAbsent(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	gopMod := filepath.Join(dir, "gop.mod")
	mustWriteFile(t, goMod, "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, gopMod, "gop 1.1\nproject main.foo Game example.test/app\n")
	loaded, err := loadRuntimeModule(goMod, filepath.Join(dir, "gox.mod"))
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

func TestResolvePackageTargetRejectsUnmarkedDependencyRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
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
		t.Fatalf("unmarked dependency runtime = %v, want ErrNotHandled", err)
	}
}

func TestResolveOrdinaryGoSubpackageInsideRuntimeModuleUsesLegacyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	ordinary := filepath.Join(fixture.framework, "ordinary")
	mustMkdirAll(t, ordinary)
	mustWriteFile(t, filepath.Join(ordinary, "main.go"), "package ordinary\n")
	resolver := fixture.resolver(t)
	_, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/framework/ordinary"})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("ordinary Go subpackage = %v", err)
	}
}

func TestUnsupportedPatternOnlyRejectsMatchedRuntimeTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	legacy := filepath.Join(fixture.app, "legacy")
	mustMkdirAll(t, legacy)
	mustWriteFile(t, filepath.Join(legacy, "main.go"), "package main\nfunc main() {}\n")
	nested := filepath.Join(legacy, "nested")
	mustMkdirAll(t, nested)
	mustWriteFile(t, filepath.Join(nested, "go.mod"), "module example.test/nested\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(nested, "main.foo"), "// nested runtime project must be outside the pattern\n")
	resolver := fixture.resolver(t)
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: filepath.Join(legacy, "...")}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("legacy pattern = %v, want ErrNotHandled", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: filepath.Join(fixture.project, "...")}); err == nil || !strings.Contains(err.Error(), "directory pattern") {
		t.Fatalf("runtime pattern error = %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: filepath.Join(fixture.app, "...")}); err == nil || !strings.Contains(err.Error(), "directory pattern") {
		t.Fatalf("parent pattern containing runtime project error = %v", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/legacy/..."}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("legacy package pattern = %v, want ErrNotHandled", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/..."}); err == nil || !strings.Contains(err.Error(), "package pattern") {
		t.Fatalf("package pattern containing runtime project error = %v", err)
	}
}

func TestResolveRuntimeWithoutPack(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	goxmod := filepath.Join(fixture.framework, "gox.mod")
	data, err := os.ReadFile(goxmod)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte("pack pack index.data\n"), nil)
	if err := os.WriteFile(goxmod, data, 0644); err != nil {
		t.Fatal(err)
	}
	rt := resolveRuntime(t, fixture.resolver(t), &xgoprojs.DirProj{Dir: fixture.project})
	if rt.PackDir != "" || rt.PackIndex != "" {
		t.Fatalf("pack = %q, %q", rt.PackDir, rt.PackIndex)
	}
	args, err := providerArgs(rt, actionRun, BuildPolicy{}, "", "", nil)
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
	origin := ResolvedModule{Selected: ModuleRef{Path: "example.test/provider", Dir: dir, GoMod: filepath.Join(dir, "go.mod")}, Main: true}
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

func TestResolveRuntimeRejectsAmbiguousAndMultiFile(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
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

func TestResolveNonRuntimeNotHandled(t *testing.T) {
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
	for _, want := range []string{"1.8.1", "(devel)", "runtime-provider capability 1.8.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("checkRequiredXGo() error = %q, want %q", err, want)
		}
	}
}

func mustRunGo(t *testing.T, dir, goWork string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = replaceEnv(replaceEnv(os.Environ(), "GOWORK", goWork), "GOFLAGS", "")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func canonicalDir(t *testing.T, path string) string {
	t.Helper()
	got, err := canonicalExistingDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func canonicalFile(t *testing.T, path string) string {
	t.Helper()
	got, err := canonicalExistingFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

const fakeProviderSource = `package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	if marker := os.Getenv("FAKE_PROVIDER_MARKER"); marker != "" {
		_ = os.WriteFile(marker, []byte("started"), 0600)
	}
	if value := os.Getenv("FAKE_PROVIDER_EXIT"); value != "" {
		code, _ := strconv.Atoi(value)
		os.Exit(code)
	}
	if os.Getenv("FAKE_PROVIDER_BLOCK") == "1" {
		for {
			time.Sleep(time.Hour)
		}
	}
	if len(os.Args) < 3 || os.Args[1] != "xgo-runtime-v1" {
		os.Exit(90)
	}
	switch os.Args[2] {
	case "run":
		for i, arg := range os.Args[3:] {
			if arg == "--" {
				fmt.Printf("run-args=%s\n", strings.Join(os.Args[i+4:], "|"))
				return
			}
		}
		os.Exit(91)
	case "build":
		var output string
		for _, arg := range os.Args[3:] {
			if strings.HasPrefix(arg, "--output=") {
				output = strings.TrimPrefix(arg, "--output=")
			}
		}
		self, err := os.Executable()
		if err != nil { panic(err) }
		data, err := os.ReadFile(self)
		if err != nil { panic(err) }
		if err := os.WriteFile(output, data, 0755); err != nil { panic(err) }
		if runtime.GOOS == "darwin" {
			if data, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", output).CombinedOutput(); err != nil {
				fmt.Fprintln(os.Stderr, string(data))
				os.Exit(93)
			}
		}
	default:
		os.Exit(92)
	}
}
`

func writeModuleProxy(t *testing.T, proxy, modulePath string, versions map[string]map[string]string) {
	t.Helper()
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	moduleProxyDir := filepath.Join(proxy, filepath.FromSlash(escaped), "@v")
	if err := os.MkdirAll(moduleProxyDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionNames := make([]string, 0, len(versions))
	for version := range versions {
		versionNames = append(versionNames, version)
	}
	sort.Strings(versionNames)
	if err := os.WriteFile(filepath.Join(moduleProxyDir, "list"), []byte(strings.Join(versionNames, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, version := range versionNames {
		files := versions[version]
		source := t.TempDir()
		for name, content := range files {
			path := filepath.Join(source, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
		mod := module.Version{Path: modulePath, Version: version}
		zipPath := filepath.Join(moduleProxyDir, version+".zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatal(err)
		}
		zipErr := modzip.CreateFromDir(zipFile, mod, source)
		closeErr := zipFile.Close()
		if zipErr != nil {
			t.Fatal(zipErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		info := fmt.Sprintf("{\"Version\":%q,\"Time\":\"2026-01-01T00:00:00Z\"}\n", version)
		if err := os.WriteFile(filepath.Join(moduleProxyDir, version+".info"), []byte(info), 0644); err != nil {
			t.Fatal(err)
		}
		goMod, ok := files["go.mod"]
		if !ok {
			t.Fatalf("module %s@%s has no go.mod", modulePath, version)
		}
		if err := os.WriteFile(filepath.Join(moduleProxyDir, version+".mod"), []byte(goMod), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
