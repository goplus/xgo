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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/x/xgoprojs"
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
			rt, err := fixture.resolver(t).Resolve(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
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
	rt, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/framework/example"})
	if err != nil {
		t.Fatal(err)
	}
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
	rt, err := fixture.resolver(t).Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err != nil {
		t.Fatal(err)
	}
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

func TestResolveRuntimeVendorFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	mustMkdirAll(t, filepath.Join(fixture.app, "vendor"))
	mustWriteFile(t, filepath.Join(fixture.app, "vendor", "modules.txt"), "# fixture\n")
	_, err := fixture.resolver(t).Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if !errors.Is(err, ErrRuntimeVendorUnsupported) {
		t.Fatalf("vendor error = %v", err)
	}
}

func TestResolveVendorLeavesLegacyTargetsUnhandled(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	tests := []struct {
		name  string
		class bool
		auto  bool
		flags []string
	}{
		{name: "main-explicit", flags: []string{"-mod=vendor"}},
		{name: "main-automatic", auto: true},
		{name: "class-explicit", class: true, flags: []string{"-mod=vendor"}},
		{name: "class-automatic", class: true, auto: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := t.TempDir()
			project := filepath.Join(app, "game")
			mustMkdirAll(t, project)
			mustMkdirAll(t, filepath.Join(app, "vendor"))
			mustWriteFile(t, filepath.Join(app, "vendor", "modules.txt"), "# vendored\n")
			if test.class {
				framework := filepath.Join(app, "framework")
				mustMkdirAll(t, framework)
				mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
				mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\n")
				mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.2.3 //xgo:class\n\nreplace example.test/framework => ./framework\n")
			} else {
				mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n")
				mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/app\n")
			}
			mustWriteFile(t, filepath.Join(project, "main.foo"), "// legacy project\n")
			t.Setenv("GOWORK", "off")
			resolver, err := NewResolver(context.Background(), app, test.flags)
			if err != nil {
				t.Fatal(err)
			}
			if test.auto {
				resolver.policy.graph.ModMode = ""
			}
			_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
			if !errors.Is(err, ErrNotHandled) {
				t.Fatalf("legacy vendor target = %v", err)
			}
		})
	}
}

func TestResolveRuntimeTargetStillRejectsVendor(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	app := t.TempDir()
	project := filepath.Join(app, "game")
	framework := filepath.Join(app, "framework")
	mustMkdirAll(t, project)
	mustMkdirAll(t, framework)
	mustMkdirAll(t, filepath.Join(app, "vendor"))
	mustWriteFile(t, filepath.Join(app, "vendor", "modules.txt"), "# vendored\n")
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.2.3 //xgo:class\n\nreplace example.test/framework => ./framework\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\nruntime v1 example.test/framework/cmd/provider\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// runtime project\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
	if !errors.Is(err, ErrRuntimeVendorUnsupported) {
		t.Fatalf("runtime vendor target = %v", err)
	}
}

func TestResolveVendorIgnoresUnmarkedDependencyRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	app := t.TempDir()
	project := filepath.Join(app, "game")
	framework := filepath.Join(app, "framework")
	mustMkdirAll(t, project)
	mustMkdirAll(t, framework)
	mustMkdirAll(t, filepath.Join(app, "vendor"))
	mustWriteFile(t, filepath.Join(app, "vendor", "modules.txt"), "# vendored\n")
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3

replace example.test/framework => ./framework
`)
	// The target module has class metadata, so vendor preflight must make a
	// positive trust decision rather than returning early before probing.
	mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/app\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\nruntime v1 example.test/framework/cmd/provider\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// unmarked dependency runtime project\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("unmarked vendor dependency runtime = %v, want ErrNotHandled", err)
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

func TestResolverDefersAmbientPolicyFailureForLegacy(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/plain\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {}\n")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "'unterminated")
	resolver, err := NewResolver(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: dir}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("legacy target = %v", err)
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

func TestRuntimeRunAndBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	resolver := fixture.resolver(t, "-trimpath=true", "-buildvcs=false")
	rt, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status, err := resolver.Run(context.Background(), rt, []string{"", "a b", "--"}, Streams{Stdout: &stdout, Stderr: &stderr})
	if err != nil || status.Code != 0 || status.Signaled {
		t.Fatalf("run = %#v, %v, stderr=%s", status, err, &stderr)
	}
	if got := strings.TrimSpace(stdout.String()); got != "run-args=|a b|--" {
		t.Fatalf("stdout = %q", got)
	}

	final := filepath.Join(fixture.root, "bin", "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(final), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	status, gotFinal, err := resolver.Build(context.Background(), rt, final, Streams{Stdout: &stdout, Stderr: &stderr})
	if err != nil || status.Code != 0 || gotFinal != final {
		t.Fatalf("build = %#v, %q, %v, stderr=%s", status, gotFinal, err, &stderr)
	}
	info, err := os.Stat(final)
	if err != nil || info.Size() <= int64(len("old")) {
		t.Fatalf("artifact = %#v, %v", info, err)
	}
}

func TestRuntimeBuildFailurePreservesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	resolver := fixture.resolver(t)
	rt, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(fixture.root, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_PROVIDER_EXIT", "42")
	status, _, err := resolver.Build(context.Background(), rt, final, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	if err != nil || status.Code != 42 {
		t.Fatalf("build failure = %#v, %v", status, err)
	}
	data, readErr := os.ReadFile(final)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("old output changed: %q, %v", data, readErr)
	}
	assertNoOutputWorkDirs(t, filepath.Dir(final))
}

func TestRuntimeBuildCancellationPreservesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	resolver := fixture.resolver(t)
	rt, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(fixture.root, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(fixture.root, "provider-started")
	t.Setenv("FAKE_PROVIDER_MARKER", marker)
	t.Setenv("FAKE_PROVIDER_BLOCK", "1")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = resolver.Build(ctx, rt, final, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	}()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("runtime provider did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("canceled runtime provider did not exit")
	}
	data, readErr := os.ReadFile(final)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("canceled build changed old output: %q, %v", data, readErr)
	}
	assertNoOutputWorkDirs(t, filepath.Dir(final))
}

func assertNoOutputWorkDirs(t *testing.T, parent string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(parent, ".xgo-runtime-output-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("runtime output work directories remain: %v", matches)
	}
}

func TestRuntimeInstallUsesEffectiveGOBIN(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newRuntimeFixture(t)
	resolver := fixture.resolver(t)
	rt, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(fixture.root, "custom-bin")
	t.Setenv("GOBIN", bin)
	status, final, err := resolver.Install(context.Background(), rt, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	if err != nil || status.Code != 0 {
		t.Fatalf("install = %#v, %q, %v", status, final, err)
	}
	want := filepath.Join(bin, executableName("game"))
	if final != want {
		t.Fatalf("install output = %q, want %q", final, want)
	}
	if info, err := os.Stat(final); err != nil || info.Size() == 0 {
		t.Fatalf("installed artifact = %#v, %v", info, err)
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
