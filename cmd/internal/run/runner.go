package run

import (
	"os"
	"os/exec"

	"github.com/goplus/xgo/x/xgoprojs"
)

func tryRunWithCommandRunner(proj xgoprojs.Proj, args []string, workDir string) (bool, error) {
	projectDir, err := resolveProjectDir(proj, workDir)
	if err != nil {
		return false, err
	}

	runner, err := readCommandRunner(projectDir)
	if err != nil {
		return false, err
	}
	if runner == nil {
		return false, nil
	}

	binaryPath, cleanup, err := prepareRunnerBinary(projectDir, runner)
	if err != nil {
		return true, err
	}
	defer cleanup()

	return true, runCommandRunner(binaryPath, projectDir, args)
}

func runCommandRunner(binaryPath, projectDir string, args []string) error {
	cmd := exec.Command(binaryPath, append([]string{projectDir}, args...)...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
