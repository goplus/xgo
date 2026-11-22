package run

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/goplus/mod/modfile"
)

func installRunnerBinary(runner *modfile.Runner) (string, func(), error) {
	temporaryDirectory, err := os.MkdirTemp("", "xgo-runner-install-*")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(temporaryDirectory)
	}

	binaryPath, err := installRunnerBinaryToDirectory(temporaryDirectory, runner.Path, runner.Version)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return binaryPath, cleanup, nil
}

func installRunnerBinaryToDirectory(targetDirectory, packagePath, version string) (string, error) {
	packageReference := packageRef(packagePath, version)
	output, err := runGoCommand("", []string{"GOBIN=" + targetDirectory}, "install", packageReference)
	if err != nil {
		return "", fmt.Errorf("install runner %s: %w\n%s", packageReference, err, output)
	}

	binaryPath := filepath.Join(targetDirectory, runnerBinaryFilename(packagePath))
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("installed runner binary %s: %w", binaryPath, err)
	}
	return binaryPath, nil
}

func runnerBinaryFilename(packagePath string) string {
	filename := path.Base(packagePath)
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	return filename
}
