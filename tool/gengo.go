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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/goplus/gogen"
	"github.com/goplus/mod/modcache"
	"github.com/goplus/mod/modfetch"
	"github.com/goplus/mod/xgomod"
	"github.com/qiniu/x/errors"
)

const (
	testingGoFile    = "_test"
	autoGenFile      = "xgo_autogen.go"
	autoGenTestFile  = "xgo_autogen_test.go"
	autoGen2TestFile = "xgo_autogen2_test.go"
	autoGenPrefix    = "xgo_autogen_"
)

type GenFlags int

const (
	GenFlagCheckOnly GenFlags = 1 << iota
	GenFlagSingleFile
	GenFlagMultiFiles
	GenFlagPrintError
	GenFlagPrompt
)

// -----------------------------------------------------------------------------

// GenGo generates xgo_autogen.go for an XGo package directory.
func GenGo(dir string, conf *Config, genTestPkg bool) (string, bool, error) {
	return GenGoEx(dir, conf, genTestPkg, 0)
}

// GenGoEx generates xgo_autogen.go for an XGo package directory.
func GenGoEx(dir string, conf *Config, genTestPkg bool, flags GenFlags) (string, bool, error) {
	recursively := strings.HasSuffix(dir, "/...")
	if recursively {
		dir = dir[:len(dir)-4]
	}
	return dir, recursively, genGoDir(dir, conf, genTestPkg, recursively, flags)
}

func genGoDir(dir string, conf *Config, genTestPkg, recursively bool, flags GenFlags) (err error) {
	if conf == nil {
		conf = new(Config)
	}
	if recursively {
		var (
			list errors.List
			fn   func(path string, d fs.DirEntry, err error) error
		)
		if flags&GenFlagSingleFile != 0 {
			fn = func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				return genGoEntry(&list, path, d, conf, flags)
			}
		} else {
			fn = func(path string, d fs.DirEntry, err error) error {
				if err == nil && d.IsDir() {
					if strings.HasPrefix(d.Name(), "_") || (path != dir && hasMod(path)) { // skip _
						return filepath.SkipDir
					}
					if e := genGoIn(path, conf, genTestPkg, flags); e != nil && notIgnNotated(e, conf) {
						if flags&GenFlagPrintError != 0 {
							fmt.Fprintln(os.Stderr, e)
						}
						list.Add(e)
					}
				}
				return err
			}
		}
		err = filepath.WalkDir(dir, fn)
		if err != nil {
			return errors.NewWith(err, `filepath.WalkDir(dir, fn)`, -2, "filepath.WalkDir", dir, fn)
		}
		return list.ToError()
	}
	if flags&GenFlagSingleFile != 0 {
		var list errors.List
		var entries, e = os.ReadDir(dir)
		if e != nil {
			return errors.NewWith(e, `os.ReadDir(dir)`, -2, "os.ReadDir", dir)
		}
		for _, d := range entries {
			genGoEntry(&list, filepath.Join(dir, d.Name()), d, conf, flags)
		}
		return list.ToError()
	}
	if e := genGoIn(dir, conf, genTestPkg, flags); e != nil && notIgnNotated(e, conf) {
		if (flags & GenFlagPrintError) != 0 {
			fmt.Fprintln(os.Stderr, e)
		}
		err = e
	}
	return
}

func hasMod(dir string) bool {
	_, err := os.Lstat(dir + "/go.mod")
	return err == nil
}

func notIgnNotated(e error, conf *Config) bool {
	return !(conf != nil && conf.IgnoreNotatedError && IgnoreNotated(e))
}

func genGoEntry(list *errors.List, path string, d fs.DirEntry, conf *Config, flags GenFlags) error {
	fname := d.Name()
	if strings.HasPrefix(fname, "_") { // skip _
		if d.IsDir() {
			return filepath.SkipDir
		}
	} else if !d.IsDir() && (strings.HasSuffix(fname, ".xgo") || strings.HasSuffix(fname, ".gop")) {
		if e := genGoSingleFile(path, 4, conf, flags); e != nil && notIgnNotated(e, conf) {
			if flags&GenFlagPrintError != 0 {
				fmt.Fprintln(os.Stderr, e)
			}
			list.Add(e)
		}
	}
	return nil
}

func genGoSingleFile(file string, extn int, conf *Config, flags GenFlags) (err error) {
	dir, fname := filepath.Split(file)
	autogen := dir + fname[:len(fname)-extn] + "_autogen.go"
	if (flags & GenFlagPrompt) != 0 {
		fmt.Fprintln(os.Stderr, "GenGo", file, "...")
	}
	out, err := LoadFiles(".", []string{file}, conf)
	if err != nil {
		return errors.NewWith(err, `LoadFiles(files, conf)`, -2, "tool.LoadFiles", file)
	}
	if flags&GenFlagCheckOnly != 0 {
		return nil
	}
	if err := out.WriteFile(autogen); err != nil {
		return errors.NewWith(err, `out.WriteFile(autogen)`, -2, "(*gogen.Package).WriteFile", out, autogen)
	}
	return nil
}

func genGoIn(dir string, conf *Config, genTestPkg bool, flags GenFlags, gen ...*bool) (err error) {
	out, test, err := LoadDir(dir, conf, genTestPkg, (flags&GenFlagPrompt) != 0, (flags&GenFlagMultiFiles) != 0)
	if err != nil {
		if NotFound(err) { // no XGo source files
			return nil
		}
		return errors.NewWith(err, `LoadDir(dir, conf, genTestPkg)`, -5, "tool.LoadDir", dir, conf, genTestPkg)
	}
	if flags&GenFlagCheckOnly != 0 {
		return nil
	}
	os.MkdirAll(dir, 0755)

	if flags&GenFlagMultiFiles != 0 {
		err = writeMultiFiles(out, dir)
		if err != nil {
			return errors.NewWith(err, `writeMultiFiles(out, dir)`, -2, "(*gogen.Package).WriteFile", out, dir)
		}
	} else {
		file := filepath.Join(dir, autoGenFile)
		err = out.WriteFile(file)
		if err != nil {
			return errors.NewWith(err, `out.WriteFile(file)`, -2, "(*gogen.Package).WriteFile", out, file)
		}
	}
	if gen != nil { // say `xgo_autogen.go generated`
		*gen[0] = true
	}

	testFile := filepath.Join(dir, autoGenTestFile)
	err = out.WriteFile(testFile, testingGoFile)
	if err != nil && err != syscall.ENOENT {
		return errors.NewWith(err, `out.WriteFile(testFile, testingGoFile)`, -2, "(*gogen.Package).WriteFile", out, testFile, testingGoFile)
	}

	if test != nil {
		testFile = filepath.Join(dir, autoGen2TestFile)
		err = test.WriteFile(testFile, testingGoFile)
		if err != nil {
			return errors.NewWith(err, `test.WriteFile(testFile, testingGoFile)`, -2, "(*gogen.Package).WriteFile", test, testFile, testingGoFile)
		}
	} else {
		err = nil
	}
	return
}

func writeMultiFiles(pkg *gogen.Package, dir string) error {
	names := make(map[string]string)
	pkg.ForEachFile(func(fname string, file *gogen.File) {
		if fname == "" {
			names[fname] = autoGenFile
		} else {
			_, name := filepath.Split(fname)
			name = name[:len(name)-len(filepath.Ext(name))]
			if strings.HasSuffix(name, "_test") {
				return
			}
			names[fname] = autoGenPrefix + name + ".go"
		}
	})
	for fname, file := range names {
		err := pkg.WriteFile(filepath.Join(dir, file), fname)
		if err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

const (
	modWritable = 0755
	modReadonly = 0555
)

// GenGoPkgPath generates xgo_autogen.go for an XGo package.
func GenGoPkgPath(workDir, pkgPath string, conf *Config, allowExtern bool) (localDir string, recursively bool, err error) {
	return GenGoPkgPathEx(workDir, pkgPath, conf, allowExtern, 0)
}

func remotePkgPath(pkgPath string, conf *Config, recursively bool, flags GenFlags) (localDir string, _recursively bool, err error) {
	remotePkgPathDo(pkgPath, func(dir, _ string) {
		os.Chmod(dir, modWritable)
		defer os.Chmod(dir, modReadonly)
		localDir = dir
		_recursively = recursively
		err = genGoDir(dir, conf, false, recursively, flags)
	}, func(e error) {
		err = e
	})
	return
}

// GenGoPkgPathEx generates xgo_autogen.go for an XGo package.
func GenGoPkgPathEx(workDir, pkgPath string, conf *Config, allowExtern bool, flags GenFlags) (localDir string, recursively bool, err error) {
	recursively = strings.HasSuffix(pkgPath, "/...")
	if recursively {
		pkgPath = pkgPath[:len(pkgPath)-4]
	} else if allowExtern && strings.Contains(pkgPath, "@") {
		return remotePkgPath(pkgPath, conf, false, flags)
	}

	mod, err := xgomod.Load(workDir)
	if NotFound(err) && allowExtern {
		return remotePkgPath(pkgPath, conf, recursively, flags)
	} else if err != nil {
		return
	}

	pkg, err := mod.Lookup(pkgPath)
	if err != nil {
		return
	}
	localDir = pkg.Dir
	if pkg.Type == xgomod.PkgtExtern {
		os.Chmod(localDir, modWritable)
		defer os.Chmod(localDir, modReadonly)
	}
	err = genGoDir(localDir, conf, false, recursively, flags)
	return
}

func remotePkgPathDo(pkgPath string, doSth func(pkgDir, modDir string), onErr func(e error)) {
	modVer, leftPart, err := modfetch.GetPkg(pkgPath, "")
	if err != nil {
		onErr(err)
	} else if dir, err := modcache.Path(modVer); err != nil {
		onErr(err)
	} else {
		doSth(filepath.Join(dir, leftPart), dir)
	}
}

// -----------------------------------------------------------------------------

// GenGoFiles generates xgo_autogen.go for specified XGo files.
func GenGoFiles(autogen string, files []string, conf *Config) (outFiles []string, err error) {
	if conf == nil {
		conf = new(Config)
	}
	if autogen == "" {
		autogen = "xgo_autogen.go"
		if len(files) == 1 {
			file := files[0]
			srcDir, fname := filepath.Split(file)
			if hasMultiXgoFiles(srcDir) {
				autogen = filepath.Join(srcDir, "xgo_autogen_"+fname+".go")
			}
		}
	}
	out, err := LoadFiles(".", files, conf)
	if err != nil {
		err = errors.NewWith(err, `LoadFiles(files, conf)`, -2, "tool.LoadFiles", files, conf)
		return
	}
	err = out.WriteFile(autogen)
	if err != nil {
		err = errors.NewWith(err, `out.WriteFile(autogen)`, -2, "(*gogen.Package).WriteFile", out, autogen)
	}
	outFiles = []string{autogen}
	return
}

func hasMultiXgoFiles(srcDir string) bool {
	var has bool
	if f, err := os.Open(srcDir); err == nil {
		defer f.Close()
		fis, _ := f.ReadDir(-1)
		for _, fi := range fis {
			ext := filepath.Ext(fi.Name())
			if !fi.IsDir() && (ext == ".xgo" || ext == ".gop") {
				if has {
					return true
				}
				has = true
			}
		}
	}
	return false
}

// -----------------------------------------------------------------------------
