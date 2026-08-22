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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/xgo/x/xgoprojs"
)

func TestTryDispatchRequiresTargets(t *testing.T) {
	if _, err := TryRun(context.Background(), "", nil, nil, nil, Streams{}); err == nil {
		t.Fatal("TryRun accepted a nil target")
	}
	if _, err := TryBuild(context.Background(), "", nil, nil, "", Streams{}); err == nil {
		t.Fatal("TryBuild accepted a nil target")
	}
	if _, err := TryInstall(context.Background(), "", nil, nil, Streams{}); err == nil {
		t.Fatal("TryInstall accepted zero targets")
	}
	if _, err := TryInstall(context.Background(), "", []xgoprojs.Proj{nil}, nil, Streams{}); err == nil {
		t.Fatal("TryInstall accepted a nil target entry")
	}
}

func TestTryBuildKeepWorkAllowsNilStderr(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	setFixtureRequiredXGo(t, fixture, "1.0")
	output := filepath.Join(fixture.root, "game")
	var stdout bytes.Buffer
	result, err := TryBuild(context.Background(), fixture.app, &xgoprojs.DirProj{Dir: fixture.project}, []string{"-work=true"}, output, Streams{Stdout: &stdout})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Handled || result.Status.Code != 0 || result.Output != output {
		t.Fatalf("build = %#v", result)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("build output = %#v, %v", info, err)
	}
}

func TestDispatchRejectsMixedTargetsBeforeDriver(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	setFixtureRequiredXGo(t, fixture, "1.0")
	plain := filepath.Join(fixture.app, "plain")
	mustMkdirAll(t, plain)
	mustWriteFile(t, filepath.Join(plain, "main.go"), "package main\nfunc main() {}\n")
	marker := filepath.Join(fixture.root, "driver-started")
	t.Setenv("FAKE_DRIVER_MARKER", marker)
	result, err := TryInstall(context.Background(), "", []xgoprojs.Proj{&xgoprojs.DirProj{Dir: plain}, &xgoprojs.DirProj{Dir: fixture.project}}, nil, Streams{})
	if err == nil || !strings.Contains(err.Error(), "exactly one target") {
		t.Fatalf("dispatch = %#v, %v", result, err)
	}
	if result.Handled {
		t.Fatalf("mixed dispatch was handled: %#v", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("driver started before target validation: %v", err)
	}
}
