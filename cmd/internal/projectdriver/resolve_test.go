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

	"github.com/goplus/xgo/x/xgoprojs"
)

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

func TestDefaultExecutableNameSemanticImportVersion(t *testing.T) {
	for path, want := range map[string]string{
		"example.test/tool/v1":  "v1",
		"example.test/tool/v10": "tool",
		"example.test/tool/v19": "tool",
		"example.test/tool/v02": "v02",
	} {
		t.Run(path, func(t *testing.T) {
			if got := defaultExecutableName(t.TempDir(), path); got != want {
				t.Fatalf("defaultExecutableName() = %q, want %q", got, want)
			}
		})
	}
}

func TestResolverCWDAnchorsRelativeTargets(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	file := filepath.Join(project, "main.foo")
	mustMkdirAll(t, project)
	mustWriteFile(t, file, "project\n")
	project = canonicalDir(t, project)
	file = canonicalFile(t, file)
	resolver := &Resolver{cwd: root}

	directory, err := resolver.classifyDirectoryTarget(context.Background(), &xgoprojs.DirProj{Dir: "project"})
	if err != nil || directory.projectDir != project {
		t.Fatalf("relative directory = %q, %v", directory.projectDir, err)
	}
	files, err := resolver.classifyFileTarget(filepath.Join("project", "main.foo"))
	if err != nil || files.expectedFile != file {
		t.Fatalf("relative file = %q, %v", files.expectedFile, err)
	}
}

func TestResolveDriverTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	goMod := canonicalFile(t, filepath.Join(fixture.app, "go.mod"))
	goModData, err := os.ReadFile(goMod)
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]xgoprojs.Proj{
		"directory": &xgoprojs.DirProj{Dir: fixture.project},
		"file":      &xgoprojs.FilesProj{Files: []string{fixture.mainFile}},
		"package":   &xgoprojs.PkgPathProj{Path: "example.test/app/game"},
	} {
		t.Run(name, func(t *testing.T) {
			drv := resolveDriver(t, fixture.resolver(t), target)
			if drv.ProjectDir != canonicalDir(t, fixture.project) || drv.ProjectFile != canonicalFile(t, fixture.mainFile) {
				t.Fatalf("project identity = %q, %q", drv.ProjectDir, drv.ProjectFile)
			}
			if drv.DriverPackage != "example.test/framework/cmd/driver" {
				t.Fatalf("driver = %#v", drv)
			}
			if drv.Origin.Selected.Path != "example.test/framework" || drv.Origin.Selected.Version != "v1.2.3" || drv.Origin.Replace == nil {
				t.Fatalf("origin = %#v", drv.Origin)
			}
			if drv.Origin.Selected.Dir != "" || drv.Origin.Replace.Path != canonicalDir(t, fixture.framework) {
				t.Fatalf("replacement identity was flattened: %#v", drv.Origin)
			}
			if drv.PackDir != "pack" || drv.PackIndex != "index.data" || len(drv.Declaration.SHA256) != 64 {
				t.Fatalf("metadata = %#v", drv)
			}
			if drv.TargetModFile.Path != goMod || drv.TargetModFile.SHA256 != sha256Bytes(goModData) {
				t.Fatalf("target modfile = %#v", drv.TargetModFile)
			}
		})
	}
}

func TestResolveRejectsAnyNestedDriverDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t)
	target := &xgoprojs.DirProj{Dir: fixture.project}
	t.Setenv(driverGuardEnv, "1")
	if _, err := resolver.Resolve(context.Background(), target); !errors.Is(err, ErrDriverRecursive) {
		t.Fatalf("Resolve() = %v, want ErrDriverRecursive", err)
	}
}

func TestResolvePackageTargetKeepsCallerGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	dependencyProject := filepath.Join(fixture.framework, "example")
	mustMkdirAll(t, filepath.Join(dependencyProject, "pack"))
	mustWriteFile(t, filepath.Join(dependencyProject, "main.foo"), "// dependency project\n")
	mustWriteFile(t, filepath.Join(dependencyProject, "pack", "index.data"), "{}\n")

	resolver := fixture.resolver(t)
	drv := resolveDriver(t, resolver, &xgoprojs.PkgPathProj{Path: "example.test/framework/example"})
	if drv.ModuleRoot != canonicalDir(t, fixture.framework) || drv.Graph.WorkDir != canonicalDir(t, fixture.app) {
		t.Fatalf("graph roots = module %q, work %q", drv.ModuleRoot, drv.Graph.WorkDir)
	}
	if drv.Origin.Main || drv.Origin.Replace == nil || drv.Origin.Selected.Version != "v1.2.3" {
		t.Fatalf("caller replacement graph was lost: %#v", drv.Origin)
	}
	if err := validateDriver(context.Background(), drv); err != nil {
		t.Fatalf("driver validation switched graphs: %v", err)
	}
}

func TestResolveXGoOnlyPackageTargetWithModfile(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	data, err := os.ReadFile(filepath.Join(fixture.app, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(fixture.app, "driver.mod"), string(data))
	resolver := fixture.resolver(t, "-modfile=driver.mod")
	if _, err := resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/game"}); err != nil {
		t.Fatal(err)
	}
}
