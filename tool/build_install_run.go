/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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

package tool

import (
	"log"
	"os"

	"github.com/goplus/xgo/x/gocmd"
	"github.com/qiniu/x/errors"
)

func genFlags(flags []GenFlags) GenFlags {
	if flags != nil {
		return flags[0]
	}
	return 0
}

// -----------------------------------------------------------------------------

// InstallDir installs an XGo package directory.
func InstallDir(dir string, conf *Config, install *gocmd.InstallConfig, flags ...GenFlags) (err error) {
	_, _, err = GenGoEx(dir, conf, false, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGo(dir, conf, false)`, -2, "tool.GenGo", dir, conf, false)
	}
	return gocmd.Install(dir, install)
}

// InstallPkgPath installs an XGo package.
func InstallPkgPath(workDir, pkgPath string, conf *Config, install *gocmd.InstallConfig, flags ...GenFlags) (err error) {
	localDir, recursively, err := GenGoPkgPathEx(workDir, pkgPath, conf, true, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGoPkgPath(workDir, pkgPath, conf, true)`, -2, "tool.GenGoPkgPath", workDir, pkgPath, conf, true)
	}
	old := chdir(localDir)
	defer os.Chdir(old)
	return gocmd.Install(cwdParam(recursively), install)
}

func cwdParam(recursively bool) string {
	if recursively {
		return "./..."
	}
	return "."
}

// InstallFiles installs specified XGo files.
func InstallFiles(files []string, conf *Config, install *gocmd.InstallConfig) (err error) {
	files, err = GenGoFiles("", files, conf)
	if err != nil {
		return errors.NewWith(err, `GenGoFiles("", files, conf)`, -2, "tool.GenGoFiles", "", files, conf)
	}
	return gocmd.InstallFiles(files, install)
}

func chdir(dir string) string {
	old, err := os.Getwd()
	if err != nil {
		log.Panicln(err)
	}
	err = os.Chdir(dir)
	if err != nil {
		log.Panicln(err)
	}
	return old
}

// -----------------------------------------------------------------------------

// BuildDir builds an XGo package directory.
func BuildDir(dir string, conf *Config, build *gocmd.BuildConfig, flags ...GenFlags) (err error) {
	_, _, err = GenGoEx(dir, conf, false, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGo(dir, conf, false)`, -2, "tool.GenGo", dir, conf, false)
	}
	return gocmd.Build(dir, build)
}

// BuildPkgPath builds an XGo package.
func BuildPkgPath(workDir, pkgPath string, conf *Config, build *gocmd.BuildConfig, flags ...GenFlags) (err error) {
	localDir, recursively, err := GenGoPkgPathEx(workDir, pkgPath, conf, false, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGoPkgPath(workDir, pkgPath, conf, false)`, -2, "tool.GenGoPkgPath", workDir, pkgPath, conf, false)
	}
	old, mod := chdirAndMod(localDir)
	defer restoreDirAndMod(old, mod)
	return gocmd.Build(cwdParam(recursively), build)
}

// BuildFiles builds specified XGo files.
func BuildFiles(files []string, conf *Config, build *gocmd.BuildConfig) (err error) {
	files, err = GenGoFiles("", files, conf)
	if err != nil {
		return errors.NewWith(err, `GenGoFiles("", files, conf)`, -2, "tool.GenGoFiles", "", files, conf)
	}
	return gocmd.BuildFiles(files, build)
}

func chdirAndMod(dir string) (old string, mod os.FileMode) {
	mod = 0755
	if info, err := os.Stat(dir); err == nil {
		mod = info.Mode().Perm()
	}
	os.Chmod(dir, 0777)
	old = chdir(dir)
	return
}

func restoreDirAndMod(old string, mod os.FileMode) {
	os.Chmod(".", mod)
	os.Chdir(old)
}

// -----------------------------------------------------------------------------

// If no go.mod and used XGo, use XGOROOT as buildDir.
func getBuildDir(conf *Config) string {
	if conf != nil && conf.XGoDeps != nil && *conf.XGoDeps != 0 {
		return conf.XGo.Root
	}
	return ""
}

// RunDir runs an application from an XGo package directory.
func RunDir(dir string, args []string, conf *Config, run *gocmd.RunConfig, flags ...GenFlags) (err error) {
	_, _, err = GenGoEx(dir, conf, false, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGo(dir, conf, false)`, -2, "tool.GenGo", dir, conf, false)
	}
	return gocmd.RunDir(getBuildDir(conf), dir, args, run)
}

// RunPkgPath runs an application from an XGo package.
func RunPkgPath(pkgPath string, args []string, chDir bool, conf *Config, run *gocmd.RunConfig, flags ...GenFlags) (err error) {
	localDir, recursively, err := GenGoPkgPathEx("", pkgPath, conf, true, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGoPkgPath("", pkgPath, conf, true)`, -2, "tool.GenGoPkgPath", "", pkgPath, conf, true)
	}
	if recursively {
		return errors.NewWith(errors.New("can't use ... pattern for `xgo run` command"), `recursively`, -1, "", recursively)
	}
	if chDir {
		old := chdir(localDir)
		defer os.Chdir(old)
		localDir = "."
	}
	return gocmd.RunDir("", localDir, args, run)
}

// RunFiles runs an application from specified XGo files.
func RunFiles(autogen string, files []string, args []string, conf *Config, run *gocmd.RunConfig) (err error) {
	files, err = GenGoFiles(autogen, files, conf)
	if err != nil {
		return errors.NewWith(err, `GenGoFiles(autogen, files, conf)`, -2, "tool.GenGoFiles", autogen, files, conf)
	}
	return gocmd.RunFiles(getBuildDir(conf), files, args, run)
}

// -----------------------------------------------------------------------------

// TestDir tests an XGo package directory.
func TestDir(dir string, conf *Config, test *gocmd.TestConfig, flags ...GenFlags) (err error) {
	_, _, err = GenGoEx(dir, conf, true, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGo(dir, conf, true)`, -2, "tool.GenGo", dir, conf, true)
	}
	return gocmd.Test(dir, test)
}

// TestPkgPath tests an XGo package.
func TestPkgPath(workDir, pkgPath string, conf *Config, test *gocmd.TestConfig, flags ...GenFlags) (err error) {
	localDir, recursively, err := GenGoPkgPathEx(workDir, pkgPath, conf, false, genFlags(flags))
	if err != nil {
		return errors.NewWith(err, `GenGoPkgPath(workDir, pkgPath, conf, false)`, -2, "tool.GenGoPkgPath", workDir, pkgPath, conf, false)
	}
	old, mod := chdirAndMod(localDir)
	defer restoreDirAndMod(old, mod)
	return gocmd.Test(cwdParam(recursively), test)
}

// TestFiles tests specified XGo files.
func TestFiles(files []string, conf *Config, test *gocmd.TestConfig) (err error) {
	files, err = GenGoFiles("", files, conf)
	if err != nil {
		return errors.NewWith(err, `GenGoFiles("", files, conf)`, -2, "tool.GenGoFiles", "", files, conf)
	}
	return gocmd.TestFiles(files, test)
}

// -----------------------------------------------------------------------------
