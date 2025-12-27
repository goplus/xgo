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
	"github.com/goplus/mod/modfile"
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
	conf, err := tool.NewDefaultConf(".", tool.ConfFlagNoTestFiles, pass.Tags())
	if err != nil {
		log.Panicln("tool.NewDefaultConf:", err)
	}
	defer conf.UpdateCache()

	if !conf.Mod.HasModfile() {
		conf.XGoDeps = new(int)
	}
	confCmd := conf.NewGoCmdConf()
	confCmd.Flags = pass.Args

	// Try custom runner first, fallback to normal execution
	if err := tryCustomRunner(proj, args, conf); err == nil {
		return
	}

	run(proj, args, !noChdir, conf, confCmd)
}

// tryCustomRunner attempts to run with a custom runner if defined
func tryCustomRunner(proj xgoprojs.Proj, args []string, conf *tool.Config) error {
	projDir, err := getProjDir(proj, conf)
	if err != nil {
		return err
	}

	runner := loadCustomRunner(projDir)
	if runner == nil {
		return fmt.Errorf("no custom runner")
	}

	return executeRunner(runner, projDir, args)
}

// getProjDir extracts the project directory from different project types
func getProjDir(proj xgoprojs.Proj, conf *tool.Config) (string, error) {
	var dir string

	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		dir = v.Dir
	case *xgoprojs.FilesProj:
		if len(v.Files) == 0 {
			return "", fmt.Errorf("no files in project")
		}
		dir = filepath.Dir(v.Files[0])
	case *xgoprojs.PkgPathProj:
		return resolvePkgPath(v.Path, conf)
	default:
		return "", fmt.Errorf("unsupported project type")
	}

	return filepath.Abs(dir)
}

// loadCustomRunner loads the custom runner from gop.mod in the project directory
func loadCustomRunner(projDir string) *modfile.Runner {
	// Try gop.mod first
	gopMod := filepath.Join(projDir, "gop.mod")
	if data, err := os.ReadFile(gopMod); err == nil {
		if opt, err := modfile.ParseLax(gopMod, data, nil); err == nil {
			if len(opt.Projects) > 0 && opt.Projects[0].Runner != nil {
				return opt.Projects[0].Runner
			}
		}
	}

	// Fallback to xgomod.Load (requires go.mod)
	mod, err := xgomod.Load(projDir)
	if err != nil || mod.Opt == nil || len(mod.Opt.Projects) == 0 {
		return nil
	}
	return mod.Opt.Projects[0].Runner
}

// executeRunner executes the custom runner command
func executeRunner(runner *modfile.Runner, projDir string, args []string) error {
	cmdPath, err := ensureRunnerBinary(runner)
	if err != nil {
		return err
	}

	cmd := exec.Command(cmdPath, append([]string{projDir}, args...)...)
	cmd.Dir = projDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ensureRunnerBinary ensures the runner binary exists, building it if necessary
func ensureRunnerBinary(runner *modfile.Runner) (string, error) {
	cmdName := filepath.Base(runner.Path)

	// Try to find in PATH
	if path, err := exec.LookPath(cmdName); err == nil {
		return path, nil
	}

	// Try GOPATH/bin
	binPath := filepath.Join(getGoPath(), "bin", cmdName)
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	// Build the runner
	return buildRunner(runner, cmdName)
}

// buildRunner builds the runner binary and returns its path
func buildRunner(runner *modfile.Runner, cmdName string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "xgo-runner-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize temporary go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module temp\n\ngo 1.18\n"), 0644); err != nil {
		return "", fmt.Errorf("create go.mod: %w", err)
	}

	// Determine version
	pkgSpec := runner.Path
	if runner.Version != "" {
		pkgSpec += "@" + runner.Version
	} else {
		pkgSpec += "@latest"
	}

	// Fetch the package
	fmt.Printf("Fetching %s...\n", pkgSpec)
	if err := runCommand(tmpDir, "go", "get", pkgSpec); err != nil {
		return "", fmt.Errorf("fetch %s: %w", pkgSpec, err)
	}

	// Build the binary
	outPath := filepath.Join(getGoPath(), "bin", cmdName)
	fmt.Printf("Building %s to %s...\n", runner.Path, outPath)
	if err := runCommand(tmpDir, "go", "build", "-o", outPath, runner.Path); err != nil {
		return "", fmt.Errorf("build %s: %w", runner.Path, err)
	}

	return outPath, nil
}

// resolvePkgPath resolves a package path to its local directory
func resolvePkgPath(pkgPath string, conf *tool.Config) (string, error) {
	// Strip /... suffix
	if strings.HasSuffix(pkgPath, "/...") {
		return "", fmt.Errorf("cannot use /... pattern for custom runner")
	}

	// Handle versioned packages
	if strings.Contains(pkgPath, "@") {
		return resolveRemotePkg(pkgPath)
	}

	// Try local module first
	if mod, err := xgomod.Load(""); err == nil {
		if pkg, err := mod.Lookup(pkgPath); err == nil {
			return pkg.Dir, nil
		}
	}

	// Fallback to remote package
	return resolveRemotePkg(pkgPath)
}

// resolveRemotePkg resolves a remote package to its local cache directory
func resolveRemotePkg(pkgPath string) (string, error) {
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

// getGoPath returns the GOPATH, defaulting to ~/go
func getGoPath() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath
	}
	return filepath.Join(os.Getenv("HOME"), "go")
}

// runCommand executes a command in the specified directory
func runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
