package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/mod/modfile"
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

func TestReadCommandRunnerFromGopMod(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun v1.2.3
`)
	writeFile(t, filepath.Join(projectDir, "main.spx"), "")

	runner, err := loadProjectRunner(&xgoprojs.DirProj{Dir: projectDir}, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner == nil {
		t.Fatal("loadProjectRunner() returned nil")
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

	runner, err := loadProjectRunner(&xgoprojs.DirProj{Dir: projectDir}, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner != nil {
		t.Fatalf("loadProjectRunner() = %+v, want nil", runner)
	}
}

func TestReadCommandRunnerAbsent(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
`)
	writeFile(t, filepath.Join(projectDir, "main.spx"), "")

	runner, err := loadProjectRunner(&xgoprojs.DirProj{Dir: projectDir}, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner != nil {
		t.Fatalf("loadProjectRunner() = %+v, want nil", runner)
	}
}

func TestLookupPackageDirPrefersModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.21\n")
	writeFile(t, filepath.Join(root, "cmd", "runner", "main.go"), "package main\nfunc main() {}\n")

	directory, err := lookupPackageDir(root, "example.com/app/cmd/runner")
	if err != nil {
		t.Fatal(err)
	}
	if directory != filepath.Join(root, "cmd", "runner") {
		t.Fatalf("lookupPackageDir() dir = %q", directory)
	}
}

func TestLookupPackageDirPropagatesLoadError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\nrequire (\n")

	_, err := lookupPackageDir(root, "example.com/app")
	if err == nil {
		t.Fatal("lookupPackageDir() error = nil, want error")
	}
}

func TestLookupPackageDirPrefersReplace(t *testing.T) {
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

	directory, err := lookupPackageDir(appRoot, "example.com/runner/cmd/pcrun")
	if err != nil {
		t.Fatal(err)
	}
	if directory != filepath.Join(runnerRoot, "cmd", "pcrun") {
		t.Fatalf("lookupPackageDir() dir = %q", directory)
	}
}

func TestReadCommandRunnerRejectsPathVersionSyntax(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun@latest
`)
	writeFile(t, filepath.Join(projectDir, "main.spx"), "")

	_, err := loadProjectRunner(&xgoprojs.DirProj{Dir: projectDir}, projectDir)
	if err == nil {
		t.Fatal("loadProjectRunner() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid runner path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadCommandRunnerRejectsInvalidImportPath(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner "bad path"
`)
	writeFile(t, filepath.Join(projectDir, "main.spx"), "")

	_, err := loadProjectRunner(&xgoprojs.DirProj{Dir: projectDir}, projectDir)
	if err == nil {
		t.Fatal("loadProjectRunner() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid runner path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallRunnerUsesExplicitVersionQuery(t *testing.T) {
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

	binaryPath, cleanup, err := installRunnerBinary(&modfile.Runner{Path: "example.com/runner/cmd/pcrun", Version: "v1.2.3"})
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

func TestInstallRunnerUsesLatestQuery(t *testing.T) {
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

	binaryPath, cleanup, err := installRunnerBinary(&modfile.Runner{Path: "example.com/runner/cmd/pcrun", Version: "latest"})
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

func TestRunWithConfiguredRunner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based fake go test")
	}

	projectDir := t.TempDir()
	outputFile := filepath.Join(t.TempDir(), "runner.out")
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
	cat >"$GOBIN/pcrun" <<'EOF'
#!/bin/sh
set -eu
{
	printf '%s\n' "$1"
	shift
	printf '%s' "$*"
} >"$TEST_RUNNER_OUTPUT"
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
	t.Setenv("PATH", fakeGoDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_RUNNER_OUTPUT", outputFile)
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.spx Game github.com/example/app
runner example.com/runner/cmd/pcrun
`)
	writeFile(t, filepath.Join(projectDir, "main.spx"), "")

	handled, err := runWithConfiguredRunner(&xgoprojs.DirProj{Dir: projectDir}, []string{"alpha", "beta"}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("runWithConfiguredRunner() = false, want true")
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	want := projectDir + "\nalpha beta"
	if got != want {
		t.Fatalf("runner output = %q, want %q", got, want)
	}
}

func TestReadCommandRunnerSelectsMatchingProjectForFiles(t *testing.T) {
	projectDir := t.TempDir()
	alphaFile := filepath.Join(projectDir, "main.alpha")
	betaFile := filepath.Join(projectDir, "main.beta")
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.alpha App github.com/example/alpha
runner example.com/runner/cmd/alpha

project main.beta App github.com/example/beta
runner example.com/runner/cmd/beta
`)
	writeFile(t, alphaFile, "")
	writeFile(t, betaFile, "")

	runner, err := loadProjectRunner(&xgoprojs.FilesProj{Files: []string{betaFile}}, projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if runner == nil || runner.Path != "example.com/runner/cmd/beta" {
		t.Fatalf("loadProjectRunner() = %+v, want beta runner", runner)
	}
}

func TestReadCommandRunnerRejectsAmbiguousDirectoryProject(t *testing.T) {
	projectDir := t.TempDir()
	writeFile(t, filepath.Join(projectDir, "gop.mod"), `xgo 1.6.0

project main.alpha App github.com/example/alpha
runner example.com/runner/cmd/alpha

project main.beta App github.com/example/beta
runner example.com/runner/cmd/beta
`)
	writeFile(t, filepath.Join(projectDir, "main.alpha"), "")
	writeFile(t, filepath.Join(projectDir, "main.beta"), "")

	_, err := loadProjectRunner(&xgoprojs.DirProj{Dir: projectDir}, projectDir)
	if err == nil {
		t.Fatal("loadProjectRunner() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "multiple projects") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunnerBinaryFilename(t *testing.T) {
	name := runnerBinaryFilename("example.com/runner/cmd/pcrun")
	if runtime.GOOS == "windows" && name != "pcrun.exe" {
		t.Fatalf("windows runner binary = %q, want pcrun.exe", name)
	}
	if runtime.GOOS != "windows" && name != "pcrun" {
		t.Fatalf("non-windows runner binary = %q, want pcrun", name)
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
