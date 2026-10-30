package run

import (
	"os"
	"os/exec"
	"strings"
)

func executeRunnerBinary(binaryPath, projectDirectory string, args []string) error {
	cmd := newCommandInDir(binaryPath, projectDirectory, append([]string{projectDirectory}, args...)...)
	// Configured runners are project-controlled executables, so they intentionally
	// inherit the caller environment just like other tools launched by xgo.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runGoCommand(directory string, extraEnv []string, args ...string) (string, error) {
	cmd := newCommandInDir("go", directory, args...)
	// Runner installation/build should not be redirected by an ambient go.work file.
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Env = append(cmd.Env, extraEnv...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func newCommandInDir(command, directory string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	if directory != "" {
		cmd.Dir = directory
	}
	return cmd
}
