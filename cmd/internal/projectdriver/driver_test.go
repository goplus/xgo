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
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/goplus/xgo/x/xgoprojs"
)

func TestDriverRunAndBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t, "-trimpath=true", "-buildvcs=false")
	drv := resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})
	var stdout, stderr bytes.Buffer
	status, err := resolver.Run(context.Background(), drv, []string{"", "a b", "--"}, Streams{Stdout: &stdout, Stderr: &stderr})
	if err != nil || status.Code != 0 || status.Signaled {
		t.Fatalf("run = %#v, %v, stderr=%s", status, err, &stderr)
	}
	if got := strings.TrimSpace(stdout.String()); got != "run-args=|a b|--" {
		t.Fatalf("stdout = %q", got)
	}

	final := filepath.Join(fixture.root, "bin", "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(final), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	status, gotFinal, err := resolver.Build(context.Background(), drv, final, Streams{Stdout: &stdout, Stderr: &stderr})
	if err != nil || status.Code != 0 || gotFinal != final {
		t.Fatalf("build = %#v, %q, %v, stderr=%s", status, gotFinal, err, &stderr)
	}
	info, err := os.Stat(final)
	if err != nil || info.Size() <= int64(len("old")) {
		t.Fatalf("artifact = %#v, %v", info, err)
	}
}

func TestDriverBuildFailurePreservesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t)
	drv := resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})
	final := filepath.Join(fixture.root, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DRIVER_EXIT", "42")
	status, _, err := resolver.Build(context.Background(), drv, final, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	if err != nil || status.Code != 42 {
		t.Fatalf("build failure = %#v, %v", status, err)
	}
	data, readErr := os.ReadFile(final)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("old output changed: %q, %v", data, readErr)
	}
	assertNoOutputWorkDirs(t, filepath.Dir(final))
}

func TestDriverBuildCancellationPreservesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t)
	drv := resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})
	final := filepath.Join(fixture.root, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(fixture.root, "driver-started")
	t.Setenv("FAKE_DRIVER_MARKER", marker)
	t.Setenv("FAKE_DRIVER_BLOCK", "1")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = resolver.Build(ctx, drv, final, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	}()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatal("driver did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("canceled driver did not exit")
	}
	data, readErr := os.ReadFile(final)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("canceled build changed old output: %q, %v", data, readErr)
	}
	assertNoOutputWorkDirs(t, filepath.Dir(final))
}

func assertNoOutputWorkDirs(t *testing.T, parent string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(parent, ".xgo-driver-output-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("driver output work directories remain: %v", matches)
	}
}

func TestDriverInstallUsesEffectiveGOBIN(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t)
	drv := resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})
	bin := filepath.Join(fixture.root, "custom-bin")
	t.Setenv("GOBIN", bin)
	status, final, err := resolver.Install(context.Background(), drv, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	if err != nil || status.Code != 0 {
		t.Fatalf("install = %#v, %q, %v", status, final, err)
	}
	want := filepath.Join(bin, executableName("game"))
	if final != want {
		t.Fatalf("install output = %q, want %q", final, want)
	}
	if info, err := os.Stat(final); err != nil || info.Size() == 0 {
		t.Fatalf("installed artifact = %#v, %v", info, err)
	}
}

func TestDriverInstallValidatesPolicyBeforeCreatingGOBIN(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	fixture := newDriverFixture(t)
	resolver := fixture.resolver(t, "-tags=unsupported")
	drv := resolveDriver(t, resolver, &xgoprojs.DirProj{Dir: fixture.project})
	bin := filepath.Join(fixture.root, "must-not-exist", "bin")
	t.Setenv("GOBIN", bin)
	status, final, err := resolver.Install(context.Background(), drv, Streams{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)})
	if err == nil || !strings.Contains(err.Error(), "does not support flag -tags") {
		t.Fatalf("Install() = %#v, %q, %v; want unsupported -tags error", status, final, err)
	}
	if _, statErr := os.Stat(bin); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("GOBIN was created before policy validation: %v", statErr)
	}
}

func TestHostBuildEnvironmentPreservesCGO(t *testing.T) {
	env := hostBuildEnvironment([]string{
		"GOOS=target-os",
		"GOARCH=target-arch",
		"CGO_ENABLED=1",
	}, "off")
	if got, _ := environmentValue(env, "GOOS"); got != runtime.GOOS {
		t.Fatalf("GOOS = %q, want host %q", got, runtime.GOOS)
	}
	if got, _ := environmentValue(env, "GOARCH"); got != runtime.GOARCH {
		t.Fatalf("GOARCH = %q, want host %q", got, runtime.GOARCH)
	}
	if got, ok := environmentValue(env, "CGO_ENABLED"); !ok || got != "1" {
		t.Fatalf("CGO_ENABLED = %q, %t; want inherited value 1", got, ok)
	}

	env = hostBuildEnvironment([]string{"PATH=/bin"}, "off")
	if got, ok := environmentValue(env, "CGO_ENABLED"); ok {
		t.Fatalf("CGO_ENABLED = %q; host default should remain unset", got)
	}
}
