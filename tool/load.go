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
	"path"
	"path/filepath"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/mod"
	"github.com/goplus/mod/env"
	"github.com/goplus/mod/goxmod"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
	"github.com/goplus/xgo/x/gocmd"
	"github.com/goplus/xgo/x/xgoenv"
	"github.com/qiniu/x/errors"
)

var (
	ErrNotFound      = mod.ErrNoModule
	ErrIgnoreNotated = errors.New("notated error ignored")
)

// NotFound returns if cause err is ErrNotFound or not
func NotFound(err error) bool {
	return errors.Err(err) == mod.ErrNoModule
}

// IgnoreNotated returns if cause err is ErrIgnoreNotated or not.
func IgnoreNotated(err error) bool {
	return errors.Err(err) == ErrIgnoreNotated
}

// ErrorPos returns where the error occurs.
func ErrorPos(err error) token.Pos {
	switch v := err.(type) {
	case *gogen.CodeError:
		return v.Pos
	case *gogen.MatchError:
		return v.Pos()
	case *gogen.ImportError:
		return v.Pos
	}
	return token.NoPos
}

func ignNotatedErrs(err error, pkg *ast.Package, fset *token.FileSet) error {
	switch v := err.(type) {
	case errors.List:
		var ret errors.List
		for _, e := range v {
			if isNotatedErr(e, pkg, fset) {
				continue
			}
			ret = append(ret, e)
		}
		if len(ret) == 0 {
			return ErrIgnoreNotated
		}
		return ret
	default:
		if isNotatedErr(err, pkg, fset) {
			return ErrIgnoreNotated
		}
		return err
	}
}

func isNotatedErr(err error, pkg *ast.Package, fset *token.FileSet) (notatedErr bool) {
	pos := ErrorPos(err)
	f := fset.File(pos)
	if f == nil {
		return
	}
	xgof, ok := pkg.Files[f.Name()]
	if !ok {
		return
	}
	lines := token.Lines(f)
	i := f.Line(pos) - 1 // base 0
	start := lines[i]
	var end int
	if i+1 < len(lines) {
		end = lines[i+1]
	} else {
		end = f.Size()
	}
	text := string(xgof.Code[start:end])
	commentOff := strings.Index(text, "//")
	if commentOff < 0 {
		return
	}
	return strings.Contains(text[commentOff+2:], "compile error:")
}

// -----------------------------------------------------------------------------

// Config represents a configuration for loading XGo packages.
type Config struct {
	XGo  *env.XGo
	Fset *token.FileSet

	// Workspace is the caller's workspace for this run. It resolves imports
	// (for generation entry points) and answers Classes(dir) for the directory
	// being compiled or hashed. If nil, the workspace is discovered from the
	// directory being loaded.
	Workspace *Workspace

	Importer *Importer

	Filter func(fs.FileInfo) bool

	// If not nil, it is used for returning result of checks XGo dependencies.
	// see https://pkg.go.dev/github.com/goplus/gogen#File.CheckGopDeps
	XGoDeps *int

	// CacheFile specifies the file path of the cache.
	CacheFile string

	IgnoreNotatedError bool
	DontUpdateGoMod    bool
}

// ConfFlags represents configuration flags.
type ConfFlags int

const (
	ConfFlagIgnoreNotatedError ConfFlags = 1 << iota
	ConfFlagDontUpdateGoMod
	ConfFlagNoTestFiles
	ConfFlagNoCacheFile
)

// NewDefaultConf creates a dfault configuration for common cases.
func NewDefaultConf(dir string, flags ConfFlags, tags ...string) (conf *Config, err error) {
	w, err := LoadWorkspace(dir)
	if err != nil {
		return
	}
	xgo := xgoenv.Get()
	fset := token.NewFileSet()
	imp := NewImporter(w, xgo, fset)
	if len(tags) > 0 {
		imp.SetTags(strings.Join(tags, ","))
	}
	conf = &Config{
		XGo: xgo, Fset: fset, Workspace: w, Importer: imp,
		IgnoreNotatedError: flags&ConfFlagIgnoreNotatedError != 0,
		DontUpdateGoMod:    flags&ConfFlagDontUpdateGoMod != 0,
	}
	if flags&ConfFlagNoCacheFile == 0 {
		conf.CacheFile = imp.CacheFile()
		imp.Cache().Load(conf.CacheFile)
	}
	if flags&ConfFlagNoTestFiles != 0 {
		conf.Filter = FilterNoTestFiles
	}
	return
}

func (conf *Config) NewGoCmdConf() *gocmd.Config {
	return &gocmd.Config{
		XGo: conf.XGo,
	}
}

// UpdateCache updates the cache.
func (conf *Config) UpdateCache(verbose ...bool) {
	if conf.CacheFile != "" {
		c := conf.Importer.Cache()
		c.Save(conf.CacheFile)
		if verbose != nil && verbose[0] {
			fmt.Println("Times of calling go list:", c.ListTimes())
		}
	}
}

// LoadMod loads the workspace that owns a specified directory. It never fails
// with "no module": a loose directory yields a default workspace.
//
// Deprecated: use LoadWorkspace.
func LoadMod(dir string) (w *Workspace, err error) {
	return LoadWorkspace(dir)
}

// FilterNoTestFiles filters to skip all testing files.
func FilterNoTestFiles(fi fs.FileInfo) bool {
	fname := fi.Name()
	suffix := ""
	switch path.Ext(fname) {
	case ".gox":
		suffix = "test.gox"
	case ".xgo":
		suffix = "_test.xgo"
	case ".gop":
		suffix = "_test.gop"
	case ".go":
		suffix = "_test.go"
	default:
		return true
	}
	return !strings.HasSuffix(fname, suffix)
}

// -----------------------------------------------------------------------------

// LoadDir loads XGo packages from a specified directory.
func LoadDir(dir string, conf *Config, genTestPkg bool, promptGenGo ...bool) (out, test *gogen.Package, err error) {
	if conf == nil {
		conf = new(Config)
	}

	w := conf.Workspace
	if w == nil {
		if w, err = LoadWorkspace(dir); err != nil {
			err = errors.NewWith(err, `LoadWorkspace(dir)`, -2, "tool.LoadWorkspace", dir)
			return
		}
	}
	// The classfiles that apply to this directory belong to the module that
	// owns it, not to the workspace as a whole.
	cls, err := classesOf(w, dir)
	if err != nil {
		err = errors.NewWith(err, `classesOf(w, dir)`, -2, "tool.classesOf", w, dir)
		return
	}

	fset := conf.Fset
	if fset == nil {
		fset = token.NewFileSet()
	}
	pkgs, err := parser.ParseDirEx(fset, dir, parser.Config{
		ClassInfo: cls.ClassInfo,
		Filter:    conf.Filter,
		Mode:      parser.ParseComments | parser.SaveAbsFile,
	})
	if err != nil {
		return
	}
	if len(pkgs) == 0 {
		return nil, nil, ErrNotFound
	}

	xgo := conf.XGo
	if xgo == nil {
		xgo = xgoenv.Get()
	}
	imp := conf.Importer
	if imp == nil {
		imp = NewImporter(w, xgo, fset)
	}

	var pkgTest *ast.Package
	var clConf = &cl.Config{
		Fset:         fset,
		RelativeBase: relativeBaseOf(w),
		Importer:     imp,
		LookupClass:  cls.LookupClass,
	}

	for name, pkg := range pkgs {
		if strings.HasSuffix(name, "_test") {
			if pkgTest != nil {
				return nil, nil, ErrMultiTestPackges
			}
			pkgTest = pkg
			continue
		}
		if out != nil {
			return nil, nil, ErrMultiPackges
		}
		if len(pkg.Files) == 0 { // no XGo source files
			continue
		}
		if promptGenGo != nil && promptGenGo[0] {
			fmt.Fprintln(os.Stderr, "GenGo", dir, "...")
		}
		out, err = cl.NewPackage("", pkg, clConf)
		if err != nil {
			if conf.IgnoreNotatedError {
				err = ignNotatedErrs(err, pkg, fset)
			}
			return
		}
	}
	if out == nil {
		return nil, nil, ErrNotFound
	}
	if genTestPkg && pkgTest != nil {
		test, err = cl.NewPackage("", pkgTest, clConf)
	}
	afterLoad(w, dir, xgo, out, test, conf)
	return
}

func afterLoad(w *Workspace, dir string, xgo *env.XGo, out, test *gogen.Package, conf *Config) {
	if out == nil { // compile failed; nothing to do
		return
	}
	if w == nil || w.IsDefault() { // no module: nothing to persist
		if conf.XGoDeps != nil {
			*conf.XGoDeps = checkGopDeps(out)
		}
		return
	}
	gomod := ownerGoMod(w, dir)
	if gomod == "" { // the directory is not owned by a workspace module
		if conf.XGoDeps != nil {
			*conf.XGoDeps = checkGopDeps(out)
		}
		return
	}
	if modPathOf(gomod) == xgoMod { // nothing to do for XGo itself
		return
	}
	updateMod := !conf.DontUpdateGoMod
	if updateMod || conf.XGoDeps != nil {
		flags := checkGopDeps(out)
		if conf.XGoDeps != nil { // for `xgo run`
			*conf.XGoDeps = flags
		}
		if updateMod {
			if test != nil {
				flags |= checkGopDeps(test)
			}
			if flags != 0 {
				saveXGoMod(gomod, xgo, flags)
			}
		}
	}
}

func checkGopDeps(pkg *gogen.Package) (flags int) {
	pkg.ForEachFile(func(fname string, file *gogen.File) {
		flags |= file.CheckXGoDeps(pkg)
	})
	return
}

// ownerGoMod returns the go.mod path of the workspace module that owns dir, or
// "" when no workspace module does (a loose directory or a foreign nested
// module). Ownership follows Classes(dir): if that reports the directory is not
// in the workspace, there is no go.mod to update.
func ownerGoMod(w *Workspace, dir string) string {
	if _, err := w.Classes(dir); err != nil {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	// Walk up from dir looking for the nearest go.mod. Classes(dir) already
	// guaranteed the owning module is a workspace module, so the first go.mod
	// found on the way up is that module's.
	for {
		gomod := filepath.Join(abs, "go.mod")
		if fi, e := os.Stat(gomod); e == nil && !fi.IsDir() {
			return gomod
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

func relativeBaseOf(w *Workspace) string {
	if w != nil && !w.IsDefault() {
		return w.Dir()
	}
	dir, _ := os.Getwd()
	return dir
}

// -----------------------------------------------------------------------------

// LoadFiles loads an XGo package from specified files.
func LoadFiles(dir string, files []string, conf *Config) (out *gogen.Package, err error) {
	if conf == nil {
		conf = new(Config)
	}

	w := conf.Workspace
	if w == nil {
		if w, err = LoadWorkspace(dir); err != nil {
			err = errors.NewWith(err, `LoadWorkspace(dir)`, -2, "tool.LoadWorkspace", dir)
			return
		}
	}
	// The classfiles that apply belong to the module that owns the files; use
	// the directory of the first file (they are all in one package).
	clsDir := dir
	if len(files) > 0 {
		clsDir = filepath.Dir(files[0])
	}
	cls, err := classesOf(w, clsDir)
	if err != nil {
		err = errors.NewWith(err, `classesOf(w, clsDir)`, -2, "tool.classesOf", w, clsDir)
		return
	}

	fset := conf.Fset
	if fset == nil {
		fset = token.NewFileSet()
	}
	pkgs, err := parser.ParseEntries(fset, files, parser.Config{
		ClassInfo: cls.ClassInfo,
		Filter:    conf.Filter,
		Mode:      parser.ParseComments | parser.SaveAbsFile,
	})
	if err != nil {
		err = errors.NewWith(err, `parser.ParseFiles(fset, files, parser.ParseComments)`, -2, "parser.ParseFiles", fset, files, parser.ParseComments)
		return
	}
	if len(pkgs) != 1 {
		err = errors.NewWith(ErrMultiPackges, `len(pkgs) != 1`, -1, "!=", len(pkgs), 1)
		return
	}
	xgo := conf.XGo
	if xgo == nil {
		xgo = xgoenv.Get()
	}
	for _, pkg := range pkgs {
		imp := conf.Importer
		if imp == nil {
			imp = NewImporter(w, xgo, fset)
		}
		clConf := &cl.Config{
			Fset:         fset,
			RelativeBase: relativeBaseOf(w),
			Importer:     imp,
			LookupClass:  cls.LookupClass,
		}
		out, err = cl.NewPackage("", pkg, clConf)
		if err != nil {
			if conf.IgnoreNotatedError {
				err = ignNotatedErrs(err, pkg, fset)
			}
			return
		}
		break
	}
	afterLoad(w, clsDir, xgo, out, nil, conf)
	return
}

// -----------------------------------------------------------------------------

var (
	ErrMultiPackges     = errors.New("multiple packages")
	ErrMultiTestPackges = errors.New("multiple test packages")
)

// -----------------------------------------------------------------------------

// GetFileClassType get xgo module file classType.
//
// Deprecated: Don't use it
func GetFileClassType(cls *goxmod.Classes, file *ast.File, filename string) (classType string, isTest bool) {
	return cl.GetFileClassType(file, filename, cls.LookupClass)
}

// -----------------------------------------------------------------------------
