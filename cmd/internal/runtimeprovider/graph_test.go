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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestLoadEffectiveGraphLocalReplace(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, filepath.Join(app, "game"))
	mustMkdirAll(t, filepath.Join(framework, "cmd", "provider"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "cmd", "provider", "main.go"), "package main\nfunc main() {}\n")

	policy, err := preparePolicies(context.Background(), app, nil)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := loadEffectiveGraph(context.Background(), filepath.Join(app, "game"), policy.graph)
	if err != nil {
		t.Fatal(err)
	}
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
	framework, err = canonicalExistingDir(framework)
	if err != nil {
		t.Fatal(err)
	}
	if origin.Replace.Dir != framework || origin.Replace.GoMod != filepath.Join(framework, "go.mod") {
		t.Fatalf("replacement source = %#v", origin.Replace)
	}
	dir, module, err := resolvePackageDirectory(graph, "example.test/framework/cmd/provider")
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(framework, "cmd", "provider") || module.Selected.Path != "example.test/framework" {
		t.Fatalf("resolved package = %q, %#v", dir, module)
	}
}

func TestLoadEffectiveGraphModfile(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dir, "runtime.mod"), "module example.test/app\n\ngo 1.25\n")
	policy, err := preparePolicies(context.Background(), dir, []string{"-modfile=runtime.mod"})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := loadEffectiveGraph(context.Background(), dir, policy.graph)
	if err != nil {
		t.Fatal(err)
	}
	wantModfile, err := canonicalExistingFile(filepath.Join(dir, "runtime.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if graph.TargetModFile.Path != wantModfile {
		t.Fatalf("target modfile = %q", graph.TargetModFile.Path)
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
	mustWriteFile(t, appGoMod, "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, frameworkGoMod, `module example.test/framework

go 1.25

require (
	example.test/second v1.0.0 //xgo:class
	example.test/first v1.0.0 //xgo:class
)
`)
	mustWriteFile(t, firstGoMod, "module example.test/first\n\ngo 1.25\n")
	mustWriteFile(t, secondGoMod, "module example.test/second\n\ngo 1.25\n")

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
	policy, err := preparePolicies(context.Background(), t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
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

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
