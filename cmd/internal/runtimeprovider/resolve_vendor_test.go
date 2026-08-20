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
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/modload"
	"github.com/goplus/xgo/x/xgoprojs"
)

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

func TestResolveVendorClassifiesLegacyTargetsConservatively(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	tests := []struct {
		name            string
		class           bool
		auto            bool
		flags           []string
		wantUnsupported bool
	}{
		{name: "main-explicit", flags: []string{"-mod=vendor"}},
		{name: "main-automatic", auto: true},
		{name: "class-explicit", class: true, flags: []string{"-mod=vendor"}, wantUnsupported: true},
		{name: "class-automatic", class: true, auto: true, wantUnsupported: true},
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
			if test.wantUnsupported {
				if !errors.Is(err, ErrRuntimeVendorUnsupported) {
					t.Fatalf("external class vendor target = %v, want ErrRuntimeVendorUnsupported", err)
				}
			} else if !errors.Is(err, ErrNotHandled) {
				t.Fatalf("main-module legacy vendor target = %v, want ErrNotHandled", err)
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

func TestResolveRealModuleVendorExternalClassFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	app := filepath.Join(root, "app")
	project := filepath.Join(app, "game")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, project)
	mustMkdirAll(t, filepath.Join(framework, "cmd", "provider"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	mustWriteFile(t, filepath.Join(app, "main.go"), "package app\n\nimport _ \"example.test/framework/cmd/provider\"\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), `xgo 1.8
project main.foo Game example.test/framework
runtime v1 example.test/framework/cmd/provider
`)
	mustWriteFile(t, filepath.Join(framework, "cmd", "provider", "provider.go"), "package provider\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// runtime project\n")
	mustRunGo(t, app, "off", "mod", "vendor")
	vendoredFramework := filepath.Join(app, "vendor", "example.test", "framework")
	if _, err := os.Stat(filepath.Join(vendoredFramework, "cmd", "provider", "provider.go")); err != nil {
		t.Fatalf("real vendor snapshot omitted imported provider package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vendoredFramework, "gox.mod")); !os.IsNotExist(err) {
		t.Fatalf("real vendor snapshot unexpectedly contains module-root gox.mod: %v", err)
	}
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]xgoprojs.Proj{
		"directory": &xgoprojs.DirProj{Dir: project},
		"package":   &xgoprojs.PkgPathProj{Path: "example.test/app/game"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolver.Resolve(context.Background(), target); !errors.Is(err, ErrRuntimeVendorUnsupported) {
				t.Fatalf("real vendored class target = %v, want ErrRuntimeVendorUnsupported", err)
			}
		})
	}
}

func TestResolveRealModuleVendorPlainPackageUsesLegacyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	app := filepath.Join(root, "app")
	dependency := filepath.Join(root, "dependency")
	mustMkdirAll(t, app)
	mustMkdirAll(t, dependency)
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n\nrequire example.test/dependency v1.0.0\n\nreplace example.test/dependency => ../dependency\n")
	mustWriteFile(t, filepath.Join(app, "main.go"), "package app\n\nimport _ \"example.test/dependency\"\n")
	mustWriteFile(t, filepath.Join(dependency, "go.mod"), "module example.test/dependency\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dependency, "dependency.go"), "package dependency\n")
	mustRunGo(t, app, "off", "mod", "vendor")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app"}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("plain vendored package = %v, want ErrNotHandled", err)
	}
}

func TestResolveRealWorkspaceVendorClassFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	app := filepath.Join(root, "app")
	project := filepath.Join(app, "game")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, project)
	mustMkdirAll(t, filepath.Join(framework, "cmd", "provider"))
	mustWriteFile(t, filepath.Join(root, "go.work"), "go 1.25\n\nuse ./app\n\nreplace example.test/framework => ./framework\n")
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.2.3 //xgo:class\n")
	mustWriteFile(t, filepath.Join(app, "main.go"), "package app\n\nimport _ \"example.test/framework/cmd/provider\"\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\nruntime v1 example.test/framework/cmd/provider\n")
	mustWriteFile(t, filepath.Join(framework, "cmd", "provider", "provider.go"), "package provider\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// runtime project\n")
	goWork := filepath.Join(root, "go.work")
	mustRunGo(t, root, goWork, "work", "vendor")
	if _, err := os.Stat(filepath.Join(root, "vendor", "modules.txt")); err != nil {
		t.Fatalf("workspace vendor snapshot missing modules.txt: %v", err)
	}
	t.Setenv("GOWORK", goWork)
	resolver, err := NewResolver(context.Background(), app, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project}); !errors.Is(err, ErrRuntimeVendorUnsupported) {
		t.Fatalf("workspace vendored class target = %v, want ErrRuntimeVendorUnsupported", err)
	}
}

func TestResolveWorkspaceVendorPackageUsesTargetModuleMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	caller := filepath.Join(root, "caller")
	runtimeModule := filepath.Join(root, "runtime")
	runtimeProject := filepath.Join(runtimeModule, "game")
	legacyModule := filepath.Join(root, "legacy")
	legacyProject := filepath.Join(legacyModule, "game")
	mustMkdirAll(t, caller)
	mustMkdirAll(t, runtimeProject)
	mustMkdirAll(t, legacyProject)
	mustWriteFile(t, filepath.Join(root, "go.work"), `go 1.25

use (
	./caller
	./runtime
	./legacy
)
`)
	mustWriteFile(t, filepath.Join(caller, "go.mod"), "module example.test/caller\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(caller, "caller.go"), "package caller\n")
	mustWriteFile(t, filepath.Join(runtimeModule, "go.mod"), "module example.test/runtime\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(runtimeModule, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/runtime\nruntime v1 example.test/runtime/cmd/provider\n")
	mustWriteFile(t, filepath.Join(runtimeProject, "main.foo"), "// runtime project\n")
	mustWriteFile(t, filepath.Join(legacyModule, "go.mod"), "module example.test/legacy\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(legacyModule, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/legacy\n")
	mustWriteFile(t, filepath.Join(legacyProject, "main.legacy"), "// legacy project\n")
	goWork := filepath.Join(root, "go.work")
	t.Setenv("GOWORK", goWork)
	resolver, err := NewResolver(context.Background(), caller, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/runtime/game"}); !errors.Is(err, ErrRuntimeVendorUnsupported) {
		t.Fatalf("workspace runtime package in vendor mode = %v, want ErrRuntimeVendorUnsupported", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/legacy/game"}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("workspace legacy package in vendor mode = %v, want ErrNotHandled", err)
	}

	if runtime, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/runtime/game", false); err != nil || !runtime {
		t.Fatalf("workspace fallback runtime classification = %v, %v; want true", runtime, err)
	}
	if runtime, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/legacy/game", false); err != nil || runtime {
		t.Fatalf("workspace fallback legacy classification = %v, %v; want false", runtime, err)
	}
}

func TestProbeVendorWorkspacePackageRejectsExternalClassWithoutReadingReplacement(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	caller := filepath.Join(root, "caller")
	member := filepath.Join(root, "member")
	project := filepath.Join(member, "game")
	for _, dir := range []string{caller, project} {
		mustMkdirAll(t, dir)
	}
	mustWriteFile(t, filepath.Join(root, "go.work"), `go 1.25

use (
	./caller
	./member
)

replace example.test/framework => ./missing-live-replacement
`)
	mustWriteFile(t, filepath.Join(caller, "go.mod"), "module example.test/caller\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(member, "go.mod"), "module example.test/member\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n")
	mustWriteFile(t, filepath.Join(member, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/member\n")
	mustWriteFile(t, filepath.Join(project, "main.legacy"), "// indeterminate external class project\n")

	goWork := filepath.Join(root, "go.work")
	t.Setenv("GOWORK", goWork)
	resolver, err := NewResolver(context.Background(), caller, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/member/game", false); !errors.Is(err, ErrRuntimeVendorUnsupported) {
		t.Fatalf("workspace external class classification = %v, want ErrRuntimeVendorUnsupported", err)
	}
}

func TestProbeVendorWorkspacePackageUsesLongestMemberAndRejectsNestedModule(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	caller := filepath.Join(root, "caller")
	parent := filepath.Join(root, "parent")
	child := filepath.Join(root, "child")
	childProject := filepath.Join(child, "game")
	nested := filepath.Join(parent, "nested")
	nestedProject := filepath.Join(nested, "game")
	for _, dir := range []string{caller, parent, childProject, nestedProject} {
		mustMkdirAll(t, dir)
	}
	mustWriteFile(t, filepath.Join(root, "go.work"), "go 1.25\n\nuse (\n\t./caller\n\t./parent\n\t./child\n)\n")
	mustWriteFile(t, filepath.Join(caller, "go.mod"), "module example.test/caller\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(parent, "go.mod"), "module example.test/shared\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(parent, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/shared\n")
	mustWriteFile(t, filepath.Join(child, "go.mod"), "module example.test/shared/sub\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(child, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/shared/sub\nruntime v1 example.test/shared/sub/cmd/provider\n")
	mustWriteFile(t, filepath.Join(childProject, "main.foo"), "// runtime project in longest module match\n")
	mustWriteFile(t, filepath.Join(nested, "go.mod"), "module example.test/nested\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(nestedProject, "main.legacy"), "// nested module project\n")

	goWork := filepath.Join(root, "go.work")
	t.Setenv("GOWORK", goWork)
	resolver, err := NewResolver(context.Background(), caller, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if runtime, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/shared/sub/game", false); err != nil || !runtime {
		t.Fatalf("longest workspace member classification = %v, %v; want true", runtime, err)
	}
	if _, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/shared/nested/game", false); err == nil {
		t.Fatal("workspace package crossing a nested module boundary was classified")
	}
}

func TestResolveVendorRecursivePatternFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	app := t.TempDir()
	project := filepath.Join(app, "game")
	mustMkdirAll(t, project)
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/app\nruntime v1 example.test/app/cmd/provider\n")
	mustWriteFile(t, filepath.Join(app, "app.go"), "package app\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// runtime project\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]xgoprojs.Proj{
		"directory": &xgoprojs.DirProj{Dir: filepath.Join(app, "...")},
		"package":   &xgoprojs.PkgPathProj{Path: "example.test/app/..."},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolver.Resolve(context.Background(), target); !errors.Is(err, ErrRuntimeVendorUnsupported) {
				t.Fatalf("recursive vendor target = %v, want ErrRuntimeVendorUnsupported", err)
			}
		})
	}
}

func TestVendorRuntimeProbePropagatesFilesystemErrors(t *testing.T) {
	projects := []*modfile.Project{{Runtime: &modfile.Runtime{Protocol: "v1", Package: "example.test/provider"}}}
	if _, err := hasRuntimeProject(filepath.Join(t.TempDir(), "missing"), projects); err == nil {
		t.Fatal("missing project directory was classified as no runtime")
	}

	if runtime.GOOS == "windows" {
		t.Skip("self-referential symlink setup is not portable to Windows")
	}
	app := t.TempDir()
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/app\n")
	mustWriteFile(t, filepath.Join(app, "main.foo"), "// legacy project\n")
	if err := os.Symlink("vendor", filepath.Join(app, "vendor")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: app}); err == nil || errors.Is(err, ErrNotHandled) {
		t.Fatalf("vendor manifest I/O failure = %v, want explicit error", err)
	}
}

func TestEffectiveVendorModeMatchesGoCommandDefaults(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	vendorDir := filepath.Join(app, "vendor")
	mustMkdirAll(t, vendorDir)
	goMod := filepath.Join(app, "go.mod")
	mustWriteFile(t, goMod, "module example.test/app\n\ngo 1.21\n")
	loaded, err := modload.LoadFrom(goMod, filepath.Join(app, "gox.mod"))
	if err != nil {
		t.Fatal(err)
	}

	modulePolicy := GraphPolicy{GoWork: "off"}
	if vendor, err := effectiveVendorMode(modulePolicy, goMod, loaded.File); err != nil || !vendor {
		t.Fatalf("module vendor without modules.txt = %v, %v; want true", vendor, err)
	}
	mustWriteFile(t, filepath.Join(vendorDir, "modules.txt"), "## workspace\n")
	if vendor, err := effectiveVendorMode(modulePolicy, goMod, loaded.File); err != nil || vendor {
		t.Fatalf("workspace manifest outside workspace = %v, %v; want false", vendor, err)
	}
	mustWriteFile(t, filepath.Join(vendorDir, "modules.txt"), "# module manifest\n")
	if vendor, err := effectiveVendorMode(modulePolicy, goMod, loaded.File); err != nil || !vendor {
		t.Fatalf("module manifest in module mode = %v, %v; want true", vendor, err)
	}

	goWork := filepath.Join(root, "go.work")
	mustWriteFile(t, goWork, "go 1.21\n\nuse ./app\n")
	workspaceVendor := filepath.Join(root, "vendor")
	mustMkdirAll(t, workspaceVendor)
	mustWriteFile(t, filepath.Join(workspaceVendor, "modules.txt"), "# module manifest\n")
	workspacePolicy := GraphPolicy{GoWork: goWork}
	if vendor, err := effectiveVendorMode(workspacePolicy, goMod, loaded.File); err != nil || vendor {
		t.Fatalf("module manifest in workspace mode = %v, %v; want false", vendor, err)
	}
	mustWriteFile(t, filepath.Join(workspaceVendor, "modules.txt"), "## workspace; future annotation\n")
	if vendor, err := effectiveVendorMode(workspacePolicy, goMod, loaded.File); err != nil || !vendor {
		t.Fatalf("Go 1.21 workspace manifest = %v, %v; want true", vendor, err)
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
