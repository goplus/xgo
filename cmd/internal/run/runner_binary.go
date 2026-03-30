package run

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goplus/mod/modcache"
	"github.com/goplus/mod/modfile"
)

func prepareRunnerBinary(projectDir string, runner *modfile.Runner) (string, func(), error) {
	pkgPath := runner.Path
	version := runner.Version

	pkg, err := lookupModulePackage(projectDir, pkgPath)
	if err != nil {
		return "", nil, err
	}
	if pkg != nil && pkg.ModDir != "" && !modcache.InPath(pkg.ModDir) {
		return buildLocalRunnerBinary(pkg.Dir)
	}
	if version == "" {
		version = "latest"
	}
	return installTempRunnerBinary(pkgPath, version)
}

func buildLocalRunnerBinary(packageDir string) (string, func(), error) {
	return withRunnerTempBinary("build", func(tempDir string) (string, error) {
		binaryPath := filepath.Join(tempDir, "runner"+runnerBinaryExt())
		if err := buildRunnerExecutable(packageDir, binaryPath); err != nil {
			return "", err
		}
		return binaryPath, nil
	})
}

func installTempRunnerBinary(pkgPath, version string) (string, func(), error) {
	return withRunnerTempBinary("install", func(tempDir string) (string, error) {
		return installRunnerBinaryToDir(tempDir, pkgPath, version)
	})
}

func withRunnerTempBinary(kind string, prepare func(tempDir string) (string, error)) (string, func(), error) {
	tempDir, err := newRunnerTempDir(kind)
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	binaryPath, err := prepare(tempDir)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return binaryPath, cleanup, nil
}

func installRunnerExecutable(targetDir, pkgPath, version string) error {
	output, err := goCommandOutputWithEnv("", []string{"GOBIN=" + targetDir}, "install", pkgPath+"@"+version)
	if err != nil {
		return formatGoCommandError(fmt.Sprintf("install runner %s@%s", pkgPath, version), output, err)
	}
	return nil
}

func installRunnerBinaryToDir(targetDir, pkgPath, version string) (string, error) {
	if err := installRunnerExecutable(targetDir, pkgPath, version); err != nil {
		return "", err
	}
	binaryPath := filepath.Join(targetDir, runnerBinaryName(pkgPath))
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("installed runner binary %s: %w", binaryPath, err)
	}
	return binaryPath, nil
}

func runnerBinaryName(pkgPath string) string {
	return path.Base(pkgPath) + runnerBinaryExt()
}

func newRunnerTempDir(kind string) (string, error) {
	return os.MkdirTemp("", "xgo-runner-"+kind+"-*")
}

func runnerBinaryExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func buildRunnerExecutable(packageDir, binaryPath string) error {
	if err := validateMainPackage(packageDir); err != nil {
		return err
	}
	output, err := goCommandOutput(packageDir, "build", "-o", binaryPath, ".")
	if err != nil {
		return formatGoCommandError(fmt.Sprintf("build runner in %s", packageDir), output, err)
	}
	return nil
}

func validateMainPackage(packageDir string) error {
	output, err := goCommandOutput(packageDir, "list", "-f", "{{.Name}}", ".")
	if err != nil {
		return formatGoCommandError(fmt.Sprintf("inspect runner package %s", packageDir), output, err)
	}
	if output != "main" {
		return fmt.Errorf("runner package %s is not a main package", packageDir)
	}
	return nil
}

func formatGoCommandError(prefix, output string, err error) error {
	if output == "" {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	return fmt.Errorf("%s: %w\n%s", prefix, err, output)
}

func goCommandOutput(dir string, args ...string) (string, error) {
	return goCommandOutputWithEnv(dir, nil, args...)
}

func goCommandOutputWithEnv(dir string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Env = append(cmd.Env, extraEnv...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}
