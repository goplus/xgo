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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/xgo/x/xgoprojs"
)

type driverFixture struct {
	root      string
	app       string
	project   string
	framework string
	mainFile  string
}

func newDriverFixture(t *testing.T) driverFixture {
	t.Helper()
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	project := filepath.Join(app, "game")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, filepath.Join(project, "pack"))
	mustMkdirAll(t, filepath.Join(framework, "cmd", "driver"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), `xgo 1.8

project main.foo Game example.test/framework
class *.bar Worker
pack pack index.data
driver v1 example.test/framework/cmd/driver
`)
	mustWriteFile(t, filepath.Join(framework, "cmd", "driver", "main.go"), fakeDriverSource)
	mainFile := filepath.Join(project, "main.foo")
	mustWriteFile(t, mainFile, "// fake project source\n")
	mustWriteFile(t, filepath.Join(project, "pack", "index.data"), "{}\n")
	return driverFixture{root: root, app: app, project: project, framework: framework, mainFile: mainFile}
}

func (f driverFixture) resolver(t *testing.T, flags ...string) *Resolver {
	t.Helper()
	resolver, err := NewResolver(context.Background(), f.app, flags)
	if err != nil {
		t.Fatal(err)
	}
	resolver.xgoVersion = "v99.0.0"
	return resolver
}

func resolveDriver(t *testing.T, resolver *Resolver, target xgoprojs.Proj) *Driver {
	t.Helper()
	driver, err := resolver.Resolve(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	return driver
}
func setFixtureRequiredXGo(t *testing.T, fixture driverFixture, version string) {
	t.Helper()
	goxmodPath := filepath.Join(fixture.framework, "gox.mod")
	goxmod, err := os.ReadFile(goxmodPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(goxmod), "xgo 1.8", "xgo "+version, 1)
	if updated == string(goxmod) {
		t.Fatalf("fixture gox.mod has no expected xgo directive: %s", goxmod)
	}
	if err := os.WriteFile(goxmodPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustModuleFile(t *testing.T, path, module string) {
	t.Helper()
	mustWriteFile(t, path, "module "+module+"\n\ngo 1.25\n")
}

func mustOverlay(t *testing.T, root string, replacements map[string]string) string {
	t.Helper()
	path := filepath.Join(root, "overlay.json")
	data, err := json.Marshal(overlayJSON{Replace: replacements})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, string(data))
	return path
}

func mustPolicies(t *testing.T, cwd string, cli ...string) parsedFlags {
	t.Helper()
	policy, err := preparePolicies(context.Background(), cwd, cli)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func mustGraph(t *testing.T, dir string, policy GraphPolicy) *effectiveGraph {
	t.Helper()
	graph, err := loadEffectiveGraph(context.Background(), dir, policy)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func writableModuleCache(t *testing.T) string {
	t.Helper()
	cache := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(cache, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0o700)
			}
			return os.Chmod(path, 0o600)
		})
	})
	return cache
}

func canonicalDir(t *testing.T, path string) string {
	t.Helper()
	got, err := canonicalExistingDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func canonicalFile(t *testing.T, path string) string {
	t.Helper()
	got, err := canonicalExistingFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func mustRunGo(t *testing.T, dir, goWork string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = withNeutralGOFLAGS(replaceEnv(os.Environ(), "GOWORK", goWork))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func unsetTestEnv(t *testing.T, key string) {
	t.Helper()
	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func useDriverCapableXGo(t *testing.T) {
	t.Helper()
	previous := currentXGoVersion
	currentXGoVersion = func() string { return "v99.0.0" }
	t.Cleanup(func() { currentXGoVersion = previous })
}

func environmentValue(env []string, key string) (string, bool) {
	for _, entry := range env {
		name, value, ok := strings.Cut(entry, "=")
		if ok && name == key {
			return value, true
		}
	}
	return "", false
}

const fakeDriverSource = `package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	if marker := os.Getenv("FAKE_DRIVER_MARKER"); marker != "" {
		_ = os.WriteFile(marker, []byte("started"), 0600)
	}
	if value := os.Getenv("FAKE_DRIVER_EXIT"); value != "" {
		code, _ := strconv.Atoi(value)
		os.Exit(code)
	}
	if os.Getenv("FAKE_DRIVER_BLOCK") == "1" {
		for {
			time.Sleep(time.Hour)
		}
	}
	if len(os.Args) < 3 || os.Args[1] != "xgo-driver-v1" {
		os.Exit(90)
	}
	switch os.Args[2] {
	case "run":
		for i, arg := range os.Args[3:] {
			if arg == "--" {
				fmt.Printf("run-args=%s\n", strings.Join(os.Args[i+4:], "|"))
				return
			}
		}
		os.Exit(91)
	case "build":
		var output string
		for _, arg := range os.Args[3:] {
			if strings.HasPrefix(arg, "--output=") {
				output = strings.TrimPrefix(arg, "--output=")
			}
		}
		self, err := os.Executable()
		if err != nil { panic(err) }
		data, err := os.ReadFile(self)
		if err != nil { panic(err) }
		if err := os.WriteFile(output, data, 0755); err != nil { panic(err) }
		if runtime.GOOS == "darwin" {
			if data, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", output).CombinedOutput(); err != nil {
				fmt.Fprintln(os.Stderr, string(data))
				os.Exit(93)
			}
		}
	default:
		os.Exit(92)
	}
}
`
