package run

import "github.com/goplus/xgo/x/xgoprojs"

func runWithConfiguredRunner(proj xgoprojs.Proj, args []string, workDir string) (bool, error) {
	projectDirectory, err := resolveProjectDir(proj, workDir)
	if err != nil {
		return false, err
	}

	runner, err := loadProjectRunner(proj, projectDirectory)
	if err != nil {
		return false, err
	}
	if runner == nil {
		return false, nil
	}

	runnerBinaryPath, cleanup, err := installRunnerBinary(runner)
	if err != nil {
		return true, err
	}
	defer cleanup()

	return true, executeRunnerBinary(runnerBinaryPath, projectDirectory, args)
}
