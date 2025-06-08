//go:build ignore
// +build ignore

package make_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/xgo/env"
)

const (
	inWindows = (runtime.GOOS == "windows")
)

var script = "all.bash"
var gopRoot = ""
var xgoBinFiles = []string{"gop", "xgo"}
var installer = "cmd/make.go"
var versionFile = "VERSION"
var mainVersionFile = "env/version.go"

func checkPathExist(path string, isDir bool) bool {
	stat, err := os.Stat(path)
	isExists := !os.IsNotExist(err)
	if isDir {
		return isExists && stat.IsDir()
	}
	return isExists && !stat.IsDir()
}

func trimRight(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

func execCommand(command string, arg ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(command, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			err = errors.New(string(stderr.String()))
		}
	}
	return stdout.String(), err
}

func getBranch() string {
	stdout, err := execCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return trimRight(stdout)
}

func detectGoBinPath() string {
	goBin, ok := os.LookupEnv("GOBIN")
	if ok {
		return goBin
	}
	goPath, ok := os.LookupEnv("GOPATH")
	if ok {
		list := filepath.SplitList(goPath)
		if len(list) > 0 {
			// Put in first directory of $GOPATH.
			return filepath.Join(list[0], "bin")
		}
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "go", "bin")
}

func init() {
	pwd, _ := os.Getwd()
	gopRoot = filepath.Join(pwd, "..")
	installer = filepath.Join(gopRoot, installer)
	versionFile = filepath.Join(gopRoot, versionFile)
	mainVersionFile = filepath.Join(gopRoot, mainVersionFile)

	if inWindows {
		script = "all.bat"
		for index := range xgoBinFiles {
			xgoBinFiles[index] += ".exe"
		}
	}
}

func cleanGopRunCacheFiles(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	runCacheDir := filepath.Join(homeDir, ".xgo", "run")
	files := []string{"go.mod", "go.sum"}
	for _, file := range files {
		fullPath := filepath.Join(runCacheDir, file)
		if checkPathExist(fullPath, false) {
			t.Fatalf("Failed: %s found in %s directory\n", file, runCacheDir)
		}
	}
}

func TestAllScript(t *testing.T) {
	os.Chdir(gopRoot)
	cmd := exec.Command(filepath.Join(gopRoot, script))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed: %v:\nOut: %s\n", err, output)
	}

	goBinPath := detectGoBinPath()

	for i, file := range xgoBinFiles {
		if i == 0 { // skip gop
			continue
		}
		if !checkPathExist(filepath.Join(gopRoot, "bin", file), false) {
			t.Fatalf("Failed: %s not found in ./bin directory\n", file)
		}
		if !checkPathExist(filepath.Join(goBinPath, file), false) {
			t.Fatalf("Failed: %s not found in %s/bin directory\n", file, goBinPath)
		}
	}

	cleanGopRunCacheFiles(t)

	cmd = exec.Command(filepath.Join(gopRoot, "bin", xgoBinFiles[1]), "version")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed: %v:\nOut: %s\n", err, output)
	}
}

func TestTagFlagInGitRepo(t *testing.T) {
	os.Chdir(gopRoot)

	// Setup
	tag := "v1.0.90"
	tag2 := "v1.0.91"
	bigtag := "v1.999.12"
	releaseBranch := "v1.0"
	nonExistBranch := "non-exist-branch"
	sourceBranch := getBranch()

	// Teardown
	t.Cleanup(func() {
		gitCmd := exec.Command("git", "tag", "-d", tag)
		gitCmd.CombinedOutput()
		gitCmd = exec.Command("git", "tag", "-d", tag2)
		gitCmd.CombinedOutput()
		execCommand("git", "checkout", sourceBranch)
		execCommand("git", "branch", "-D", nonExistBranch)
		execCommand("git", "branch", "-D", releaseBranch)
	})

	t.Run("release new version with bad tag", func(t *testing.T) {
		cmd := exec.Command("go", "run", installer, "--nopush", "--tag", "xyz")
		if _, err := cmd.CombinedOutput(); err == nil {
			t.Fatal("Failed: release a bad tag should be failed.")
		}
	})

	t.Run("empty tag should failed", func(t *testing.T) {
		cmd := exec.Command("go", "run", installer, "--nopush", "--tag", "")
		if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "Usage") {
			t.Fatalf("Failed: %v, out: %s\n", err, out)
		}
	})

	t.Run("failed when release branch corresponding to tag does not exists", func(t *testing.T) {
		cmd := exec.Command("go", "run", installer, "--nopush", "--tag", bigtag)
		if _, err := cmd.CombinedOutput(); err == nil {
			t.Fatal("Failed: a corresponding release branch does not exists should be failed.")
		}
	})

	t.Run("release new version on release branch", func(t *testing.T) {
		execCommand("git", "checkout", sourceBranch)
		execCommand("git", "branch", "-D", releaseBranch)
		_, err := execCommand("git", "checkout", "-b", releaseBranch)
		if err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command("go", "run", installer, "--nopush", "--tag", tag)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Log(string(out))
			t.Fatalf("Failed: release tag: %s on branch: %s should not be failed", tag, releaseBranch)
		}

		if !checkPathExist(versionFile, false) {
			t.Fatal("Failed: a VERSION file not found.")
		}

		if data, _ := os.ReadFile(versionFile); string(data) != tag {
			t.Fatalf("Failed: content of VERSION file: '%s' not match tag: %s", data, tag)
		}

		// Make sure tag exists.
		gitCmd := exec.Command("git", "tag")
		if allTags, _ := gitCmd.CombinedOutput(); !strings.Contains(string(allTags), tag) {
			t.Fatalf("Failed: %s tag not found in tag list of this git repo.\n", tag)
		}
	})

	t.Run("release new version on non-release branch", func(t *testing.T) {
		_, err := execCommand("git", "checkout", "-b", nonExistBranch)
		if err != nil {
			t.Fatal(err)
		}

		execCommand("git", "branch", "-D", releaseBranch)
		_, err = execCommand("git", "checkout", "-b", releaseBranch)
		if err != nil {
			t.Log("current branch:", getBranch())
			t.Fatal(err)
		}

		execCommand("git", "checkout", nonExistBranch)

		cmd := exec.Command("go", "run", installer, "--nopush", "--tag", tag2)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Log(string(out))
			t.Fatalf("Failed: release tag: %s on branch: %s should not be failed", tag2, nonExistBranch)
		}

		if getBranch() != nonExistBranch {
			t.Fatal("Failed: getBranch() != nonExistBranch")
		}
		execCommand("git", "checkout", releaseBranch)

		if !checkPathExist(versionFile, false) {
			t.Fatal("Failed: a VERSION file not found.")
		}

		if data, _ := os.ReadFile(versionFile); string(data) != tag2 {
			t.Fatalf("Failed: content of VERSION file: '%s' not match tag: %s", data, tag2)
		}

		gitCmd := exec.Command("git", "tag")
		if allTags, _ := gitCmd.CombinedOutput(); !strings.Contains(string(allTags), tag2) {
			t.Fatalf("Failed: %s tag not found in tag list of this git repo.\n", tag)
		}
	})
}

func TestTagFlagInNonGitRepo(t *testing.T) {
	os.Chdir(gopRoot)

	// Setup
	gitDir := filepath.Join(gopRoot, ".git")
	gitBackupDir := filepath.Join(gopRoot, ".gitBackup")
	// Rename .git dir
	if checkPathExist(gitDir, true) {
		os.Rename(gitDir, gitBackupDir)
	}

	// Teardown
	t.Cleanup(func() {
		if checkPathExist(gitBackupDir, true) {
			os.Rename(gitBackupDir, gitDir)
		}
	})

	t.Run("specify new tag", func(t *testing.T) {
		cmd := exec.Command("go", "run", installer, "--tag", "v1.0.98")
		output, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "Error") {
			t.Fatal("Failed: release on a non-git repo should be failed.")
		}
	})
}

func TestInstallInNonGitRepo(t *testing.T) {
	os.Chdir(gopRoot)

	// Setup
	gitDir := filepath.Join(gopRoot, ".git")
	gitBackupDir := filepath.Join(gopRoot, ".gitBackup")
	// Rename .git dir
	if checkPathExist(gitDir, true) {
		os.Rename(gitDir, gitBackupDir)
	}

	// Teardown
	t.Cleanup(func() {
		if checkPathExist(gitBackupDir, true) {
			os.Rename(gitBackupDir, gitDir)
		}
		if checkPathExist(versionFile, false) {
			os.Remove(versionFile)
		}
	})

	installCmd := func() *exec.Cmd {
		return exec.Command("go", "run", installer, "--install")
	}

	t.Run("failed build operation", func(t *testing.T) {
		os.Remove(versionFile)
		cmd := installCmd()
		output, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "Error") {
			t.Log(string(output))
			t.Fatal("Failed: build XGo in a non-git repo and no VERSION file should be failed.")
		}
	})

	t.Run("install with VERSION file", func(t *testing.T) {
		version := fmt.Sprintf("v%s.65535", env.MainVersion)
		// Create VERSION file
		if err := os.WriteFile(versionFile, []byte(version), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := installCmd()
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed: %v, output: %s\n", err, output)
		}

		cmd = exec.Command(filepath.Join(gopRoot, "bin", xgoBinFiles[1]), "version")
		output, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(output), version) {
			t.Fatalf("Failed: %v, output: %s\n", err, output)
		}
	})
}

func TestHandleMultiFlags(t *testing.T) {
	os.Chdir(gopRoot)

	cmd := exec.Command("go", "run", installer, "--install", "--test", "--uninstall")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed: %v:\nOut: %s\n", err, output)
	}

	goBinPath := detectGoBinPath()

	// Uninstall will be the last action to run, test if all build assets being cleared.
	for _, file := range xgoBinFiles {
		if checkPathExist(filepath.Join(gopRoot, "bin", file), false) {
			t.Fatalf("Failed: %s found in ./bin directory\n", file)
		}
		if checkPathExist(filepath.Join(goBinPath, file), false) {
			t.Fatalf("Failed: %s found in %s/bin directory\n", file, goBinPath)
		}
	}
}
