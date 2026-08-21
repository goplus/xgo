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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestGraphFileViewReadsOverlayReplacement(t *testing.T) {
	root := t.TempDir()
	logical := filepath.Join(root, "go.mod")
	replacement := filepath.Join(root, "draft.mod")
	mustWriteFile(t, logical, "module example.test/plain\n\ngo 1.25\n")
	mustWriteFile(t, replacement, "module example.test/overlay\n\ngo 1.25\n")
	overlay := mustOverlay(t, root, map[string]string{logical: replacement})
	view, err := newGraphFileView(GraphPolicy{Overlay: overlay}, root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := view.readFile(logical)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "module example.test/overlay") {
		t.Fatalf("overlay read = %q", got)
	}
}

func TestGraphFileViewSynthesizesOverlayParentDirectories(t *testing.T) {
	root := t.TempDir()
	actual := filepath.Join(root, "actual.foo")
	logical := filepath.Join(root, "virtual", "nested", "main.foo")
	mustWriteFile(t, actual, "overlay project\n")
	overlay := mustOverlay(t, root, map[string]string{logical: actual})
	view, err := newGraphFileView(GraphPolicy{Overlay: overlay}, root)
	if err != nil {
		t.Fatal(err)
	}
	names, err := view.regularFileNames(filepath.Join(root, "virtual", "nested"))
	if err != nil {
		t.Fatalf("regularFileNames() = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"main.foo"}) {
		t.Fatalf("regularFileNames() = %#v, want [main.foo]", names)
	}
	dir, err := view.canonicalDir(filepath.Join(root, "virtual", "nested"))
	if err != nil || dir != filepath.Join(root, "virtual", "nested") {
		t.Fatalf("canonicalDir() = %q, %v", dir, err)
	}
}

func TestGraphFileViewDeletedParentAllowsAddedChild(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	actual := filepath.Join(root, "replacement.foo")
	logical := filepath.Join(parent, "child", "main.foo")
	mustMkdirAll(t, parent)
	mustWriteFile(t, filepath.Join(parent, "old.foo"), "old\n")
	mustWriteFile(t, actual, "new\n")
	overlay := mustOverlay(t, root, map[string]string{parent: "", logical: actual})
	view, err := newGraphFileView(GraphPolicy{Overlay: overlay}, root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := view.readFile(logical)
	if err != nil || string(got) != "new\n" {
		t.Fatalf("readFile() = %q, %v", got, err)
	}
	names, err := view.regularFileNames(filepath.Join(parent, "child"))
	if err != nil {
		t.Fatalf("regularFileNames() = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"main.foo"}) {
		t.Fatalf("regularFileNames() = %#v, want [main.foo]", names)
	}
}

func TestGraphFileViewDoesNotInspectReplacementDestinations(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	mustMkdirAll(t, project)
	missing := filepath.Join(root, "missing.foo")
	directory := filepath.Join(root, "replacement-dir")
	mustMkdirAll(t, directory)
	replacements := map[string]string{
		filepath.Join(project, "missing.foo"):   missing,
		filepath.Join(project, "directory.foo"): directory,
	}
	if runtime.GOOS != "windows" {
		target := filepath.Join(root, "target.foo")
		mustWriteFile(t, target, "target\n")
		symlink := filepath.Join(root, "symlink.foo")
		if err := os.Symlink(target, symlink); err != nil {
			t.Fatal(err)
		}
		replacements[filepath.Join(project, "symlink.foo")] = symlink
	}
	overlay := mustOverlay(t, root, replacements)
	view, err := newGraphFileView(GraphPolicy{Overlay: overlay}, root)
	if err != nil {
		t.Fatal(err)
	}
	names, err := view.regularFileNames(project)
	if err != nil {
		t.Fatalf("regularFileNames() = %v", err)
	}
	want := []string{"directory.foo", "missing.foo"}
	if runtime.GOOS != "windows" {
		want = append(want, "symlink.foo")
	}
	sort.Strings(want)
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("regularFileNames() = %#v, want %#v", names, want)
	}
}

func TestGraphFileViewAnchorsRelativeDirectoryNames(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "project", "game"))
	view, err := newGraphFileView(GraphPolicy{}, root)
	if err != nil {
		t.Fatal(err)
	}
	names, err := view.directoryNames("project")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"game"}) {
		t.Fatalf("directoryNames() = %#v, want [game]", names)
	}
}

func TestGraphEnvironmentClearsAmbientGOFLAGS(t *testing.T) {
	base := []string{"GOFLAGS=-tags=ambient", "GOWORK=/old/work", "PATH=/bin"}
	env := graphEnvironment(base, "off")
	if got, ok := environmentValue(env, "GOFLAGS"); !ok || got != "" {
		t.Fatalf("GOFLAGS = %q, %t; want cleared", got, ok)
	}
	if got, ok := environmentValue(env, "GOWORK"); !ok || got != "off" {
		t.Fatalf("GOWORK = %q, %t; want off", got, ok)
	}
}

func TestReadTargetModFileViewAllowsOverlayOnlyModfile(t *testing.T) {
	root := t.TempDir()
	logical := filepath.Join(root, "go.mod")
	actual := filepath.Join(root, "overlay.mod")
	mustWriteFile(t, actual, "module example.test/overlay\n\ngo 1.25\n")
	overlay := mustOverlay(t, root, map[string]string{logical: actual})
	view, err := newGraphFileView(GraphPolicy{Overlay: overlay}, root)
	if err != nil {
		t.Fatal(err)
	}
	identity, classes, err := readTargetModFileView(logical, view)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Path != logical || len(identity.SHA256) != 64 || len(classes) != 0 {
		t.Fatalf("overlay-only modfile = %#v, classes %#v", identity, classes)
	}
}

func TestLoadEffectiveGraphLocalReplace(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, filepath.Join(app, "game"))
	mustMkdirAll(t, filepath.Join(framework, "cmd", "driver"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	mustModuleFile(t, filepath.Join(framework, "go.mod"), "example.test/framework")
	mustWriteFile(t, filepath.Join(framework, "cmd", "driver", "main.go"), "package main\nfunc main() {}\n")

	policy := mustPolicies(t, app)
	graph := mustGraph(t, filepath.Join(app, "game"), policy.graph)
	if got, want := graph.Target.Selected.Path, "example.test/app"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
	origin := graph.Modules["example.test/framework"]
	if !reflect.DeepEqual(graph.ClassModules, []ResolvedModule{origin}) {
		t.Fatalf("resolved class modules = %#v", graph.ClassModules)
	}
	if origin.Selected.Version != "v1.2.3" || origin.Selected.Dir != "" || origin.Replace == nil {
		t.Fatalf("replacement was flattened: %#v", origin)
	}
	framework, err := canonicalExistingDir(framework)
	if err != nil {
		t.Fatal(err)
	}
	if origin.Replace.Dir != framework || origin.Replace.GoMod != filepath.Join(framework, "go.mod") {
		t.Fatalf("replacement source = %#v", origin.Replace)
	}
	dir, module, err := resolvePackageDirectory(context.Background(), graph, "example.test/framework/cmd/driver", app, policy.graph)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(framework, "cmd", "driver") || module.Selected.Path != "example.test/framework" {
		t.Fatalf("resolved package = %q, %#v", dir, module)
	}
}

func TestResolvePackageDirectoryHonorsNestedModuleBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	app := t.TempDir()
	nested := filepath.Join(app, "nested")
	mustMkdirAll(t, nested)
	mustModuleFile(t, filepath.Join(app, "go.mod"), "example.test/app")
	mustModuleFile(t, filepath.Join(nested, "go.mod"), "example.test/nested")
	mustWriteFile(t, filepath.Join(nested, "main.go"), "package main\n")
	policy := mustPolicies(t, app)
	graph := mustGraph(t, app, policy.graph)
	if _, _, err := resolvePackageDirectory(context.Background(), graph, "example.test/app/nested", app, policy.graph); err == nil {
		t.Fatal("package path crossing a nested module boundary resolved successfully")
	}
}

func TestSanitizeGraphFlagsPropagatesFilesystemErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-referential symlink setup is not portable to Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "loop.mod")
	if err := os.Symlink("loop.mod", path); err != nil {
		t.Fatal(err)
	}
	policy := parsedFlags{graph: GraphPolicy{ModFile: path}}
	if _, err := sanitizeGraphFlags(policy); err == nil {
		t.Fatal("graph input I/O failure was treated as a missing file")
	}
}

func TestLoadEffectiveGraphModfile(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	dir := t.TempDir()
	mustModuleFile(t, filepath.Join(dir, "go.mod"), "example.test/app")
	mustModuleFile(t, filepath.Join(dir, "driver.mod"), "example.test/app")
	policy := mustPolicies(t, dir, "-modfile=driver.mod")
	graph := mustGraph(t, dir, policy.graph)
	wantModfile, err := canonicalExistingFile(filepath.Join(dir, "driver.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if graph.TargetModFile.Path != wantModfile {
		t.Fatalf("target modfile = %q", graph.TargetModFile.Path)
	}
	owner, err := goModForDir(context.Background(), policy.graph, dir)
	if err != nil {
		t.Fatal(err)
	}
	wantOwner, err := canonicalExistingFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if owner != wantOwner {
		t.Fatalf("owning go.mod = %q, want %q", owner, wantOwner)
	}
}

func TestReadTargetModFileClassMarkerBoundaries(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	mustWriteFile(t, goMod, `module example.test/app

go 1.25

require (
	example.test/second v1.0.0 //gop:class payload
	example.test/classroom v1.0.0 //xgo:classroom
	example.test/first v1.0.0 // xgo:class
)
`)
	identity, paths, err := readTargetModFile(goMod)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"example.test/second", "example.test/first"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("class paths = %#v, want %#v", paths, want)
	}
	if identity.Path != canonicalFile(t, goMod) || len(identity.SHA256) != 64 {
		t.Fatalf("identity = %#v", identity)
	}
}

func TestToXGoGraphPreservesResolvedClassModuleOrder(t *testing.T) {
	second := ResolvedModule{Selected: ModuleRef{Path: "example.test/second"}}
	first := ResolvedModule{Selected: ModuleRef{Path: "example.test/first"}}
	graph := &effectiveGraph{
		Target:       ResolvedModule{Selected: ModuleRef{Path: "example.test/app"}},
		Modules:      map[string]ResolvedModule{},
		ClassModules: []ResolvedModule{second, first},
	}
	got := toXGoGraph(graph)
	if len(got.ClassModules) != 2 || got.ClassModules[0].Selected.Path != "example.test/second" || got.ClassModules[1].Selected.Path != "example.test/first" {
		t.Fatalf("ClassModules = %#v", got.ClassModules)
	}
}

func TestRetargetEffectiveGraphPreservesClassModuleOrder(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	framework := filepath.Join(root, "framework")
	firstDir := filepath.Join(root, "first")
	secondDir := filepath.Join(root, "second")
	for _, dir := range []string{app, framework, firstDir, secondDir} {
		mustMkdirAll(t, dir)
	}
	appGoMod := filepath.Join(app, "go.mod")
	frameworkGoMod := filepath.Join(framework, "go.mod")
	firstGoMod := filepath.Join(firstDir, "go.mod")
	secondGoMod := filepath.Join(secondDir, "go.mod")
	mustModuleFile(t, appGoMod, "example.test/app")
	mustWriteFile(t, frameworkGoMod, `module example.test/framework

go 1.25

require (
	example.test/second v1.0.0 //xgo:class
	example.test/first v1.0.0 //xgo:class
)
`)
	mustModuleFile(t, firstGoMod, "example.test/first")
	mustModuleFile(t, secondGoMod, "example.test/second")

	appModule := ResolvedModule{Selected: ModuleRef{Path: "example.test/app", Dir: app, GoMod: appGoMod}, Main: true}
	frameworkModule := ResolvedModule{Selected: ModuleRef{Path: "example.test/framework", Dir: framework, GoMod: frameworkGoMod}}
	firstModule := ResolvedModule{Selected: ModuleRef{Path: "example.test/first", Dir: firstDir, GoMod: firstGoMod}}
	secondModule := ResolvedModule{Selected: ModuleRef{Path: "example.test/second", Dir: secondDir, GoMod: secondGoMod}}
	graph := &effectiveGraph{
		Target: appModule,
		Modules: map[string]ResolvedModule{
			"example.test/app":       appModule,
			"example.test/framework": frameworkModule,
			"example.test/first":     firstModule,
			"example.test/second":    secondModule,
		},
		TargetModFile: fileIdentity{Path: canonicalFile(t, appGoMod)},
	}

	retargeted, err := retargetEffectiveGraph(graph, frameworkModule)
	if err != nil {
		t.Fatal(err)
	}
	want := []ResolvedModule{secondModule, firstModule}
	if !reflect.DeepEqual(retargeted.ClassModules, want) {
		t.Fatalf("resolved class modules = %#v, want %#v", retargeted.ClassModules, want)
	}
	if retargeted.TargetModFile.Path != canonicalFile(t, frameworkGoMod) {
		t.Fatalf("target modfile = %q", retargeted.TargetModFile.Path)
	}
}

func TestPreparePoliciesIgnoresXGoGoCmd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	t.Setenv("XGO_GOCMD", filepath.Join(t.TempDir(), "does-not-exist"))
	t.Setenv("GOWORK", "off")
	policy := mustPolicies(t, t.TempDir())
	if filepath.Base(policy.graph.GoCommand) != "go" {
		t.Fatalf("Go command = %q", policy.graph.GoCommand)
	}
}

func TestNormalizeListedModuleUsesResolvedValidation(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "source")
	mustMkdirAll(t, dir)
	goMod := filepath.Join(root, "external.mod")
	mustWriteFile(t, goMod, "module example.test/mod\n")
	_, err := normalizeListedModule(goListModule{
		Path: "example.test/mod", Version: "v1.2.3", Dir: dir, GoMod: goMod,
	})
	if err == nil || !strings.Contains(err.Error(), "matching Go module-cache metadata") {
		t.Fatalf("validation error = %v", err)
	}
}
