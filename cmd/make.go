//go:build ignore
// +build ignore

/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
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

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func checkPathExist(path string, isDir bool) bool {
	stat, err := os.Lstat(path) // Note: os.Lstat() will not follow the symbolic link.
	isExists := !os.IsNotExist(err)
	if isDir {
		return isExists && stat.IsDir()
	}
	return isExists && !stat.IsDir()
}

func trimRight(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

// Path returns single path to check
type Path struct {
	path  string
	isDir bool
}

func (p *Path) checkExists(rootDir string) bool {
	absPath := filepath.Join(rootDir, p.path)
	return checkPathExist(absPath, p.isDir)
}

func getXgoRoot() string {
	pwd, _ := os.Getwd()

	pathsToCheck := []Path{
		{path: "cmd/xgo", isDir: true},
		{path: "builtin", isDir: true},
		{path: "go.mod", isDir: false},
		{path: "go.sum", isDir: false},
	}

	for _, path := range pathsToCheck {
		if !path.checkExists(pwd) {
			println("Error: This script should be run at the root directory of xgo repository.")
			os.Exit(1)
		}
	}
	return pwd
}

var xgoRoot = getXgoRoot()
var initCommandExecuteEnv = os.Environ()
var commandExecuteEnv = initCommandExecuteEnv

// Always put `gop` command as the first item and `xgo` command as the second item,
// as it will be referenced by below code.
var xgoBinFiles = []string{"gop", "xgo"}

const (
	inWindows = (runtime.GOOS == "windows")
)

func init() {
	if inWindows {
		for index := range xgoBinFiles {
			xgoBinFiles[index] += ".exe"
		}
	}
}

type ExecCmdError struct {
	Err    error
	Stderr string
}

func (p *ExecCmdError) Error() string {
	if e := p.Stderr; e != "" {
		return e
	}
	return p.Err.Error()
}

type iGitRemote interface {
	CheckRemoteUrl()
	CreateBranchFrom(branch, remote string) error
	PushCommits(remote, branch string) error
	DeleteBranch(branch string) error
}

type (
	gitRemoteImpl struct{}
	gitRemoteNone struct{}
)

func (p *gitRemoteImpl) CheckRemoteUrl() {
	if getGitRemoteUrl("xgo") == "" {
		log.Fatalln("Error: git remote xgo not found, please use `git remote add xgo git@github.com:goplus/xgo.git`.")
	}
}

func (p *gitRemoteImpl) CreateBranchFrom(branch, remote string) (err error) {
	_, err = execCommand("git", "fetch", remote)
	if err != nil {
		return
	}
	execCommand("git", "branch", "-D", branch)
	_, err = execCommand("git", "checkout", "-b", branch, remote+"/"+branch)
	return
}

func (p *gitRemoteImpl) PushCommits(remote, branch string) error {
	_, err := execCommand("git", "push", remote, branch)
	return err
}

func (p *gitRemoteImpl) DeleteBranch(branch string) error {
	_, err := execCommand("git", "branch", "-D", branch)
	return err
}

func (p *gitRemoteNone) CheckRemoteUrl()                                    {}
func (p *gitRemoteNone) CreateBranchFrom(branch, remote string) (err error) { return nil }
func (p *gitRemoteNone) PushCommits(remote, branch string) error            { return nil }
func (p *gitRemoteNone) DeleteBranch(branch string) error                   { return nil }

var (
	gitRemote iGitRemote = &gitRemoteImpl{}
)

func execCommand(command string, arg ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(command, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = commandExecuteEnv
	err := cmd.Run()
	if err != nil {
		err = &ExecCmdError{Err: err, Stderr: stderr.String()}
	}
	return stdout.String(), err
}

func getTagRev(tag string) string {
	const commit = "commit "
	stdout, err := execCommand("git", "show", tag)
	if err != nil || !strings.HasPrefix(stdout, commit) {
		return ""
	}
	data := stdout[len(commit):]
	if pos := strings.IndexByte(data, '\n'); pos > 0 {
		return data[:pos]
	}
	return ""
}

func getGitRemoteUrl(name string) string {
	stdout, err := execCommand("git", "remote", "get-url", name)
	if err != nil {
		return ""
	}
	return stdout
}

func getGitBranch() string {
	stdout, err := execCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return trimRight(stdout)
}

func gitTag(tag string) error {
	_, err := execCommand("git", "tag", tag)
	return err
}

func gitTagAndPushTo(tag string, remote, branch string) error {
	if err := gitRemote.PushCommits(remote, branch); err != nil {
		return err
	}
	if err := gitTag(tag); err != nil {
		return err
	}
	return gitRemote.PushCommits(remote, tag)
}

func gitAdd(file string) error {
	_, err := execCommand("git", "add", file)
	return err
}

func gitCommit(msg string) error {
	out, err := execCommand("git", "commit", "-a", "-m", msg)
	if err != nil {
		if e := err.(*ExecCmdError); e.Stderr == "" {
			e.Stderr = out
		}
	}
	return err
}

func gitCheckoutBranch(branch string) error {
	_, err := execCommand("git", "checkout", branch)
	return err
}

func isGitRepo() bool {
	gitDir, err := execCommand("git", "rev-parse", "--git-dir")
	if err != nil {
		return false
	}
	return checkPathExist(filepath.Join(xgoRoot, trimRight(gitDir)), true)
}

func getBuildDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02_15-04-05")
}

func getBuildVer() string {
	latestTagCommit, err := execCommand("git", "rev-list", "--tags", "--max-count=1")
	if err != nil {
		return ""
	}

	stdout, err := execCommand("git", "describe", "--tags", trimRight(latestTagCommit))
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s devel", trimRight(stdout))
}

func getGopBuildFlags() string {
	defaultXGoRoot := xgoRoot
	if gopRootFinal := os.Getenv("GOPROOT_FINAL"); gopRootFinal != "" {
		defaultXGoRoot = gopRootFinal
	}
	buildFlags := fmt.Sprintf("-X \"github.com/goplus/xgo/env.defaultXGoRoot=%s\"", defaultXGoRoot)
	buildFlags += fmt.Sprintf(" -X \"github.com/goplus/xgo/env.buildDate=%s\"", getBuildDateTime())

	version := findXgoVersion()
	buildFlags += fmt.Sprintf(" -X \"github.com/goplus/xgo/env.buildVersion=%s\"", version)

	return buildFlags
}

func detectGopBinPath() string {
	return filepath.Join(xgoRoot, "bin")
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

func linkGoplusToLocalBin() string {
	println("Start Linking.")

	gopBinPath := detectGopBinPath()
	goBinPath := detectGoBinPath()
	if !checkPathExist(gopBinPath, true) {
		log.Fatalf("Error: %s is not existed, you should build XGo before linking.\n", gopBinPath)
	}
	if !checkPathExist(goBinPath, true) {
		if err := os.MkdirAll(goBinPath, 0755); err != nil {
			fmt.Printf("Error: target directory %s is not existed and we can't create one.\n", goBinPath)
			log.Fatalln(err)
		}
	}

	for i, file := range xgoBinFiles {
		targetLink := filepath.Join(goBinPath, file)
		if i == 0 { // `gop` is the first file, we will link it to `xgo`.
			file = xgoBinFiles[1]
		}
		sourceFile := filepath.Join(gopBinPath, file)
		if !checkPathExist(sourceFile, false) {
			log.Fatalf("Error: %s is not existed, you should build XGo before linking.\n", sourceFile)
		}
		if checkPathExist(targetLink, false) {
			// Delete existed one
			if err := os.Remove(targetLink); err != nil {
				log.Fatalln(err)
			}
		}
		if err := os.Symlink(sourceFile, targetLink); err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Link %s to %s successfully.\n", sourceFile, targetLink)
	}

	println("End linking.")
	return goBinPath
}

func buildGoplusTools(useGoProxy bool) {
	commandsDir := filepath.Join(xgoRoot, "cmd")
	buildFlags := getGopBuildFlags()

	if useGoProxy {
		println("Info: we will use goproxy.cn as a Go proxy to accelerate installing process.")
		commandExecuteEnv = append(commandExecuteEnv,
			"GOPROXY=https://goproxy.cn,direct",
		)
	}

	// Install XGo binary files under current ./bin directory.
	gopBinPath := detectGopBinPath()
	if err := os.Mkdir(gopBinPath, 0755); err != nil && !os.IsExist(err) {
		println("Error: XGo can't create ./bin directory to put build assets.")
		log.Fatalln(err)
	}

	println("Building XGo tools...\n")
	os.Chdir(commandsDir)

	buildOutput, err := execCommand("go", "build", "-o", gopBinPath, "-v", "-trimpath", "-ldflags", buildFlags, "./...")
	if err != nil {
		log.Fatalln(err)
	}
	print(buildOutput)

	// Clear xgo run cache
	cleanGopRunCache()

	println("\nXGo tools built successfully!")
}

func showHelpPostInstall(installPath string) {
	println("\nNEXT STEP:")
	println("\nWe just installed XGo into the directory: ", installPath)
	message := `
To setup a better XGo development environment,
we recommend you add the above install directory into your PATH environment variable.
	`
	println(message)
}

// Install XGo tools
func install() {
	installPath := linkGoplusToLocalBin()

	println("\nXGo tools installed successfully!")

	if _, err := execCommand("xgo", "version"); err != nil {
		showHelpPostInstall(installPath)
	}
}

func runTestcases() {
	println("Start running testcases.")
	os.Chdir(xgoRoot)

	coverage := "-coverprofile=coverage.txt"
	gopCommand := filepath.Join(detectGopBinPath(), xgoBinFiles[1])
	if !checkPathExist(gopCommand, false) {
		println("Error: XGo must be installed before running testcases.")
		os.Exit(1)
	}

	testOutput, err := execCommand(gopCommand, "test", coverage, "-covermode=atomic", "./...")
	println(testOutput)
	if err != nil {
		println(err.Error())
	}

	println("End running testcases.")
}

func clean() {
	gopBinPath := detectGopBinPath()
	goBinPath := detectGoBinPath()

	// Clean links
	for _, file := range xgoBinFiles {
		targetLink := filepath.Join(goBinPath, file)
		if checkPathExist(targetLink, false) {
			if err := os.Remove(targetLink); err != nil {
				log.Fatalln(err)
			}
		}
	}

	// Clean build binary files
	if checkPathExist(gopBinPath, true) {
		if err := os.RemoveAll(gopBinPath); err != nil {
			log.Fatalln(err)
		}
	}

	cleanGopRunCache()
}

func cleanGopRunCache() {
	homeDir, _ := os.UserHomeDir()
	runCacheDir := filepath.Join(homeDir, ".xgo", "run")
	files := []string{"go.mod", "go.sum"}
	for _, file := range files {
		fullPath := filepath.Join(runCacheDir, file)
		if checkPathExist(fullPath, false) {
			if err := os.Remove(fullPath); err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func uninstall() {
	println("Uninstalling XGo and related tools.")
	clean()
	println("XGo and related tools uninstalled successfully.")
}

func isInChinaWindows() bool {
	// Run `systeminfo` command on windows to check locale.
	out, err := execCommand("systeminfo")
	if err != nil {
		fmt.Println("Run [systeminfo] command failed with error: ", err)
		return false
	}
	// Check if output contains `zh-cn;`
	return strings.Contains(out, "zh-cn;")
}

func isInChina() bool {
	if inWindows {
		return isInChinaWindows()
	}
	const prefix = "LANG=\""
	out, err := execCommand("locale")
	if err != nil {
		return false
	}
	if strings.HasPrefix(out, prefix) {
		out = out[len(prefix):]
		return strings.HasPrefix(out, "zh_CN")
	}
	return false
}

// findXgoVersion returns current version of xgo
func findXgoVersion() string {
	versionFile := filepath.Join(xgoRoot, "VERSION")
	// Read version from VERSION file
	data, err := os.ReadFile(versionFile)
	if err == nil {
		version := trimRight(string(data))
		return version
	}

	// Read version from git repo
	if !isGitRepo() {
		log.Fatal("Error: must be a git repo or a VERSION file existed.")
	}
	version := getBuildVer() // Closet tag on git log
	return version
}

// releaseNewVersion tags the repo with provided new tag, and writes new tag into VERSION file.
func releaseNewVersion(tag string) {
	if !isGitRepo() {
		log.Fatalln("Error: Releasing a new version could only be operated under a git repo.")
	}
	gitRemote.CheckRemoteUrl()
	if getTagRev(tag) != "" {
		log.Fatalln("Error: tag already exists -", tag)
	}

	version := tag
	re := regexp.MustCompile(`^v\d+?\.\d+?`)
	releaseBranch := re.FindString(version)
	if releaseBranch == "" {
		log.Fatal("Error: A valid version should be has form: vX.Y.Z")
	}

	sourceBranch := getGitBranch()

	// Checkout to release breanch
	if sourceBranch != releaseBranch {
		if err := gitCheckoutBranch(releaseBranch); err != nil {
			log.Fatalf("Error: checkout to release branch: %s failed with error: %v.", releaseBranch, err)
		}
		defer func() {
			// Checkout back to source branch
			if err := gitCheckoutBranch(sourceBranch); err != nil {
				log.Fatalf("Error: checkout to source branch: %s failed with error: %v.", sourceBranch, err)
			}
			gitRemote.DeleteBranch(releaseBranch)
		}()
	}

	// Cache new version
	versionFile := filepath.Join(xgoRoot, "VERSION")
	if err := os.WriteFile(versionFile, []byte(version), 0644); err != nil {
		log.Fatalf("Error: cache new version with error: %v\n", err)
	}

	// Commit changes
	gitAdd(versionFile)
	if err := gitCommit("release version " + version); err != nil {
		log.Fatalf("Error: git commit with error: %v\n", err)
	}

	// Tag the source code
	if err := gitTagAndPushTo(tag, "xgo", releaseBranch); err != nil {
		log.Fatalf("Error: gitTagAndPushTo with error: %v\n", err)
	}

	println("Released new version:", version)
}

func runRegtests() {
	println("\nStart running regtests.")

	cmd := exec.Command(filepath.Join(xgoRoot, "bin/"+xgoBinFiles[1]), "go", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Join(xgoRoot, "demo")
	err := cmd.Run()
	if err != nil {
		code := cmd.ProcessState.ExitCode()
		if code == 0 {
			code = 1
		}
		os.Exit(code)
	}
}

func main() {
	isInstall := flag.Bool("install", false, "Install XGo")
	isBuild := flag.Bool("build", false, "Build XGo tools")
	isTest := flag.Bool("test", false, "Run testcases")
	isRegtest := flag.Bool("regtest", false, "Run regtests")
	isUninstall := flag.Bool("uninstall", false, "Uninstall XGo")
	isGoProxy := flag.Bool("proxy", false, "Set GOPROXY for people in China")
	isAutoProxy := flag.Bool("autoproxy", false, "Check to set GOPROXY automatically")
	noPush := flag.Bool("nopush", false, "Don't push to remote repo")
	tag := flag.String("tag", "", "Release an new version with specified tag")

	flag.Parse()

	useGoProxy := *isGoProxy
	if !useGoProxy && *isAutoProxy {
		useGoProxy = isInChina()
	}
	flagActionMap := map[*bool]func(){
		isBuild: func() { buildGoplusTools(useGoProxy) },
		isInstall: func() {
			buildGoplusTools(useGoProxy)
			install()
		},
		isUninstall: uninstall,
		isTest:      runTestcases,
		isRegtest:   runRegtests,
	}

	// Sort flags, for example: install flag should be checked earlier than test flag.
	flags := []*bool{isBuild, isInstall, isTest, isRegtest, isUninstall}
	hasActionDone := false

	if *tag != "" {
		if *noPush {
			gitRemote = &gitRemoteNone{}
		}
		releaseNewVersion(*tag)
		hasActionDone = true
	}

	for _, flag := range flags {
		if *flag {
			flagActionMap[flag]()
			hasActionDone = true
		}
	}

	if !hasActionDone {
		println("Usage:\n")
		flag.PrintDefaults()
	}
}
