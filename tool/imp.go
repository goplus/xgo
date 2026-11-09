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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goplus/gogen/packages"
	"github.com/goplus/gogen/packages/cache"
	"github.com/goplus/mod"
	"github.com/goplus/mod/env"
	"github.com/goplus/mod/goxmod"
	"github.com/goplus/mod/modfetch"
	"github.com/goplus/mod/modfile"
)

// -----------------------------------------------------------------------------

// Importer represents an XGo importer.
type Importer struct {
	impFrom *packages.Importer
	ws      *Workspace // the caller's workspace: answers Classes(dir)
	res     *Workspace // resolves imports: ws, or the XGo-root fallback
	xgo     *env.XGo
	fset    *token.FileSet

	Flags GenFlags // can change this for loading XGo modules

	importStack map[string]bool
	generating  map[string]bool // package dirs whose xgo_autogen.go is being generated
}

// NewImporter creates an XGo Importer. w may be nil, in which case the default
// workspace for the current directory is used.
func NewImporter(w *Workspace, xgo *env.XGo, fset *token.FileSet) *Importer {
	const (
		defaultFlags = GenFlagPrompt | GenFlagPrintError
	)
	if w == nil {
		w, _ = LoadWorkspace(".")
	}
	// Fallback keeps its meaning: a default workspace (no module information at
	// all) borrows the XGo installation's own go.mod so that loose scripts can
	// still resolve what the toolchain itself depends on. This is only for
	// import resolution; the caller's workspace still answers Classes(dir), so
	// the XGo-root go.mod never leaks its classes into user code.
	res := w
	if w.IsDefault() {
		if gw, err := loadGoModWorkspace(filepath.Join(xgo.Root, "go.mod")); err == nil {
			res = gw
		}
	}
	impFrom := packages.NewImporter(fset, res.Dir())
	ret := &Importer{
		ws: w, res: res, xgo: xgo, impFrom: impFrom, fset: fset,
		Flags:       defaultFlags,
		importStack: make(map[string]bool),
		generating:  make(map[string]bool),
	}
	impFrom.SetCache(cache.New(ret.PkgHash))
	return ret
}

// loadGoModWorkspace loads the workspace for a single go.mod file (the XGo-root
// fallback), wrapped with its directory as the resolution base.
func loadGoModWorkspace(gomod string) (*Workspace, error) {
	w, err := mod.LoadGoMod(gomod)
	if err != nil {
		return nil, err
	}
	return newWorkspace(w, filepath.Dir(gomod), "", gomod), nil
}

func (p *Importer) SetTags(tags string) {
	p.impFrom.SetTags(tags)
	if c, ok := p.impFrom.Cache().(*cache.Impl); ok {
		c.SetTags(tags)
	}
}

// CacheFile returns file path of the cache.
func (p *Importer) CacheFile() string {
	cacheDir, _ := os.UserCacheDir()
	cacheDir += "/xgo-build/"
	os.MkdirAll(cacheDir, 0755)

	fname := ""
	h := sha256.New()
	io.WriteString(h, runtime.Version())
	if root := p.res.Dir(); root != "" {
		io.WriteString(h, root)
		fname = filepath.Base(root)
	}
	// GOWORK can point two workspaces at the same directory, so the go.work path
	// (when there is one) also feeds the hash.
	if gowork := p.res.gowork; gowork != "" {
		io.WriteString(h, gowork)
	}
	if tags := p.impFrom.Tags(); tags != "" {
		io.WriteString(h, tags)
	}
	hash := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return cacheDir + hash + fname
}

// Cache returns the cache object.
func (p *Importer) Cache() *cache.Impl {
	return p.impFrom.Cache().(*cache.Impl)
}

// PkgHash calculates hash value for a package.
// It is required by cache.New func.
func (p *Importer) PkgHash(pkgPath string, self bool) string {
	if pkg, e := p.res.Lookup(pkgPath); e == nil {
		switch pkg.Type {
		case mod.PkgtStandard:
			return cache.HashSkip
		case mod.PkgtExtern: // immutable: module cache version
			return pkg.Real.String()
		case mod.PkgtLocal: // can change: workspace module or local replace
			return p.dirHash(pkg.Dir, self)
		}
	}
	if isPkgInMod(pkgPath, xgoMod) || isPkgInMod(pkgPath, xMod) {
		return p.xgo.Version
	}
	log.Println("PkgHash: unexpected package -", pkgPath)
	return cache.HashInvalid
}

const (
	xgoMod = "github.com/goplus/xgo"
	xMod   = "github.com/qiniu/x"
)

// Import imports a Go/XGo package.
func (p *Importer) Import(pkgPath string) (pkg *types.Package, err error) {
	if p.importStack[pkgPath] {
		return nil, fmt.Errorf("cycle import detected: package %s imports itself", pkgPath)
	}
	p.importStack[pkgPath] = true
	defer delete(p.importStack, pkgPath)
	if strings.HasPrefix(pkgPath, xgoMod) {
		if suffix := pkgPath[len(xgoMod):]; suffix == "" || suffix[0] == '/' {
			xgoRoot := p.xgo.Root
			if suffix == "/cl/internal/gop-in-go/foo" { // for test github.com/goplus/xgo/cl
				if err = p.genGoExtern(xgoRoot+suffix, false); err != nil {
					return
				}
			}
			return p.impFrom.ImportFrom(pkgPath, xgoRoot, 0)
		}
	}
	if isPkgInMod(pkgPath, xMod) {
		return p.impFrom.ImportFrom(pkgPath, p.xgo.Root, 0)
	}
	ret, e := p.res.Lookup(pkgPath)
	if e != nil {
		// Defer to `go list` (which reports the "not in std" error) when either
		// the caller has no module information at all (a loose script), or the
		// path looks like a standard package (its first element has no dot) that
		// the module graph did not resolve. The latter preserves the old
		// classifier, which treated every non-dotted path as standard.
		if p.ws.IsDefault() || looksStandard(pkgPath) {
			return p.impFrom.Import(pkgPath)
		}
		return nil, e
	}
	switch ret.Type {
	case mod.PkgtStandard:
		return p.impFrom.ImportFrom(pkgPath, p.xgo.Root, 0)
	case mod.PkgtExtern:
		if _, err = modfetch.Get(ret.Real.String()); err != nil {
			return
		}
		modDir := ret.ModDir
		goModfile := filepath.Join(modDir, "go.mod")
		if _, e := os.Lstat(goModfile); e != nil { // no go.mod
			os.Chmod(modDir, modWritable)
			defer os.Chmod(modDir, modReadonly)
			os.WriteFile(goModfile, defaultGoMod(ret.ModPath), 0644)
		}
		return p.impFrom.ImportFrom(pkgPath, modDir, 0)
	case mod.PkgtLocal:
		// Do not regenerate a directory whose generation is in progress (this
		// replaces the old "do not regenerate the root package of the main
		// module" guard, which cannot be expressed with several main modules).
		if !p.generating[cleanDir(ret.Dir)] {
			if err = p.genGoExtern(ret.Dir, false); err != nil {
				return
			}
		}
	}
	return p.impFrom.Import(pkgPath)
}

func (p *Importer) genGoExtern(dir string, isExtern bool) (err error) {
	genfile := filepath.Join(dir, autoGenFile)
	if _, err = os.Lstat(genfile); err != nil { // no xgo_autogen.go
		if isExtern {
			os.Chmod(dir, modWritable)
			defer os.Chmod(dir, modReadonly)
		}
		gen := false
		err = genGoIn(dir, &Config{XGo: p.xgo, Importer: p, Fset: p.fset, Workspace: p.ws}, false, p.Flags, &gen)
		if err != nil {
			return
		}
		if gen {
			if p.shouldTidy() {
				cmd := exec.Command("go", "mod", "tidy")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Dir = dir
				err = cmd.Run()
			} else {
				// In a go.work workspace the generated directory may depend on a
				// sibling with no require line; `go mod tidy` ignores go.work and
				// would try to fetch that sibling. Leave it to the user.
				fmt.Fprintln(os.Stderr, autoGenFile, "generated in", dir,
					"- run `go work sync` (or `go mod tidy` in that module) if its requirements changed")
			}
		}
	}
	return
}

// shouldTidy reports whether `go mod tidy` is safe to run after generating a
// directory: only when no go.work is in effect, so the module is self-contained
// as it was before workspaces.
func (p *Importer) shouldTidy() bool {
	return !p.res.HasGoWork()
}

func isPkgInMod(pkgPath, modPath string) bool {
	if strings.HasPrefix(pkgPath, modPath) {
		suffix := pkgPath[len(modPath):]
		return suffix == "" || suffix[0] == '/'
	}
	return false
}

// looksStandard reports whether pkgPath could be a standard-library package:
// its first path element has no dot. The go command treats such a path as
// standard; a dotted first element is a module path. This keeps the old
// classifier's behavior of routing non-dotted, unresolved paths to `go list`.
func looksStandard(pkgPath string) bool {
	elem := pkgPath
	if i := strings.IndexByte(elem, '/'); i >= 0 {
		elem = elem[:i]
	}
	return elem != "" && !strings.Contains(elem, ".")
}

func defaultGoMod(modPath string) []byte {
	return []byte(`module ` + modPath + `

go 1.16
`)
}

func (p *Importer) dirHash(dir string, self bool) string {
	cls, _ := classesOf(p.ws, dir)
	return dirHash(cls, p.xgo, dir, self)
}

func dirHash(cls *goxmod.Classes, xgo *env.XGo, dir string, self bool) string {
	h := sha256.New()
	if self {
		fmt.Fprintf(h, "go\t%s\n", runtime.Version())
		fmt.Fprintf(h, "xgo\t%s\n", xgo.Version)
	}
	if fis, err := os.ReadDir(dir); err == nil {
		for _, fi := range fis {
			if fi.IsDir() {
				continue
			}
			fname := fi.Name()
			if strings.HasPrefix(fname, "_") || !canCl(cls, fname) {
				continue
			}
			if v, e := fi.Info(); e == nil {
				fmt.Fprintf(h, "file\t%s\t%x\t%x\n", fname, v.Size(), v.ModTime().UnixNano())
			}
		}
	}
	return base64.RawStdEncoding.EncodeToString(h.Sum(nil))
}

// canCl reports whether fname is an XGo source file, reading the classfile set
// of the directory's own module. When cls is nil (the classfile set could not
// be determined) only the fixed extensions count; the error surfaces when that
// directory is actually compiled.
func canCl(cls *goxmod.Classes, fname string) bool {
	switch path.Ext(fname) {
	case ".go", ".xgo", ".gop", ".gox":
		return true
	}
	return cls != nil && cls.IsClass(modfile.ClassExt(fname))
}

// -----------------------------------------------------------------------------
