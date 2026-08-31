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
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/xgo/x/xgoprojs"
)

func TestResolveWorkspaceVendorPackageUsesTargetModuleMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	root := t.TempDir()
	caller := filepath.Join(root, "caller")
	driverModule := filepath.Join(root, "driver")
	driverProject := filepath.Join(driverModule, "game")
	legacyModule := filepath.Join(root, "legacy")
	legacyProject := filepath.Join(legacyModule, "game")
	mustMkdirAll(t, caller)
	mustMkdirAll(t, driverProject)
	mustMkdirAll(t, legacyProject)
	mustWriteFile(t, filepath.Join(root, "go.work"), `go 1.25

use (
	./caller
	./driver
	./legacy
)
`)
	mustWriteFile(t, filepath.Join(caller, "go.mod"), "module example.test/caller\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(caller, "caller.go"), "package caller\n")
	mustWriteFile(t, filepath.Join(driverModule, "go.mod"), "module example.test/driver\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(driverModule, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/driver\ndriver v1 example.test/driver/cmd/driver\n")
	mustWriteFile(t, filepath.Join(driverProject, "main.foo"), "// driver-backed project\n")
	mustWriteFile(t, filepath.Join(legacyModule, "go.mod"), "module example.test/legacy\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(legacyModule, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/legacy\n")
	mustWriteFile(t, filepath.Join(legacyProject, "main.legacy"), "// legacy project\n")
	goWork := filepath.Join(root, "go.work")
	t.Setenv("GOWORK", goWork)
	resolver, err := NewResolver(context.Background(), caller, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/driver/game"}); !errors.Is(err, ErrDriverVendorUnsupported) {
		t.Fatalf("workspace driver package in vendor mode = %v, want ErrDriverVendorUnsupported", err)
	}
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/legacy/game"}); !errors.Is(err, ErrNotHandled) {
		t.Fatalf("workspace legacy package in vendor mode = %v, want ErrNotHandled", err)
	}

	if driver, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/driver/game", false); err != nil || !driver {
		t.Fatalf("workspace fallback driver classification = %v, %v; want true", driver, err)
	}
	if driver, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/legacy/game", false); err != nil || driver {
		t.Fatalf("workspace fallback legacy classification = %v, %v; want false", driver, err)
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
	if _, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/member/game", false); !errors.Is(err, ErrDriverVendorUnsupported) {
		t.Fatalf("workspace external class classification = %v, want ErrDriverVendorUnsupported", err)
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
	mustWriteFile(t, filepath.Join(child, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/shared/sub\ndriver v1 example.test/shared/sub/cmd/driver\n")
	mustWriteFile(t, filepath.Join(childProject, "main.foo"), "// driver-backed project in longest module match\n")
	mustWriteFile(t, filepath.Join(nested, "go.mod"), "module example.test/nested\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(nestedProject, "main.legacy"), "// nested module project\n")

	goWork := filepath.Join(root, "go.work")
	t.Setenv("GOWORK", goWork)
	resolver, err := NewResolver(context.Background(), caller, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if driver, err := resolver.probeVendorWorkspacePackage(context.Background(), "example.test/shared/sub/game", false); err != nil || !driver {
		t.Fatalf("longest workspace member classification = %v, %v; want true", driver, err)
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
	mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/app\ndriver v1 example.test/app/cmd/driver\n")
	mustWriteFile(t, filepath.Join(app, "app.go"), "package app\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// driver-backed project\n")
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
			if _, err := resolver.Resolve(context.Background(), target); !errors.Is(err, ErrDriverVendorUnsupported) {
				t.Fatalf("recursive vendor target = %v, want ErrDriverVendorUnsupported", err)
			}
		})
	}
}

func TestVendorDriverProbePropagatesFilesystemErrors(t *testing.T) {
	projects := []*modfile.Project{{Driver: &modfile.Driver{Protocol: "v1", Package: "example.test/driver"}}}
	if _, err := hasDriverProject(filepath.Join(t.TempDir(), "missing"), projects); err == nil {
		t.Fatal("missing project directory was classified as no driver")
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

func TestVendorDriverProbeRejectsProjectSymlinksAndSpecialFiles(t *testing.T) {
	projects := []*modfile.Project{{Ext: ".foo", FullExt: "main.foo", Driver: &modfile.Driver{Protocol: "v1", Package: "example.test/driver"}}}
	for _, special := range []string{"symlink", "directory"} {
		t.Run(special, func(t *testing.T) {
			dir := t.TempDir()
			project := filepath.Join(dir, "main.foo")
			if special == "symlink" {
				backing := filepath.Join(dir, "backing")
				mustWriteFile(t, backing, "// project\n")
				if err := os.Symlink(backing, project); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			} else if err := os.Mkdir(project, 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := hasDriverProject(dir, projects); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
				t.Fatalf("hasDriverProject() error = %v", err)
			}
		})
	}
}
