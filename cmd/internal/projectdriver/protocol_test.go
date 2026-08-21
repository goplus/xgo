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
	"reflect"
	"strings"
	"testing"
)

func testDriver() *Driver {
	return &Driver{
		ProjectDir:    "/project",
		ProjectFile:   "/project/main.foo",
		ModuleRoot:    "/project",
		DriverPackage: "example.test/framework/cmd/driver",
		Origin: ResolvedModule{
			Selected: ModuleRef{Path: "example.test/framework", Version: "v1.2.3"},
			Replace: &ModuleRef{
				Path: "/framework", Dir: "/framework", GoMod: "/framework/go.mod",
			},
		},
		Protocol:       "v1",
		ProjectExt:     ".foo",
		ProjectFullExt: "*.foo",
		PackDir:        "payload",
		PackIndex:      "index.json",
		GoxMod:         "/framework/gox.mod",
		GoxModSHA256:   strings.Repeat("a", 64),
		Graph: GraphPolicy{
			GoCommand: "/usr/bin/go",
			WorkDir:   "/project",
			GoWork:    "off",
			ModMode:   modModeMod,
			ModFile:   "/project/alt.mod",
		},
	}
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
		"--project-dir=/project",
		"--project-file=/project/main.foo",
		"--module-root=/project",
		"--driver-package=example.test/framework/cmd/driver",
		"--selected-path=example.test/framework",
		"--selected-version=v1.2.3",
		"--origin-main=false",
		"--replace-path=/framework",
		"--replace-version=",
		"--replace-dir=/framework",
		"--replace-gomod=/framework/go.mod",
		"--project-ext=.foo",
		"--project-full-ext=*.foo",
		"--pack-dir=payload",
		"--pack-index=index.json",
		"--declaration-file=/framework/gox.mod",
		"--declaration-sha256=" + strings.Repeat("a", 64),
		"--go-command=/usr/bin/go",
		"--graph-work-dir=/project",
		"--go-work=off",
		"--graph-flag=-mod=mod",
		"--graph-flag=-modfile=/project/alt.mod",
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
	if strings.Contains(joined, "selected-dir") || !strings.Contains(joined, "--replace-dir=/framework") {
		t.Fatalf("replacement identity not preserved:\n%s", joined)
	}
}

func TestDriverArgsBuildSelected(t *testing.T) {
	drv := testDriver()
	drv.Origin.Replace = nil
	drv.Origin.Selected.Dir = "/framework"
	drv.Origin.Selected.GoMod = "/framework/go.mod"
	got, err := driverArgs(drv, actionBuild, BuildPolicy{}, "/tmp/stage/game", "/out/game", nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"--selected-dir=/framework", "--selected-gomod=/framework/go.mod", "--output=/tmp/stage/game", "--final-output=/out/game"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "--replace-") || strings.Contains(joined, "\n--\n") {
		t.Fatalf("invalid build argv:\n%s", joined)
	}
}

func TestDriverArgsInvalid(t *testing.T) {
	drv := testDriver()
	drv.Protocol = "v2"
	if _, err := driverArgs(drv, actionRun, BuildPolicy{}, "", "", nil); err == nil {
		t.Fatal("unsupported protocol succeeded")
	}
	drv.Protocol = "v1"
	if _, err := driverArgs(drv, actionBuild, BuildPolicy{}, "", "", nil); err == nil {
		t.Fatal("build without output succeeded")
	}
}

func TestValidateArgv(t *testing.T) {
	big := strings.Repeat("x", 256<<10)
	if err := validateArgv("driver", []string{big}, nil); !errors.Is(err, ErrDriverArgvTooLarge) {
		t.Fatalf("validateArgv = %v", err)
	}
}
