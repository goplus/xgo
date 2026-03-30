package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/xgo/x/xgoprojs"
)

func TestResolveProjectDir(t *testing.T) {
	root := t.TempDir()
	filesDir := filepath.Join(root, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(filesDir, "main.gop")
	if err := os.WriteFile(file, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	moduleRoot := filepath.Join(root, "module")
	writeFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/app\n\ngo 1.21\n")
	writeFile(t, filepath.Join(moduleRoot, "pkg", "main.gop"), "package main\n")

	cases := []struct {
		name string
		proj xgoprojs.Proj
		want string
	}{
		{name: "dir", proj: &xgoprojs.DirProj{Dir: filesDir}, want: filesDir},
		{name: "files", proj: &xgoprojs.FilesProj{Files: []string{file}}, want: filesDir},
		{name: "pkg", proj: &xgoprojs.PkgPathProj{Path: "example.com/app/pkg"}, want: filepath.Join(moduleRoot, "pkg")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveProjectDir(tc.proj, moduleRoot)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("resolveProjectDir() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveProjectPackageDirRejectsPattern(t *testing.T) {
	_, err := resolveProjectPackageDir(".", "example.com/app/...")
	if err == nil {
		t.Fatal("resolveProjectPackageDir() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "/...") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadCommandRunnerFromGopMod(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun v1.2.3
`)

	runner, err := readCommandRunner(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner == nil {
		t.Fatal("readCommandRunner() returned nil")
	}
	if runner.Path != "example.com/runner/cmd/pcrun" || runner.Version != "v1.2.3" {
		t.Fatalf("unexpected runner: %+v", runner)
	}
}

func TestReadCommandRunnerOnlyReadsProjectGopMod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun
`)
	projectDir := filepath.Join(root, "sub", "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	runner, err := readCommandRunner(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner != nil {
		t.Fatalf("readCommandRunner() = %+v, want nil", runner)
	}
}

func TestReadCommandRunnerAbsent(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
`)

	runner, err := readCommandRunner(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner != nil {
		t.Fatalf("readCommandRunner() = %+v, want nil", runner)
	}
}

func TestLookupModulePackagePrefersModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.21\n")
	writeFile(t, filepath.Join(root, "cmd", "runner", "main.go"), "package main\nfunc main() {}\n")

	pkg, err := lookupModulePackage(root, "example.com/app/cmd/runner")
	if err != nil {
		t.Fatal(err)
	}
	if pkg == nil {
		t.Fatal("lookupModulePackage() should resolve local module")
	}
	if pkg.Dir != filepath.Join(root, "cmd", "runner") {
		t.Fatalf("lookupModulePackage() dir = %q", pkg.Dir)
	}
}

func TestLookupModulePackagePrefersReplace(t *testing.T) {
	root := t.TempDir()
	runnerRoot := filepath.Join(root, "runner")
	appRoot := filepath.Join(root, "app")

	writeFile(t, filepath.Join(runnerRoot, "go.mod"), "module example.com/runner\n\ngo 1.21\n")
	writeFile(t, filepath.Join(runnerRoot, "cmd", "pcrun", "main.go"), "package main\nfunc main() {}\n")
	writeFile(t, filepath.Join(appRoot, "go.mod"), `module example.com/app

go 1.21

require example.com/runner v0.0.0

replace example.com/runner => ../runner
`)

	pkg, err := lookupModulePackage(appRoot, "example.com/runner/cmd/pcrun")
	if err != nil {
		t.Fatal(err)
	}
	if pkg == nil {
		t.Fatal("lookupModulePackage() should resolve local replace")
	}
	if pkg.Dir != filepath.Join(runnerRoot, "cmd", "pcrun") {
		t.Fatalf("lookupModulePackage() dir = %q", pkg.Dir)
	}
}

func TestReadCommandRunnerRejectsPathVersionSyntax(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun@latest
`)

	_, err := readCommandRunner(projectDir)
	if err == nil {
		t.Fatal("readCommandRunner() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "must not include @version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallTempRunnerBinaryUsesExplicitVersionQuery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based fake go test")
	}

	installLog := filepath.Join(t.TempDir(), "install.log")
	fakeGoDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeGoDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(fakeGoDir, "go"), `#!/bin/sh
set -eu
cmd="$1"
shift
case "$cmd" in
install)
	printf '%s\n' "$1" >>"$FAKE_GO_INSTALL_LOG"
	cat >"$GOBIN/pcrun" <<'EOF'
#!/bin/sh
printf 'remote' > "$1"
EOF
	chmod +x "$GOBIN/pcrun"
	exit 0
	;;
esac
echo "unexpected go command: $cmd $*" >&2
exit 1
`)
	if err := os.Chmod(filepath.Join(fakeGoDir, "go"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_GO_INSTALL_LOG", installLog)
	t.Setenv("PATH", fakeGoDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	binaryPath, cleanup, err := installTempRunnerBinary("example.com/runner/cmd/pcrun", "v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(installLog)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(data))
	if len(lines) != 1 || lines[0] != "example.com/runner/cmd/pcrun@v1.2.3" {
		t.Fatalf("unexpected install log: %q", data)
	}

	outputFile := filepath.Join(t.TempDir(), "runner.out")
	cmd := exec.Command(binaryPath, outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run temp runner: %v\n%s", err, out)
	}
	got, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "remote" {
		t.Fatalf("temp runner output = %q, want remote", got)
	}
}

func TestInstallTempRunnerBinaryUsesLatestQuery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based fake go test")
	}

	installLog := filepath.Join(t.TempDir(), "install.log")
	fakeGoDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeGoDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(fakeGoDir, "go"), `#!/bin/sh
set -eu
cmd="$1"
shift
case "$cmd" in
install)
	printf '%s\n' "$1" >>"$FAKE_GO_INSTALL_LOG"
	cat >"$GOBIN/pcrun" <<'EOF'
#!/bin/sh
printf 'remote-latest' > "$1"
EOF
	chmod +x "$GOBIN/pcrun"
	exit 0
	;;
esac
echo "unexpected go command: $cmd $*" >&2
exit 1
`)
	if err := os.Chmod(filepath.Join(fakeGoDir, "go"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_GO_INSTALL_LOG", installLog)
	t.Setenv("PATH", fakeGoDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	binaryPath, cleanup, err := installTempRunnerBinary("example.com/runner/cmd/pcrun", "latest")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	data, err := os.ReadFile(installLog)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(data))
	if len(lines) != 1 || lines[0] != "example.com/runner/cmd/pcrun@latest" {
		t.Fatalf("unexpected install log: %q", data)
	}

	outputFile := filepath.Join(t.TempDir(), "runner.out")
	cmd := exec.Command(binaryPath, outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run temp runner: %v\n%s", err, out)
	}
	got, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "remote-latest" {
		t.Fatalf("temp runner output = %q, want remote-latest", got)
	}
}

func TestBuildLocalRunnerBinaryRebuildsEveryTime(t *testing.T) {
	sourceDir := filepath.Join(t.TempDir(), "runner")
	writeFile(t, filepath.Join(sourceDir, "go.mod"), "module example.com/runner\n\ngo 1.21\n")
	writeFile(t, filepath.Join(sourceDir, "main.go"), `package main

import "os"

func main() {
	_ = os.WriteFile(os.Args[1], []byte("v1"), 0644)
}
`)

	binary1, cleanup1, err := buildLocalRunnerBinary(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup1()
	writeFile(t, filepath.Join(sourceDir, "main.go"), `package main

import "os"

func main() {
	_ = os.WriteFile(os.Args[1], []byte("v2"), 0644)
}
`)
	binary2, cleanup2, err := buildLocalRunnerBinary(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup2()
	if binary1 == binary2 {
		t.Fatal("local runner should not reuse cached binary path")
	}

	outputFile := filepath.Join(t.TempDir(), "runner.out")
	cmd := exec.Command(binary2, outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run rebuilt local runner: %v\n%s", err, out)
	}
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "v2" {
		t.Fatalf("rebuilt local runner output = %q, want v2", data)
	}
}

func TestRunWithCommandRunner(t *testing.T) {
	root := t.TempDir()
	runnerRoot := filepath.Join(root, "runner")
	projectDir := filepath.Join(root, "project")
	outputFile := filepath.Join(root, "runner.out")

	writeFile(t, filepath.Join(runnerRoot, "go.mod"), "module example.com/runner\n\ngo 1.21\n")
	writeFile(t, filepath.Join(runnerRoot, "cmd", "pcrun", "main.go"), `package main

import (
	"os"
	"strings"
)

func main() {
	data := os.Args[1] + "\n" + strings.Join(os.Args[2:], "|")
	if err := os.WriteFile(os.Getenv("TEST_RUNNER_OUTPUT"), []byte(data), 0644); err != nil {
		panic(err)
	}
}
`)
	writeFile(t, filepath.Join(projectDir, "go.mod"), `module example.com/app

go 1.21

require example.com/runner v0.0.0

replace example.com/runner => ../runner
`)
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun
`)

	t.Setenv("TEST_RUNNER_OUTPUT", outputFile)
	handled, err := tryRunWithCommandRunner(&xgoprojs.DirProj{Dir: projectDir}, []string{"alpha", "beta"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("tryRunWithCommandRunner() = false, want true")
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	want := projectDir + "\nalpha|beta"
	if got != want {
		t.Fatalf("runner output = %q, want %q", got, want)
	}
}

func TestBuildRunnerExecutableRejectsNonMainPackage(t *testing.T) {
	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "go.mod"), "module example.com/runner\n\ngo 1.21\n")
	writeFile(t, filepath.Join(sourceDir, "runner.go"), "package helper\n")

	err := buildRunnerExecutable(sourceDir, filepath.Join(t.TempDir(), "runner"+runnerBinaryExt()))
	if err == nil {
		t.Fatal("buildRunnerExecutable() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "not a main package") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunnerBinaryExt(t *testing.T) {
	if runtime.GOOS == "windows" && runnerBinaryExt() != ".exe" {
		t.Fatal("windows runner binary must end with .exe")
	}
	if runtime.GOOS != "windows" && runnerBinaryExt() != "" {
		t.Fatal("non-windows runner binary should not have extension")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
