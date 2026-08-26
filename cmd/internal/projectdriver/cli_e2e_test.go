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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCLIEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the XGo command and fake driver")
	}
	buildGoWork := os.Getenv("GOWORK")
	modDir := protocolModuleDir(t, buildGoWork)
	fixture := newDriverFixture(t)
	configureSharedProtocolDriver(t, fixture, modDir)
	goxmod := filepath.Join(fixture.framework, "gox.mod")
	metadata, err := os.ReadFile(goxmod)
	if err != nil {
		t.Fatal(err)
	}
	metadata = []byte(strings.Replace(string(metadata), "pack pack index.data\n", "", 1))
	if err := os.WriteFile(goxmod, metadata, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DRIVER_EXPECT_NO_PACK", "1")
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	xgo := filepath.Join(t.TempDir(), executableName("xgo"))
	build := exec.Command("go", "build", "-o", xgo, "./cmd/xgo")
	build.Dir = repo
	build.Env = replaceEnv(os.Environ(), "GOWORK", buildGoWork)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build XGo: %v\n%s", err, output)
	}

	run := cliCommand(xgo, repo, fixture.app, "run", "./game", "--", "", "a b", "--")
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("xgo run: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "run-args=|a b|--") {
		t.Fatalf("xgo run output = %q", output)
	}
	assertNoAutogen(t, fixture.project)

	artifact := filepath.Join(fixture.root, executableName("built-game"))
	command := cliCommand(xgo, repo, fixture.app, "build", "-o", artifact, "./game")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("xgo build: %v\n%s", err, output)
	}
	if info, err := os.Stat(artifact); err != nil || info.Size() == 0 {
		t.Fatalf("built artifact = %#v, %v", info, err)
	}
	assertNoAutogen(t, fixture.project)

	bin := filepath.Join(fixture.root, "install-bin")
	install := cliCommand(xgo, repo, fixture.app, "install", "./game")
	install.Env = append(install.Env, "GOBIN="+bin)
	if output, err := install.CombinedOutput(); err != nil {
		t.Fatalf("xgo install: %v\n%s", err, output)
	}
	installed := filepath.Join(bin, executableName("game"))
	if info, err := os.Stat(installed); err != nil || info.Size() == 0 {
		t.Fatalf("installed artifact = %#v, %v", info, err)
	}

	unsupported := []struct {
		name string
		args []string
		flag string
	}{
		{name: "run debug", args: []string{"run", "-debug", "./game"}, flag: "-debug"},
		{name: "run quiet", args: []string{"run", "-quiet", "./game"}, flag: "-quiet"},
		{name: "build debug", args: []string{"build", "-debug", "-o", filepath.Join(fixture.root, "debug-build"), "./game"}, flag: "-debug"},
		{name: "install debug", args: []string{"install", "-debug", "./game"}, flag: "-debug"},
	}
	for _, test := range unsupported {
		t.Run(test.name, func(t *testing.T) {
			marker := filepath.Join(fixture.root, strings.ReplaceAll(test.name, " ", "-")+"-started")
			command := cliCommand(xgo, repo, fixture.app, test.args...)
			command.Env = append(command.Env, "FAKE_DRIVER_MARKER="+marker)
			output, err := command.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "does not support flag "+test.flag) {
				t.Fatalf("command = %v\n%s", err, output)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("driver started after rejected flag: %v", err)
			}
		})
	}

	marker := filepath.Join(fixture.root, "driver-started")
	multiBin := filepath.Join(fixture.root, "multi-bin")
	multi := cliCommand(xgo, repo, fixture.app, "install", "./game", "./game")
	multi.Env = append(multi.Env, "GOBIN="+multiBin, "FAKE_DRIVER_MARKER="+marker)
	output, err = multi.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "exactly one target") {
		t.Fatalf("multi driver install = %v\n%s", err, output)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("driver ran before multi-target rejection: %v", err)
	}
	if _, err := os.Stat(multiBin); !os.IsNotExist(err) {
		t.Fatalf("install directory created before rejection: %v", err)
	}

	plain := filepath.Join(fixture.app, "plain")
	mustMkdirAll(t, plain)
	mustWriteFile(t, filepath.Join(plain, "main.go"), "package main\nfunc main() {}\n")
	plainOutput := filepath.Join(fixture.root, executableName("plain"))
	legacy := cliCommand(xgo, repo, fixture.app, "build", "-o", plainOutput, "./plain")
	if output, err := legacy.CombinedOutput(); err != nil {
		t.Fatalf("legacy build: %v\n%s", err, output)
	}
}

func protocolModuleDir(t *testing.T, goWork string) string {
	t.Helper()
	command := exec.Command("go", "list", "-m", "-f={{.Dir}}", "github.com/goplus/mod")
	command.Env = replaceEnv(os.Environ(), "GOWORK", goWork)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve github.com/goplus/mod source: %v\n%s", err, output)
	}
	dir := strings.TrimSpace(string(output))
	if dir == "" {
		t.Fatal("github.com/goplus/mod source directory is empty")
	}
	return canonicalDir(t, dir)
}

func configureSharedProtocolDriver(t *testing.T, fixture driverFixture, modDir string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(fixture.framework, "cmd", "driver", "main.go"), protocolFakeDriverSource)
	frameworkGoModPath := filepath.Join(fixture.framework, "go.mod")
	frameworkGoMod, err := os.ReadFile(frameworkGoModPath)
	if err != nil {
		t.Fatal(err)
	}
	frameworkGoMod = append(frameworkGoMod, []byte("\nrequire github.com/goplus/mod v0.0.0\n")...)
	if err := os.WriteFile(frameworkGoModPath, frameworkGoMod, 0644); err != nil {
		t.Fatal(err)
	}
	goModPath := filepath.Join(fixture.app, "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	goMod = append(goMod, []byte("\nrequire github.com/goplus/mod v0.0.0\n\nreplace github.com/goplus/mod => "+strconv.Quote(modDir)+"\n")...)
	if err := os.WriteFile(goModPath, goMod, 0644); err != nil {
		t.Fatal(err)
	}
	download := exec.Command("go", "mod", "download", "all")
	download.Dir = fixture.app
	download.Env = replaceEnv(os.Environ(), "GOWORK", "off")
	if output, err := download.CombinedOutput(); err != nil {
		t.Fatalf("prepare shared-protocol driver graph: %v\n%s", err, output)
	}
	goMod, err = os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "example.test/framework v1.2.3 //xgo:class") {
		t.Fatalf("fixture lost the class marker:\n%s", goMod)
	}
	if _, err := os.Stat(filepath.Join(fixture.app, "driverprotocol_tools.go")); !os.IsNotExist(err) {
		t.Fatalf("fixture contains a test-only tools file: %v", err)
	}
}

const protocolFakeDriverSource = `package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/goplus/mod/driverprotocol"
)

func main() {
	if marker := os.Getenv("FAKE_DRIVER_MARKER"); marker != "" {
		_ = os.WriteFile(marker, []byte("started"), 0600)
	}
	if value := os.Getenv("FAKE_DRIVER_EXIT"); value != "" {
		code, _ := strconv.Atoi(value)
		os.Exit(code)
	}
	request, err := driverprotocol.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(90)
	}
	if os.Getenv("FAKE_DRIVER_EXPECT_NO_PACK") != "" && request.Project.Pack != nil {
		os.Exit(94)
	}
	switch request.Action {
	case driverprotocol.ActionRun:
		fmt.Printf("run-args=%s\n", strings.Join(request.ApplicationArgs, "|"))
	case driverprotocol.ActionBuild:
		if request.Output == nil {
			os.Exit(91)
		}
		self, err := os.Executable()
		if err != nil { panic(err) }
		data, err := os.ReadFile(self)
		if err != nil { panic(err) }
		if err := os.WriteFile(request.Output.Staging, data, 0755); err != nil { panic(err) }
		if runtime.GOOS == "darwin" {
			if data, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", request.Output.Staging).CombinedOutput(); err != nil {
				fmt.Fprintln(os.Stderr, string(data))
				os.Exit(93)
			}
		}
	default:
		os.Exit(92)
	}
}
`

func cliCommand(xgo, xgoRoot, dir string, args ...string) *exec.Cmd {
	cmd := exec.Command(xgo, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "XGOROOT="+xgoRoot)
	return cmd
}

func assertNoAutogen(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "xgo_autogen.go")); !os.IsNotExist(err) {
		t.Fatalf("driver path touched xgo_autogen.go: %v", err)
	}
}
