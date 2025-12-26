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

// Package run implements the "gop run" command.
package run

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/mod/modcache"
	"github.com/goplus/mod/modfetch"
	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/tool"
	"github.com/goplus/xgo/x/gocmd"
	"github.com/goplus/xgo/x/xgoprojs"
	"github.com/qiniu/x/log"
)

// gop run
var Cmd = &base.Command{
	UsageLine: "gop run [-nc -asm -quiet -debug -prof] package [arguments...]",
	Short:     "Run an XGo program",
}

var (
	flag        = &Cmd.Flag
	flagAsm     = flag.Bool("asm", false, "generates `asm` code of XGo bytecode backend")
	flagDebug   = flag.Bool("debug", false, "print debug information")
	flagQuiet   = flag.Bool("quiet", false, "don't generate any compiling stage log")
	flagNoChdir = flag.Bool("nc", false, "don't change dir (only for `gop run pkgPath`)")
	flagProf    = flag.Bool("prof", false, "do profile and generate profile report")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	pass := base.PassBuildFlags(cmd)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}
	if flag.NArg() < 1 {
		cmd.Usage(os.Stderr)
	}

	proj, args, err := xgoprojs.ParseOne(flag.Args()...)
	if err != nil {
		log.Fatalln(err)
	}

	if *flagQuiet {
		log.SetOutputLevel(0x7000)
	} else if *flagDebug {
		gogen.SetDebug(gogen.DbgFlagAll &^ gogen.DbgFlagComments)
		cl.SetDebug(cl.DbgFlagAll)
		cl.SetDisableRecover(true)
	} else if *flagAsm {
		gogen.SetDebug(gogen.DbgFlagInstruction)
	}

	if *flagProf {
		panic("TODO: profile not impl")
	}

	noChdir := *flagNoChdir
	// Keep original behavior: load configuration from current directory
	conf, err := tool.NewDefaultConf(".", tool.ConfFlagNoTestFiles, pass.Tags())
	if err != nil {
		log.Panicln("tool.NewDefaultConf:", err)
	}
	defer conf.UpdateCache()

	if !conf.Mod.HasModfile() { // if no go.mod, check GopDeps
		conf.XGoDeps = new(int)
	}
	confCmd := conf.NewGoCmdConf()
	confCmd.Flags = pass.Args

	// Check for custom runner and run if found
	if err := runWithCustomRunnerCheck(proj, args, conf); err != nil {
		if err == errNoCustomRunner {
			// No custom runner, proceed with normal execution
			run(proj, args, !noChdir, conf, confCmd)
			return
		}
		// Custom runner executed or error occurred
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var errNoCustomRunner = fmt.Errorf("no custom runner found")

// runWithCustomRunnerCheck checks for custom runner and executes it if found.
// Returns errNoCustomRunner if no custom runner is defined.
func runWithCustomRunnerCheck(proj xgoprojs.Proj, args []string, conf *tool.Config) error {
	var projDir string

	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		projDir = v.Dir
	case *xgoprojs.FilesProj:
		if len(v.Files) > 0 {
			projDir = filepath.Dir(v.Files[0])
		} else {
			return errNoCustomRunner
		}
	case *xgoprojs.PkgPathProj:
		// For package path projects, resolve to local directory without generating files
		localDir, err := resolvePkgPath("", v.Path, conf)
		if err != nil {
			return err
		}
		projDir = localDir
	default:
		return errNoCustomRunner
	}

	// Convert to absolute path
	if !filepath.IsAbs(projDir) {
		if absDir, err := filepath.Abs(projDir); err == nil {
			projDir = absDir
		}
	}

	// Check if there's a custom runner defined in gop.mod of the target project
	runCmd := getCustomRunner(projDir)
	if runCmd == "" {
		return errNoCustomRunner
	}

	// Run the custom runner
	return runCustomRunner(runCmd, projDir, args)
}

// resolvePkgPath resolves a package path to its local directory without generating files.
// This is similar to GenGoPkgPathEx but without the file generation side effects.
func resolvePkgPath(workDir, pkgPath string, conf *tool.Config) (localDir string, err error) {
	// Strip /... suffix if present
	recursively := false
	if len(pkgPath) > 4 && pkgPath[len(pkgPath)-4:] == "/..." {
		pkgPath = pkgPath[:len(pkgPath)-4]
		recursively = true
	}

	// Check if it's a versioned remote package (contains @)
	if strings.Contains(pkgPath, "@") {
		return getRemotePkgDir(pkgPath)
	}

	// Try to load from local module
	mod, err := xgomod.Load(workDir)
	if err != nil {
		if tool.NotFound(err) {
			// Not found locally, try as remote package
			return getRemotePkgDir(pkgPath)
		}
		return "", err
	}

	// Lookup the package in the module
	pkg, err := mod.Lookup(pkgPath)
	if err != nil {
		return "", err
	}

	if recursively {
		return "", fmt.Errorf("cannot use /... pattern for custom runner")
	}

	return pkg.Dir, nil
}

// getRemotePkgDir gets the local directory for a remote package.
// This is extracted from tool.remotePkgPathDo to avoid generating files.
func getRemotePkgDir(pkgPath string) (string, error) {
	modVer, leftPart, err := modfetch.GetPkg(pkgPath, "")
	if err != nil {
		return "", err
	}
	dir, err := modcache.Path(modVer)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, leftPart), nil
}

// getCustomRunner checks if the target project directory has a custom runner
// defined in its gop.mod file. Returns the runner command package path or empty string.
func getCustomRunner(projDir string) string {
	// Try to load module from the project directory
	mod, err := xgomod.Load(projDir)
	if err != nil {
		return ""
	}
	if mod.Opt != nil && mod.Opt.Runner != nil {
		return mod.Opt.Runner.Cmd
	}
	return ""
}

// runCustomRunner runs a custom runner command defined in gop.mod.
// It builds the runner in a temporary directory if not found in PATH or GOPATH/bin.
// Supports version specification in cmdPkgPath (e.g., "pkg@version").
func runCustomRunner(cmdPkgPath, projDir string, args []string) error {
	// Parse package path and version
	// Format: "github.com/goplus/spx/v2/cmd/spxrun" or "github.com/goplus/spx/v2/cmd/spxrun@v2.0.0"
	pkgPath, version := cmdPkgPath, ""
	if idx := strings.Index(cmdPkgPath, "@"); idx > 0 {
		pkgPath = cmdPkgPath[:idx]
		version = cmdPkgPath[idx+1:]
	}

	// Extract the command name from package path
	// e.g., "github.com/goplus/spx/v2/cmd/spxrun" -> "spxrun"
	cmdName := filepath.Base(pkgPath)

	// Try to find the command in PATH first
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		// Try GOPATH/bin
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			gopath = filepath.Join(os.Getenv("HOME"), "go")
		}
		cmdPath = filepath.Join(gopath, "bin", cmdName)
		if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
			// Build the runner in a temporary directory
			fmt.Printf("Building custom runner: %s\n", cmdPkgPath)
			if cmdPath, err = buildRunner(pkgPath, version, cmdName); err != nil {
				return err
			}
		}
	}

	// Build the command arguments: first positional argument is project directory
	// Second optional argument is version (if specified in gop.mod)
	cmdArgs := []string{projDir}
	if version != "" {
		cmdArgs = append(cmdArgs, version)
	}
	cmdArgs = append(cmdArgs, args...)

	// Run the custom runner
	runnerCmd := exec.Command(cmdPath, cmdArgs...)
	runnerCmd.Dir = projDir
	runnerCmd.Stdout = os.Stdout
	runnerCmd.Stderr = os.Stderr
	runnerCmd.Stdin = os.Stdin

	return runnerCmd.Run()
}

// buildRunner builds a Go command package in a temporary directory and returns the path to the binary
// pkgPath is the package path without version (e.g., "github.com/goplus/spx/v2/cmd/spxrun")
// version is the version string (e.g., "v2.0.30" or "" for latest)
func buildRunner(pkgPath, version, cmdName string) (string, error) {
	// Create a temporary directory for building
	tmpDir, err := os.MkdirTemp("", "xgo-runner-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple go.mod in the temp directory
	goModContent := "module temp\n\ngo 1.18\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Determine version specifier
	versionSpec := "@latest"
	if version != "" {
		versionSpec = "@" + version
	}

	// First, use go get to fetch the package with the specified version
	pkgWithVersion := pkgPath + versionSpec
	fmt.Printf("Fetching %s...\n", pkgWithVersion)
	getCmd := exec.Command("go", "get", pkgWithVersion)
	getCmd.Dir = tmpDir
	getCmd.Stdout = os.Stdout
	getCmd.Stderr = os.Stderr
	if err := getCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to fetch package %s: %w", pkgWithVersion, err)
	}

	// Get the output path in GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}
	outPath := filepath.Join(gopath, "bin", cmdName)

	// Build the command (without version in build command)
	fmt.Printf("Building %s to %s...\n", pkgPath, outPath)
	buildCmd := exec.Command("go", "build", "-o", outPath, pkgPath)
	buildCmd.Dir = tmpDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build custom runner %s: %w", pkgPath, err)
	}

	return outPath, nil
}

func run(proj xgoprojs.Proj, args []string, chDir bool, conf *tool.Config, run *gocmd.RunConfig) {
	const flags = 0
	var obj string
	var err error
	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		obj = v.Dir
		err = tool.RunDir(obj, args, conf, run, flags)
	case *xgoprojs.PkgPathProj:
		obj = v.Path
		err = tool.RunPkgPath(v.Path, args, chDir, conf, run, flags)
	case *xgoprojs.FilesProj:
		err = tool.RunFiles("", v.Files, args, conf, run)
	default:
		log.Panicln("`gop run` doesn't support", reflect.TypeOf(v))
	}
	if tool.NotFound(err) {
		fmt.Fprintf(os.Stderr, "gop run %v: not found\n", obj)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		return
	}
	os.Exit(1)
}
