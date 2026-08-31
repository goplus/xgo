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
	"errors"
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
	useDriverCapableXGo(t)
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

func TestTryRunConsumesApplicationDelimiter(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	useDriverCapableXGo(t)
	var stdout bytes.Buffer
	result, err := TryRun(
		context.Background(),
		fixture.app,
		&xgoprojs.DirProj{Dir: fixture.project},
		nil,
		[]string{"--", "argument", "--"},
		Streams{Stdout: &stdout},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Handled || result.Status.Code != 0 {
		t.Fatalf("run = %#v", result)
	}
	if got := strings.TrimSpace(stdout.String()); got != "run-args=argument|--" {
		t.Fatalf("application args = %q", got)
	}
}

func TestDispatchRejectsMixedTargetsBeforeDriver(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	useDriverCapableXGo(t)
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

func TestDispatchDeferredPolicyLegacyFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/plain\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main() {}\n")
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOWORK", "off")
	target := &xgoprojs.DirProj{Dir: dir}
	for _, action := range []string{"run", "build", "install"} {
		t.Run(action, func(t *testing.T) {
			var result DispatchResult
			var err error
			switch action {
			case "run":
				result, err = TryRun(context.Background(), dir, target, []string{"-x=invalid"}, nil, Streams{})
			case "build":
				result, err = TryBuild(context.Background(), dir, target, []string{"-x=invalid"}, filepath.Join(dir, "plain"), Streams{})
			case "install":
				result, err = TryInstall(context.Background(), dir, []xgoprojs.Proj{target}, []string{"-x=invalid"}, Streams{})
			}
			if err != nil || result.Handled {
				t.Fatalf("%s = %#v, %v; want legacy fallback", action, result, err)
			}
		})
	}
}

func TestDispatchRejectsDeferredPolicyBeforeDriverActions(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	useDriverCapableXGo(t)
	for _, action := range []string{"run", "build", "install"} {
		for _, source := range []string{"cli", "ambient"} {
			t.Run(action+"/"+source, func(t *testing.T) {
				fixture := newDriverFixture(t)
				t.Setenv("GOENV", "off")
				marker := filepath.Join(fixture.root, "driver-started")
				t.Setenv("FAKE_DRIVER_MARKER", marker)
				badFlag := "-x=invalid"
				var flags []string
				if source == "ambient" {
					t.Setenv("GOFLAGS", badFlag)
				} else {
					t.Setenv("GOFLAGS", "")
					flags = []string{badFlag}
				}
				target := &xgoprojs.DirProj{Dir: fixture.project}
				output := filepath.Join(fixture.root, "must-not-exist", "game")
				var err error
				switch action {
				case "run":
					_, err = TryRun(context.Background(), fixture.app, target, flags, nil, Streams{})
				case "build":
					_, err = TryBuild(context.Background(), fixture.app, target, flags, output, Streams{})
				case "install":
					t.Setenv("GOBIN", filepath.Dir(output))
					_, err = TryInstall(context.Background(), fixture.app, []xgoprojs.Proj{target}, flags, Streams{})
				}
				if err == nil || !strings.Contains(err.Error(), badFlag) {
					t.Fatalf("%s error = %v, want rejection of %q", action, err, badFlag)
				}
				if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("driver started before rejection: %v", statErr)
				}
				if _, statErr := os.Stat(filepath.Dir(output)); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("output created before rejection: %v", statErr)
				}
			})
		}
	}
}
