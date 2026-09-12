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
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/mod/xgomod"
)

func testDriver() *Driver {
	projectDir := testAbsolutePath("project")
	frameworkDir := testAbsolutePath("framework")
	return &Driver{
		ProjectDir:    projectDir,
		ProjectFile:   filepath.Join(projectDir, "main.foo"),
		ModuleRoot:    projectDir,
		DriverPackage: "example.test/framework/cmd/driver",
		Origin: ResolvedModule{
			Selected: ModuleRef{Path: "example.test/framework", Version: "v1.2.3"},
			Replace: &ModuleRef{
				Path: frameworkDir, Dir: frameworkDir, GoMod: filepath.Join(frameworkDir, "go.mod"),
			},
		},
		ProjectExt:     ".foo",
		ProjectFullExt: "*.foo",
		PackDir:        "payload",
		PackIndex:      "index.json",
		Declaration:    xgomod.FileIdentity{Path: filepath.Join(frameworkDir, "gox.mod"), SHA256: strings.Repeat("a", 64)},
		TargetModFile: xgomod.FileIdentity{
			Path: filepath.Join(projectDir, "alt.mod"), SHA256: strings.Repeat("b", 64),
		},
		Graph: GraphPolicy{
			GoCommand: testAbsolutePath("go", "bin", executableName("go")),
			WorkDir:   projectDir,
			GoWork:    "off",
			ModMode:   modModeMod,
			ModFile:   filepath.Join(projectDir, "alt.mod"),
		},
	}
}

func testAbsolutePath(elem ...string) string {
	root := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	return filepath.Join(append([]string{root}, elem...)...)
}

func TestDriverArgsRun(t *testing.T) {
	drv := testDriver()
	got, err := driverArgs(drv, actionRun, BuildPolicy{TrimPath: true, Verbose: true, Trace: true, KeepWork: true}, "", "", []string{"", "a b", "--"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"xgo-driver-v1",
		"run",
		"--project-dir=" + drv.ProjectDir,
		"--project-file=" + drv.ProjectFile,
		"--module-root=" + drv.ModuleRoot,
		"--driver-package=example.test/framework/cmd/driver",
		"--selected-path=example.test/framework",
		"--selected-version=v1.2.3",
		"--origin-main=false",
		"--replace-path=" + drv.Origin.Replace.Path,
		"--replace-version=",
		"--replace-dir=" + drv.Origin.Replace.Dir,
		"--replace-gomod=" + drv.Origin.Replace.GoMod,
		"--project-ext=.foo",
		"--project-full-ext=*.foo",
		"--pack-dir=payload",
		"--pack-index=index.json",
		"--declaration-file=" + drv.Declaration.Path,
		"--declaration-sha256=" + strings.Repeat("a", 64),
		"--target-modfile=" + drv.TargetModFile.Path,
		"--target-modfile-sha256=" + strings.Repeat("b", 64),
		"--go-command=" + drv.Graph.GoCommand,
		"--graph-work-dir=" + drv.Graph.WorkDir,
		"--go-work=off",
		"--graph-flag=-mod=mod",
		"--graph-flag=-modfile=" + drv.Graph.ModFile,
		"--build-flag=-v=true",
		"--build-flag=-x=true",
		"--build-flag=-work=true",
		"--build-flag=-trimpath=true",
		"--", "", "a b", "--",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %#v, want %#v", got, want)
	}
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "selected-dir") || !strings.Contains(joined, "--replace-dir="+drv.Origin.Replace.Dir) {
		t.Fatalf("replacement identity not preserved:\n%s", joined)
	}
}

func TestDriverArgsBuildSelected(t *testing.T) {
	drv := testDriver()
	drv.Origin.Replace = nil
	drv.Origin.Selected.Dir = testAbsolutePath("framework")
	drv.Origin.Selected.GoMod = filepath.Join(drv.Origin.Selected.Dir, "go.mod")
	staging := testAbsolutePath("tmp", "stage", "game")
	final := testAbsolutePath("out", "game")
	got, err := driverArgs(drv, actionBuild, BuildPolicy{}, staging, final, nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"--selected-dir=" + drv.Origin.Selected.Dir, "--selected-gomod=" + drv.Origin.Selected.GoMod, "--output=" + staging, "--final-output=" + final} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "--replace-") || strings.Contains(joined, "\n--\n") {
		t.Fatalf("invalid build argv:\n%s", joined)
	}
}

func TestDriverArgsRejectsBuildWithoutOutput(t *testing.T) {
	if _, err := driverArgs(testDriver(), actionBuild, BuildPolicy{}, "", "", nil); err == nil {
		t.Fatal("build without output succeeded")
	}
}

func TestValidateArgv(t *testing.T) {
	big := strings.Repeat("x", 256<<10)
	if err := validateArgv("driver", []string{big}, nil); !errors.Is(err, ErrDriverArgvTooLarge) {
		t.Fatalf("validateArgv = %v", err)
	}
}
