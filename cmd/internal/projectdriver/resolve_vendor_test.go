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
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/mod/modload"
	"github.com/goplus/xgo/x/xgoprojs"
)

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

func TestResolveVendorIgnoresUnmarkedDependencyDriver(t *testing.T) {
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
	mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/app\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\ndriver v1 example.test/framework/cmd/driver\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// unmarked dependency driver-backed project\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("unmarked vendor dependency driver = %v, want ErrNotHandled", err)
	}
}

func TestResolveDriverVendorFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	mustMkdirAll(t, filepath.Join(fixture.app, "vendor"))
	mustWriteFile(t, filepath.Join(fixture.app, "vendor", "modules.txt"), "# fixture\n")
	_, err := fixture.resolver(t).Resolve(context.Background(), &xgoprojs.DirProj{Dir: fixture.project})
	if !errors.Is(err, ErrDriverVendorUnsupported) {
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
				if !errors.Is(err, ErrDriverVendorUnsupported) {
					t.Fatalf("external class vendor target = %v, want ErrDriverVendorUnsupported", err)
				}
			} else if !errors.Is(err, ErrNotHandled) {
				t.Fatalf("main-module legacy vendor target = %v, want ErrNotHandled", err)
			}
		})
	}
}

func TestResolveDriverTargetStillRejectsVendor(t *testing.T) {
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
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\ndriver v1 example.test/framework/cmd/driver\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// driver-backed project\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
	if !errors.Is(err, ErrDriverVendorUnsupported) {
		t.Fatalf("driver vendor target = %v", err)
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
	mustMkdirAll(t, filepath.Join(framework, "cmd", "driver"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	mustWriteFile(t, filepath.Join(app, "main.go"), "package app\n\nimport _ \"example.test/framework/cmd/driver\"\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), `xgo 1.8
project main.foo Game example.test/framework
driver v1 example.test/framework/cmd/driver
`)
	mustWriteFile(t, filepath.Join(framework, "cmd", "driver", "driver.go"), "package driver\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// driver-backed project\n")
	mustRunGo(t, app, "off", "mod", "vendor")
	vendoredFramework := filepath.Join(app, "vendor", "example.test", "framework")
	if _, err := os.Stat(filepath.Join(vendoredFramework, "cmd", "driver", "driver.go")); err != nil {
		t.Fatalf("real vendor snapshot omitted imported driver package: %v", err)
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
			if _, err := resolver.Resolve(context.Background(), target); !errors.Is(err, ErrDriverVendorUnsupported) {
				t.Fatalf("real vendored class target = %v, want ErrDriverVendorUnsupported", err)
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
	mustMkdirAll(t, filepath.Join(framework, "cmd", "driver"))
	mustWriteFile(t, filepath.Join(root, "go.work"), "go 1.25\n\nuse ./app\n\nreplace example.test/framework => ./framework\n")
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.2.3 //xgo:class\n")
	mustWriteFile(t, filepath.Join(app, "main.go"), "package app\n\nimport _ \"example.test/framework/cmd/driver\"\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\ndriver v1 example.test/framework/cmd/driver\n")
	mustWriteFile(t, filepath.Join(framework, "cmd", "driver", "driver.go"), "package driver\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// driver-backed project\n")
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
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project}); !errors.Is(err, ErrDriverVendorUnsupported) {
		t.Fatalf("workspace vendored class target = %v, want ErrDriverVendorUnsupported", err)
	}
}
