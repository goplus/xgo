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

// Package cl compiles XGo syntax trees (ast).
package cl

import (
	"fmt"
	"go/types"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/mod/modfile"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/ast/fromgo"
	"github.com/goplus/xgo/cl/internal/typesalias"
	"github.com/goplus/xgo/token"
	"github.com/qiniu/x/errors"
)

type dbgFlags int

const (
	DbgFlagLoad dbgFlags = 1 << iota
	DbgFlagLookup
	FlagNoMarkAutogen
	DbgFlagAll = DbgFlagLoad | DbgFlagLookup
)

var (
	enableRecover = true
)

var (
	debugLoad     bool
	debugLookup   bool
	noMarkAutogen bool // add const _ = true
)

func SetDisableRecover(disableRecover bool) {
	enableRecover = !disableRecover
}

func SetDebug(flags dbgFlags) {
	debugLoad = (flags & DbgFlagLoad) != 0
	debugLookup = (flags & DbgFlagLookup) != 0
	noMarkAutogen = (flags & FlagNoMarkAutogen) != 0
}

// -----------------------------------------------------------------------------

// Recorder represents a compiling event recorder.
type Recorder interface {
	// Type maps expressions to their types, and for constant
	// expressions, also their values. Invalid expressions are
	// omitted.
	//
	// For (possibly parenthesized) identifiers denoting built-in
	// functions, the recorded signatures are call-site specific:
	// if the call result is not a constant, the recorded type is
	// an argument-specific signature. Otherwise, the recorded type
	// is invalid.
	//
	// The Types map does not record the type of every identifier,
	// only those that appear where an arbitrary expression is
	// permitted. For instance, the identifier f in a selector
	// expression x.f is found only in the Selections map, the
	// identifier z in a variable declaration 'var z int' is found
	// only in the Defs map, and identifiers denoting packages in
	// qualified identifiers are collected in the Uses map.
	Type(ast.Expr, types.TypeAndValue)

	// Instantiate maps identifiers denoting generic types or functions to their
	// type arguments and instantiated type.
	//
	// For example, Instantiate will map the identifier for 'T' in the type
	// instantiation T[int, string] to the type arguments [int, string] and
	// resulting instantiated *Named type. Given a generic function
	// func F[A any](A), Instances will map the identifier for 'F' in the call
	// expression F(int(1)) to the inferred type arguments [int], and resulting
	// instantiated *Signature.
	//
	// Invariant: Instantiating Uses[id].Type() with Instances[id].TypeArgs
	// results in an equivalent of Instances[id].Type.
	Instantiate(*ast.Ident, types.Instance)

	// Def maps identifiers to the objects they define (including
	// package names, dots "." of dot-imports, and blank "_" identifiers).
	// For identifiers that do not denote objects (e.g., the package name
	// in package clauses, or symbolic variables t in t := x.(type) of
	// type switch headers), the corresponding objects are nil.
	//
	// For an embedded field, Def maps the field *Var it defines.
	//
	// Invariant: Defs[id] == nil || Defs[id].Pos() == id.Pos()
	Def(id *ast.Ident, obj types.Object)

	// Use maps identifiers to the objects they denote.
	//
	// For an embedded field, Use maps the *TypeName it denotes.
	//
	// Invariant: Uses[id].Pos() != id.Pos()
	Use(id *ast.Ident, obj types.Object)

	// Implicit maps nodes to their implicitly declared objects, if any.
	// The following node and object types may appear:
	//
	//     node               declared object
	//
	//     *ast.ImportSpec    *PkgName for imports without renames
	//     *ast.CaseClause    type-specific *Var for each type switch case clause (incl. default)
	//     *ast.Field         anonymous parameter *Var (incl. unnamed results)
	//     *ast.FunLit        function literal in *ast.OverloadFuncDecl
	//
	Implicit(node ast.Node, obj types.Object)

	// Select maps selector expressions (excluding qualified identifiers)
	// to their corresponding selections.
	Select(*ast.SelectorExpr, *types.Selection)

	// Scope maps ast.Nodes to the scopes they define. Package scopes are not
	// associated with a specific node but with all files belonging to a package.
	// Thus, the package scope can be found in the type-checked Package object.
	// Scopes nest, with the Universe scope being the outermost scope, enclosing
	// the package scope, which contains (one or more) files scopes, which enclose
	// function scopes which in turn enclose statement and function literal scopes.
	// Note that even though package-level functions are declared in the package
	// scope, the function scopes are embedded in the file scope of the file
	// containing the function declaration.
	//
	// The following node types may appear in Scopes:
	//
	//     *ast.File
	//     *ast.FuncType
	//     *ast.TypeSpec
	//     *ast.BlockStmt
	//     *ast.IfStmt
	//     *ast.SwitchStmt
	//     *ast.TypeSwitchStmt
	//     *ast.CaseClause
	//     *ast.CommClause
	//     *ast.ForStmt
	//     *ast.RangeStmt
	//     *ast.ForPhraseStmt
	//     *ast.ForPhrase
	//     *ast.LambdaExpr
	//     *ast.LambdaExpr2
	//
	Scope(ast.Node, *types.Scope)
}

// -----------------------------------------------------------------------------

type Project = modfile.Project
type Class = modfile.Class

// Config of loading XGo packages.
type Config struct {
	// Types provides type information for the package (optional).
	Types *types.Package

	// Fset provides source position information for syntax trees and types (required).
	Fset *token.FileSet

	// RelativeBase is the root directory of relative path.
	RelativeBase string

	// LookupClass lookups a class by specified file extension (required).
	// See (*github.com/goplus/mod/gopmod.Module).LookupClass.
	LookupClass func(ext string) (c *Project, ok bool)

	// An Importer resolves import paths to Packages (optional).
	Importer types.Importer

	// A Recorder records existing objects including constants, variables and
	// types etc (optional).
	Recorder Recorder

	// NoFileLine = true means not to generate file line comments.
	NoFileLine bool

	// NoAutoGenMain = true means not to auto generate main func is no entry.
	NoAutoGenMain bool

	// NoSkipConstant = true means to disable optimization of skipping constants.
	NoSkipConstant bool

	// Outline = true means to skip compiling function bodies.
	Outline bool
}

type nodeInterp struct {
	fset       *token.FileSet
	files      map[string]*ast.File
	relBaseDir string
}

func (p *nodeInterp) Position(start token.Pos) (pos token.Position) {
	pos = p.fset.Position(start)
	pos.Filename = relFile(p.relBaseDir, pos.Filename)
	return
}

func (p *nodeInterp) Caller(node ast.Node) string {
	if expr, ok := node.(*ast.CallExpr); ok {
		return p.LoadExpr(expr.Fun)
	}
	return "the function call"
}

func (p *nodeInterp) LoadExpr(node ast.Node) string {
	start := node.Pos()
	if start == token.NoPos {
		return ""
	}
	pos := p.fset.Position(start)
	f := p.files[pos.Filename]
	n := int(node.End() - start)
	return string(f.Code[pos.Offset : pos.Offset+n])
}

type loader interface {
	load()
	pos() token.Pos
}

type baseLoader struct {
	fn    func()
	start token.Pos
}

func initLoader(ctx *pkgCtx, syms map[string]loader, start, end token.Pos, name string, fn func(), genBody bool) bool {
	if name == "_" {
		if genBody {
			ctx.inits = append(ctx.inits, fn)
		}
		return false
	}
	if old, ok := syms[name]; ok {
		oldpos := ctx.Position(old.pos())
		ctx.handleErrorf(
			start, end, "%s redeclared in this block\n\tprevious declaration at %v", name, oldpos)
		return false
	}
	syms[name] = &baseLoader{start: start, fn: fn}
	return true
}

func (p *baseLoader) load() {
	p.fn()
}

func (p *baseLoader) pos() token.Pos {
	return p.start
}

type typeLoader struct {
	typ, typInit func()
	methods      []func()
	start        token.Pos
}

func getTypeLoader(ctx *pkgCtx, syms map[string]loader, start, end token.Pos, name string) *typeLoader {
	t, ok := syms[name]
	if ok {
		if start != token.NoPos {
			ld := t.(*typeLoader)
			if ld.start == token.NoPos {
				ld.start = start
			} else {
				ctx.handleErrorf(
					start, end, "%s redeclared in this block\n\tprevious declaration at %v",
					name, ctx.Position(ld.pos()))
			}
			return ld
		}
	} else {
		t = &typeLoader{start: start}
		syms[name] = t
	}
	return t.(*typeLoader)
}

func (p *typeLoader) pos() token.Pos {
	return p.start
}

func (p *typeLoader) load() {
	doNewType(p)
	doInitType(p)
	doInitMethods(p)
}

func doNewType(ld *typeLoader) {
	if typ := ld.typ; typ != nil {
		ld.typ = nil
		typ()
	}
}

func doInitType(ld *typeLoader) {
	if typInit := ld.typInit; typInit != nil {
		ld.typInit = nil
		typInit()
	}
}

func doInitMethods(ld *typeLoader) {
	if methods := ld.methods; methods != nil {
		ld.methods = nil
		for _, method := range methods {
			method()
		}
	}
}

type pkgCtx struct {
	*nodeInterp
	nproj    int                    // number of non-test projects
	projs    map[string]*gmxProject // .gmx => project
	classes  map[*ast.File]*gmxClass
	overpos  map[string]token.Pos // overload => pos
	fset     *token.FileSet
	syms     map[string]loader
	lbinames []any // names that should load before initGopPkg (can be string/func or *ast.Ident/type)
	inits    []func()
	tylds    []*typeLoader
	errs     errors.List

	generics map[string]bool // generic type record
	idents   []*ast.Ident    // toType ident recored
	inInst   int             // toType in generic instance

	goxMainClass string
	goxMain      int32 // normal gox files with main func

	featTypesAlias bool // support types alias
}

type pkgImp struct {
	gogen.PkgRef
	pkgName *types.PkgName
}

type blockCtx struct {
	*pkgCtx
	proj       *gmxProject
	pkg        *gogen.Package
	cb         *gogen.CodeBuilder
	imports    map[string]pkgImp
	autoimps   map[string]pkgImp
	lookups    []gogen.PkgRef
	tlookup    *typeParamLookup
	cstr_      gogen.Ref
	pystr_     gogen.Ref
	relBaseDir string

	classDecl *ast.GenDecl   // available when isClass
	classRecv *ast.FieldList // available when isClass
	baseClass types.Object   // available when isClass

	fileScope *types.Scope // available when isGopFile
	rec       *goxRecorder

	fileLine   bool
	isClass    bool
	isXgoFile  bool // is XGo file or not
	typesAlias bool // support types alias
}

func (p *blockCtx) cstr() gogen.Ref {
	if p.cstr_ == nil {
		p.cstr_ = p.pkg.Import(pathLibc).Ref("Str")
	}
	return p.cstr_
}

func (p *blockCtx) pystr() gogen.Ref {
	if p.pystr_ == nil {
		p.pystr_ = p.pkg.Import(pathLibpy).Ref("Str")
	}
	return p.pystr_
}

func (p *blockCtx) recorder() *goxRecorder {
	if p.isXgoFile {
		return p.rec
	}
	return nil
}

func (p *blockCtx) findImport(name string) (pi pkgImp, ok bool) {
	pi, ok = p.imports[name]
	if !ok && p.autoimps != nil {
		pi, ok = p.autoimps[name]
	}
	return
}

func (p *pkgCtx) newCodeError(pos, end token.Pos, msg string) error {
	return &gogen.CodeError{Fset: p.nodeInterp, Pos: pos, End: end, Msg: msg}
}

func (p *pkgCtx) newCodeErrorf(pos, end token.Pos, format string, args ...any) error {
	return &gogen.CodeError{Fset: p.nodeInterp, Pos: pos, End: end, Msg: fmt.Sprintf(format, args...)}
}

func (p *pkgCtx) handleErrorf(pos, end token.Pos, format string, args ...any) {
	p.handleErr(p.newCodeErrorf(pos, end, format, args...))
}

func (p *pkgCtx) handleErr(err error) {
	p.errs = append(p.errs, err)
}

func (p *pkgCtx) loadNamed(at *gogen.Package, t *types.Named) {
	o := t.Obj()
	if o.Pkg() == at.Types {
		p.loadType(o.Name())
	}
}

func (p *pkgCtx) complete() error {
	return p.errs.ToError()
}

func (p *pkgCtx) loadType(name string) {
	if sym, ok := p.syms[name]; ok {
		if ld, ok := sym.(*typeLoader); ok {
			ld.load()
		}
	}
}

func (p *pkgCtx) loadSymbol(name string) bool {
	if enableRecover {
		defer func() {
			if e := recover(); e != nil {
				p.handleRecover(e, nil)
			}
		}()
	}
	if f, ok := p.syms[name]; ok {
		if ld, ok := f.(*typeLoader); ok {
			doNewType(ld) // create this type, but don't init
			return true
		}
		delete(p.syms, name)
		f.load()
		return true
	}
	return false
}

func (p *pkgCtx) handleRecover(e any, src ast.Node) {
	err := p.recoverErr(e, src)
	p.handleErr(err)
}

func (p *pkgCtx) recoverErr(e any, src ast.Node) error {
	err, ok := e.(error)
	if !ok {
		if src != nil {
			text := p.LoadExpr(src)
			err = p.newCodeErrorf(src.Pos(), src.End(), "compile `%v`: %v", text, e)
		} else {
			err = fmt.Errorf("%v", e)
		}
	}
	return err
}

// lookupClassFile looks up the class file by name.
func (p *pkgCtx) lookupClassNode(name string) ast.Node {
	for f, cls := range p.classes {
		if name == cls.clsfile {
			return f.Name
		}
	}
	return nil
}

const (
	defaultGoFile  = ""
	skippingGoFile = "_skip"
	testingGoFile  = "_test"
)

// NewPackage creates an XGo package instance.
func NewPackage(pkgPath string, pkg *ast.Package, conf *Config) (p *gogen.Package, err error) {
	relBaseDir := conf.RelativeBase
	fset := conf.Fset
	files := pkg.Files
	interp := &nodeInterp{
		fset: fset, files: files, relBaseDir: relBaseDir,
	}
	ctx := &pkgCtx{
		fset:       fset,
		nodeInterp: interp,
		projs:      make(map[string]*gmxProject),
		classes:    make(map[*ast.File]*gmxClass),
		overpos:    make(map[string]token.Pos),
		syms:       make(map[string]loader),
		generics:   make(map[string]bool),
	}
	confGox := &gogen.Config{
		Types:           conf.Types,
		Fset:            fset,
		Importer:        conf.Importer,
		LoadNamed:       ctx.loadNamed,
		HandleErr:       ctx.handleErr,
		NodeInterpreter: interp,
		NewBuiltin:      ctx.newBuiltinDefault,
		DefaultGoFile:   defaultGoFile,
		NoSkipConstant:  conf.NoSkipConstant,
		PkgPathIox:      osxPkgPath,
		DbgPositioner:   interp,
	}
	var rec *goxRecorder
	if conf.Recorder != nil {
		rec = newRecorder(conf.Recorder)
		confGox.Recorder = rec
		defer func() {
			rec.Complete(p.Types.Scope())
		}()
	}
	if enableRecover {
		defer func() {
			if e := recover(); e != nil {
				ctx.handleRecover(e, nil)
				err = ctx.errs.ToError()
			}
		}()
	}
	p = gogen.NewPackage(pkgPath, pkg.Name, confGox)

	if !noMarkAutogen {
		p.CB().NewConstStart(nil, "_").Val(true).EndInit(1)
	}

	// sort files
	type File struct {
		*ast.File
		path string
	}
	sfiles := make([]*File, 0, len(files))
	for fpath, f := range files {
		sfiles = append(sfiles, &File{f, fpath})
	}
	sort.Slice(sfiles, func(i, j int) bool {
		return sfiles[i].path < sfiles[j].path
	})

	for _, f := range sfiles {
		gmx := f.File
		if gmx.IsClass && !gmx.IsNormalGox {
			if debugLoad {
				log.Println("==> ClassFile", f.path)
			}
			loadClass(ctx, p, f.path, gmx, conf)
		}
	}

	for _, f := range sfiles {
		fileLine := !conf.NoFileLine
		fileScope := types.NewScope(p.Types.Scope(), f.Pos(), f.End(), f.path)
		ctx := &blockCtx{
			pkg: p, pkgCtx: ctx, cb: p.CB(), relBaseDir: relBaseDir, fileScope: fileScope,
			fileLine: fileLine, isClass: f.IsClass, rec: rec, imports: make(map[string]pkgImp),
			isXgoFile: true, typesAlias: ctx.featTypesAlias,
		}
		if rec := ctx.rec; rec != nil {
			rec.Scope(f.File, fileScope)
		}
		preloadGopFile(p, ctx, f.path, f.File, conf)
	}

	proj, multi := gmxCheckProjs(p, ctx)

	gopSyms := make(map[string]bool) // TODO(xsw): remove this map
	for name := range ctx.syms {
		gopSyms[name] = true
	}

	gofiles := make([]*ast.File, 0, len(pkg.GoFiles))
	for _, gof := range pkg.GoFiles {
		f := fromgo.ASTFile(gof, 0)
		gofiles = append(gofiles, f)
		ctx := &blockCtx{
			pkg: p, pkgCtx: ctx, cb: p.CB(), relBaseDir: relBaseDir,
			imports: make(map[string]pkgImp), typesAlias: ctx.featTypesAlias,
		}
		preloadFile(p, ctx, f, skippingGoFile, false)
	}

	initGopPkg(ctx, p, gopSyms)

	// genMain = true if it is main package and no main func
	var genMain bool
	var mainClass string
	if pkg.Name == "main" {
		_, hasMain := ctx.syms["main"]
		genMain = !hasMain
	}

	for _, f := range sfiles {
		if f.IsProj {
			loadFile(ctx, f.File)
		}
	}

	if genMain { // make classfile main func if need
		if proj != nil {
			if !multi { // only one project file
				mainClass = proj.getGameClass(ctx)
			}
		} else if ctx.goxMain == 1 {
			mainClass = ctx.goxMainClass // main func in normal gox file
		}
	}

	for _, f := range sfiles {
		if !f.IsProj {
			loadFile(ctx, f.File)
		}
	}
	if conf.Outline {
		for _, f := range gofiles {
			loadFile(ctx, f)
		}
	}
	for _, ld := range ctx.tylds {
		ld.load()
	}
	for _, load := range ctx.inits {
		load()
	}
	err = ctx.complete()

	if mainClass != "" { // generate classfile main func
		genMainFunc(p, mainClass)
	} else if genMain && !conf.NoAutoGenMain { // generate empty main func
		old, _ := p.SetCurFile(defaultGoFile, false)
		p.NewFunc(nil, "main", nil, nil, false).BodyStart(p).End()
		p.RestoreCurFile(old)
	}
	return
}

func isOverloadFunc(name string) bool {
	n := len(name)
	return n > 3 && name[n-3:n-1] == "__"
}

func initGopPkg(ctx *pkgCtx, pkg *gogen.Package, gopSyms map[string]bool) {
	for name, f := range ctx.syms {
		if gopSyms[name] {
			continue
		}
		if _, ok := f.(*typeLoader); ok {
			ctx.loadType(name)
		} else if isOverloadFunc(name) {
			ctx.loadSymbol(name)
		}
	}
	for _, lbi := range ctx.lbinames {
		if name, ok := lbi.(string); ok {
			ctx.loadSymbol(name)
		} else {
			ctx.loadType(lbi.(*ast.Ident).Name)
		}
	}
	gogen.InitThisGopPkgEx(pkg.Types, ctx.overpos)
}

func loadFile(ctx *pkgCtx, f *ast.File) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					ctx.loadType(spec.(*ast.TypeSpec).Name.Name)
				}
			case token.CONST, token.VAR:
				for _, spec := range d.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						ctx.loadSymbol(name.Name)
					}
				}
			}
		case *ast.FuncDecl:
			if d.Recv == nil {
				name := d.Name.Name
				if name != "init" {
					ctx.loadSymbol(name)
				}
			} else {
				if name, ok := getRecvTypeName(ctx, d.Recv, false); ok {
					getTypeLoader(ctx, ctx.syms, token.NoPos, token.NoPos, name).load()
				}
			}
		}
	}
}

// gen testingGoFile for:
//
//	*_test.xgo
//	*_test.gop
//	*test.gox
func genGoFile(file string, goxTestFile bool) string {
	if goxTestFile || strings.HasSuffix(file, "_test.xgo") || strings.HasSuffix(file, "_test.gop") {
		return testingGoFile
	}
	return defaultGoFile
}

func preloadGopFile(p *gogen.Package, ctx *blockCtx, file string, f *ast.File, conf *Config) {
	var proj *gmxProject
	var c *gmxClass
	var classType, gameClass string
	var testType string
	var baseTypeName string
	var baseType types.Type
	var sp *spxObj
	var goxTestFile bool
	var parent = ctx.pkgCtx
	if f.IsClass {
		if f.IsNormalGox {
			classType, _, _ = ClassNameAndExt(file)
			if f.ShadowEntry != nil {
				parent.goxMainClass = classType
				parent.goxMain++
			}
		} else {
			c = parent.classes[f]
			proj, ctx.proj = c.proj, c.proj
			classType, gameClass = c.getName(parent), proj.getGameClass(parent)
			ctx.autoimps = proj.autoimps
			goxTestFile = proj.isTest
			if goxTestFile { // test classfile
				testType = classType
				if !f.IsProj {
					classType = casePrefix + testNameSuffix(testType)
				}
			}
			if f.IsProj {
				classType = gameClass
				o := proj.game
				ctx.baseClass = o
				baseTypeName, baseType = o.Name(), o.Type()
				if proj.gameIsPtr {
					baseType = types.NewPointer(baseType)
				}
			} else {
				sp = c.sp
				o := sp.obj
				ctx.baseClass = o
				baseTypeName, baseType = o.Name(), o.Type()
			}
		}
	}
	goFile := genGoFile(file, goxTestFile)
	if classType != "" {
		if debugLoad {
			log.Println("==> Preload type", classType)
		}
		if proj != nil {
			ctx.lookups = make([]gogen.PkgRef, len(proj.pkgPaths))
			for i, pkgPath := range proj.pkgPaths {
				ctx.lookups[i] = p.Import(pkgPath)
			}
		}
		syms := parent.syms
		pos := f.Pos()
		end := f.End()
		ctx.classDecl = f.ClassFieldsDecl()
		ld := getTypeLoader(parent, syms, pos, end, classType)
		ld.typ = func() {
			if debugLoad {
				log.Println("==> Load > NewType", classType)
			}
			old, _ := p.SetCurFile(goFile, true)
			defer p.RestoreCurFile(old)

			decl := p.NewTypeDefs().NewType(classType)
			ld.typInit = func() { // decycle
				if debugLoad {
					log.Println("==> Load > InitType", classType)
				}
				old, _ := p.SetCurFile(goFile, true)
				defer p.RestoreCurFile(old)

				pkg := p.Types
				var flds []*types.Var
				var tags []string
				chk := newCheckRedecl()
				if baseTypeName != "" { // base class (not normal classfile)
					flds = append(flds, types.NewField(pos, pkg, baseTypeName, baseType, true))
					tags = append(tags, "")
					chk.chkRedecl(ctx, baseTypeName, pos, end)
					if sp != nil { // for work class
						if !goxTestFile && gameClass != "" { // has project class
							typ := toType(ctx, &ast.Ident{Name: gameClass})
							getUnderlying(ctx, typ) // ensure type is loaded
							typ = types.NewPointer(typ)
							name := getTypeName(typ)
							if !chk.chkRedecl(ctx, name, pos, end) {
								fld := types.NewField(pos, pkg, name, typ, true)
								flds = append(flds, fld)
								tags = append(tags, "")
							}
						}
					} else { // embed work classes for project class
						flds = proj.embed(flds, p)
					}
				}
				rec := ctx.recorder()
				if classDecl := ctx.classDecl; classDecl != nil {
					for _, v := range classDecl.Specs {
						spec := v.(*ast.ValueSpec)
						typ := toType(ctx, spec.Type)
						tag := toFieldTag(spec.Tag)
						if len(spec.Names) == 0 {
							name := parseTypeEmbedName(spec.Type)
							if chk.chkRedecl(ctx, name.Name, spec.Type.Pos(), spec.Type.End()) {
								continue
							}
							fld := types.NewField(spec.Type.Pos(), pkg, name.Name, typ, true)
							if rec != nil {
								rec.Def(name, fld)
							}
							flds = append(flds, fld)
							tags = append(tags, tag)
						} else {
							for _, name := range spec.Names {
								if chk.chkRedecl(ctx, name.Name, name.Pos(), name.End()) {
									continue
								}
								fld := types.NewField(name.Pos(), pkg, name.Name, typ, false)
								if rec != nil {
									rec.Def(name, fld)
								}
								flds = append(flds, fld)
								tags = append(tags, tag)
							}
						}
					}
				}
				decl.InitType(p, types.NewStruct(flds, tags))
			}
			parent.tylds = append(parent.tylds, ld)
		}

		// bugfix: see TestGopxNoFunc
		parent.lbinames = append(parent.lbinames, classType)

		ctx.classRecv = &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{
				{Name: "this"},
			},
			Type: &ast.StarExpr{
				X: &ast.Ident{Name: classType},
			},
		}}}
	}

	if d := f.ShadowEntry; d != nil {
		d.Name.Name = getEntrypoint(f)
	} else if baseTypeName != "" { // isClass && not isNormalGox
		astEmptyEntrypoint(f)
	}

	preloadFile(p, ctx, f, goFile, !conf.Outline)
	if sp != nil && sp.feats != 0 {
		spfeats := sp.feats
		ld := getTypeLoader(parent, parent.syms, token.NoPos, token.NoPos, classType)
		if (spfeats & spriteClassfname) != 0 { // Classfname() string
			ld.methods = append(ld.methods, func() {
				old, _ := p.SetCurFile(goFile, true)
				defer p.RestoreCurFile(old)
				doInitType(ld)
				genClassfname(ctx, c)
			})
		}
		if (spfeats & spriteClassclone) != 0 { // Classclone() clonetype
			ld.methods = append(ld.methods, func() {
				old, _ := p.SetCurFile(goFile, true)
				defer p.RestoreCurFile(old)
				doInitType(ld)
				genClassclone(ctx, sp.clone)
			})
		}
	}
	if goxTestFile {
		parent.inits = append(parent.inits, func() {
			old, _ := p.SetCurFile(testingGoFile, true)
			gmxTestFunc(p, testType, f.IsProj)
			p.RestoreCurFile(old)
		})
	}
}

func parseTypeEmbedName(typ ast.Expr) *ast.Ident {
retry:
	switch t := typ.(type) {
	case *ast.Ident:
		return t
	case *ast.SelectorExpr:
		return t.Sel
	case *ast.StarExpr:
		typ = t.X
		goto retry
	}
	panic("TODO: parseTypeEmbedName unexpected")
}

func preloadFile(p *gogen.Package, ctx *blockCtx, f *ast.File, goFile string, genFnBody bool) {
	parent := ctx.pkgCtx
	classDecl := ctx.classDecl
	syms := parent.syms
	old, _ := p.SetCurFile(goFile, true)
	defer p.RestoreCurFile(old)

	preloadFuncDecl := func(d *ast.FuncDecl) {
		if ctx.classRecv != nil { // in class file (.spx/.gmx)
			if recv := d.Recv; recv == nil || len(recv.List) == 0 {
				d.Recv = ctx.classRecv
				d.IsClass = true
			}
		}
		name := d.Name
		fname := name.Name
		if d.Recv == nil {
			fn := func() {
				old, _ := p.SetCurFile(goFile, true)
				defer p.RestoreCurFile(old)
				loadFunc(ctx, nil, fname, d, genFnBody)
			}
			if fname == "init" {
				if genFnBody {
					if debugLoad {
						log.Println("==> Preload func init")
					}
					parent.inits = append(parent.inits, fn)
				}
			} else {
				if debugLoad {
					log.Println("==> Preload func", fname)
				}
				if initLoader(parent, syms, name.Pos(), name.End(), fname, fn, genFnBody) {
					if strings.HasPrefix(fname, "Gopx_") { // Gopx_xxx func
						ctx.lbinames = append(ctx.lbinames, fname)
					}
				}
			}
		} else if tname, ok := getRecvTypeName(parent, d.Recv, true); ok {
			if d.Static {
				if debugLoad {
					log.Printf("==> Preload static method %s.%s\n", tname, fname)
				}
				fname = staticMethod(tname, fname)
				fn := func() {
					old, _ := p.SetCurFile(goFile, true)
					defer p.RestoreCurFile(old)
					loadFunc(ctx, nil, fname, d, genFnBody)
				}
				initLoader(parent, syms, name.Pos(), name.End(), fname, fn, genFnBody)
				ctx.lbinames = append(ctx.lbinames, fname)
			} else {
				if debugLoad {
					log.Printf("==> Preload method %s.%s\n", tname, fname)
				}
				ld := getTypeLoader(parent, syms, token.NoPos, token.NoPos, tname)
				fn := func() {
					old, _ := p.SetCurFile(goFile, true)
					defer p.RestoreCurFile(old)
					doInitType(ld)
					recv := toRecv(ctx, d.Recv)
					loadFunc(ctx, recv, fname, d, genFnBody)
				}
				ld.methods = append(ld.methods, fn)
			}
		}
	}

	preloadConst := func(d *ast.GenDecl) {
		pkg := ctx.pkg
		cdecl := pkg.NewConstDefs(pkg.Types.Scope())
		for _, spec := range d.Specs {
			vSpec := spec.(*ast.ValueSpec)
			if debugLoad {
				log.Println("==> Preload const", vSpec.Names)
			}
			setNamesLoader(parent, syms, vSpec.Names, func() {
				if c := cdecl; c != nil {
					cdecl = nil
					loadConstSpecs(ctx, c, d.Specs)
					for _, s := range d.Specs {
						v := s.(*ast.ValueSpec)
						removeNames(syms, v.Names)
					}
				}
			})
		}
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.IMPORT:
				for _, item := range d.Specs {
					loadImport(ctx, item.(*ast.ImportSpec))
				}
			case token.TYPE:
				for _, spec := range d.Specs {
					t := spec.(*ast.TypeSpec)
					tName := t.Name
					name := tName.Name
					if debugLoad {
						log.Println("==> Preload type", name)
					}
					pos := tName.Pos()
					end := tName.End()
					ld := getTypeLoader(parent, syms, pos, end, name)
					defs := ctx.pkg.NewTypeDefs()
					if goFile != skippingGoFile { // is XGo file
						ld.typ = func() {
							old, _ := p.SetCurFile(goFile, true)
							defer p.RestoreCurFile(old)
							if t.Assign != token.NoPos { // alias type
								if debugLoad {
									log.Println("==> Load > AliasType", name)
								}
								typ := defs.AliasType(name, toType(ctx, t.Type), tName)
								if rec := ctx.recorder(); rec != nil {
									if obj, ok := typ.(interface{ Obj() *types.TypeName }); ok {
										rec.Def(tName, obj.Obj())
									}
								}
								return
							}
							if debugLoad {
								log.Println("==> Load > NewType", name)
							}
							decl := defs.NewType(name, tName)
							if t.Doc != nil {
								defs.SetComments(t.Doc)
							} else if d.Doc != nil {
								defs.SetComments(d.Doc)
							}
							ld.typInit = func() { // decycle
								if debugLoad {
									log.Println("==> Load > InitType", name)
								}
								decl.InitType(ctx.pkg, toType(ctx, t.Type))
								if rec := ctx.recorder(); rec != nil {
									rec.Def(tName, decl.Type().Obj())
								}
							}
						}
					} else {
						ctx.generics[name] = true
						ld.typ = func() {
							pkg := ctx.pkg.Types
							if t.Assign != token.NoPos { // alias type
								if debugLoad {
									log.Println("==> Load > AliasType", name)
								}
								aliasType(ctx, pkg, t.Pos(), name, t)
								return
							}
							if debugLoad {
								log.Println("==> Load > NewType", name)
							}
							named := newType(pkg, t.Pos(), name)

							ld.typInit = func() { // decycle
								if debugLoad {
									log.Println("==> Load > InitType", name)
								}
								initType(ctx, named, t)
							}
						}
					}
				}
			case token.CONST:
				preloadConst(d)
			case token.VAR:
				if d == classDecl { // skip class fields
					continue
				}
				for _, spec := range d.Specs {
					vSpec := spec.(*ast.ValueSpec)
					if debugLoad {
						log.Println("==> Preload var", vSpec.Names)
					}
					setNamesLoader(parent, syms, vSpec.Names, func() {
						if v := vSpec; v != nil { // only init once
							vSpec = nil
							old, _ := p.SetCurFile(goFile, true)
							defer p.RestoreCurFile(old)
							loadVars(ctx, v, d.Doc, true)
							removeNames(syms, v.Names)
						}
					})
				}
			default:
				log.Panicln("TODO - tok:", d.Tok, "spec:", reflect.TypeOf(d.Specs).Elem())
			}

		case *ast.FuncDecl:
			preloadFuncDecl(d)

		case *ast.OverloadFuncDecl:
			var recv *ast.Ident
			if ctx.classRecv != nil { // in class file (.spx/.gmx)
				if recv := d.Recv; recv == nil || len(recv.List) == 0 {
					d.Recv = &ast.FieldList{
						List: []*ast.Field{
							{
								Type: ctx.classRecv.List[0].Type.(*ast.StarExpr).X,
							},
						},
					}
					d.IsClass = true
				}
			}
			if d.Recv != nil {
				var ok bool
				recv, ok = d.Recv.List[0].Type.(*ast.Ident)
				if !ok {
					ctx.handleErrorf(d.Recv.List[0].Type.Pos(), d.Recv.List[0].Type.End(), "invalid recv type %v", ctx.LoadExpr(d.Recv.List[0].Type))
					break
				}
				ctx.lbinames = append(ctx.lbinames, recv)
				if ctx.rec != nil {
					ctx.rec.ReferUse(recv, recv.Name)
				}
			}
			onames := make([]string, 0, 4)
			exov := false
			name := d.Name
		LoopFunc:

			for idx, fn := range d.Funcs {
				switch expr := fn.(type) {
				case *ast.Ident:
					if d.Recv != nil && !d.Operator && !d.IsClass {
						ctx.handleErrorf(expr.Pos(), expr.End(), "invalid method %v", ctx.LoadExpr(expr))
						break LoopFunc
					}
					exov = true
					if d.IsClass {
						onames = append(onames, "."+expr.Name)
					} else {
						onames = append(onames, expr.Name)
						ctx.lbinames = append(ctx.lbinames, expr.Name)
					}
					if ctx.rec != nil {
						if d.IsClass {
							ctx.rec.ReferUse(expr, recv.Name+"."+expr.Name)
						} else {
							ctx.rec.ReferUse(expr, expr.Name)
						}
					}
				case *ast.SelectorExpr:
					if d.Recv == nil || d.IsClass {
						ctx.handleErrorf(expr.Pos(), expr.End(), "invalid func %v", ctx.LoadExpr(expr))
						break LoopFunc
					}
					rtyp, ok := checkOverloadMethodRecvType(recv, expr.X)
					if !ok {
						ctx.handleErrorf(expr.Pos(), expr.End(), "invalid recv type %v", ctx.LoadExpr(expr.X))
						break LoopFunc
					}

					onames = append(onames, "."+expr.Sel.Name)
					exov = true
					if ctx.rec != nil {
						ctx.rec.ReferUse(rtyp, rtyp.Name)
						ctx.rec.ReferUse(expr.Sel, rtyp.Name+"."+expr.Sel.Name)
					}
				case *ast.FuncLit:
					if d.Recv != nil && !d.Operator && !d.IsClass {
						ctx.handleErrorf(expr.Pos(), expr.End(), "invalid method %v", ctx.LoadExpr(expr))
						break LoopFunc
					}
					name1 := overloadFuncName(name.Name, idx)
					onames = append(onames, "") // const Gopo_xxx = "xxxInt,,xxxFloat"
					ctx.lbinames = append(ctx.lbinames, name1)
					id := &ast.Ident{NamePos: expr.Pos(), Name: name1}
					if ctx.rec != nil {
						ctx.rec.ReferDef(id, expr)
					}
					preloadFuncDecl(&ast.FuncDecl{
						Doc:  d.Doc,
						Name: id,
						Type: expr.Type,
						Body: expr.Body,
					})
				default:
					ctx.handleErrorf(expr.Pos(), expr.End(), "unknown func %v", ctx.LoadExpr(expr))
					break LoopFunc
				}
			}
			if exov { // need Gopo_xxx
				oname, err := overloadName(recv, name.Name, d.Operator)
				if err != nil {
					ctx.handleErrorf(name.Pos(), name.End(), "%v", err)
					break
				}
				if recv != nil {
					ctx.overpos[recv.Name+"."+name.Name] = name.NamePos
				} else {
					ctx.overpos[name.Name] = name.NamePos
				}
				oval := strings.Join(onames, ",")
				preloadConst(&ast.GenDecl{
					Doc: d.Doc,
					Tok: token.CONST,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names:  []*ast.Ident{{Name: oname}},
							Values: []ast.Expr{stringLit(oval)},
						},
					},
				})
				ctx.lbinames = append(ctx.lbinames, oname)
			} else {
				ctx.overpos[name.Name] = name.NamePos
			}
			if ctx.rec != nil {
				ctx.rec.ReferDef(d.Name, d)
			}

		default:
			log.Panicf("TODO - cl.preloadFile: unknown decl - %T\n", decl)
		}
	}
}

func checkOverloadMethodRecvType(ot *ast.Ident, recv ast.Expr) (*ast.Ident, bool) {
	rtyp, _, ok := getRecvType(recv)
	if !ok {
		return nil, false
	}
	rt, ok := rtyp.(*ast.Ident)
	if !ok || ot.Name != rt.Name {
		return nil, false
	}
	return rt, true
}

const (
	indexTable = "0123456789abcdefghijklmnopqrstuvwxyz"
)

func overloadFuncName(name string, idx int) string {
	return name + "__" + indexTable[idx:idx+1]
}

func overloadName(recv *ast.Ident, name string, isOp bool) (string, error) {
	if isOp {
		if oname, ok := binaryGopNames[name]; ok {
			name = oname
		} else {
			return "", fmt.Errorf("invalid overload operator %v", name)
		}
	}
	sep := "_"
	if strings.ContainsRune(name, '_') || (recv != nil && strings.ContainsRune(recv.Name, '_')) {
		sep = "__"
	}
	typ := ""
	if recv != nil {
		typ = recv.Name + sep
	}
	return "Gopo" + sep + typ + name, nil
}

func staticMethod(tname, name string) string {
	sep := "_"
	if strings.ContainsRune(name, '_') || strings.ContainsRune(tname, '_') {
		sep = "__"
	}
	return "Gops" + sep + tname + sep + name
}

func stringLit(val string) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(val)}
}

func newType(pkg *types.Package, pos token.Pos, name string) *types.Named {
	typName := types.NewTypeName(pos, pkg, name, nil)
	if old := pkg.Scope().Insert(typName); old != nil {
		log.Panicf("%s redeclared in this block\n\tprevious declaration at %v\n", name, "<TODO>")
	}
	return types.NewNamed(typName, nil, nil)
}

func aliasType(ctx *blockCtx, pkg *types.Package, pos token.Pos, name string, t *ast.TypeSpec) {
	if ctx.typesAlias {
		var typeParams []*types.TypeParam
		if t.TypeParams != nil {
			typeParams = toTypeParams(ctx, t.TypeParams)
			ctx.tlookup = &typeParamLookup{typeParams}
			defer func() {
				ctx.tlookup = nil
			}()
			org := ctx.inInst
			ctx.inInst = 0
			defer func() {
				ctx.inInst = org
			}()
		}
		o := typesalias.NewAlias(types.NewTypeName(pos, pkg, name, nil), toType(ctx, t.Type))
		if typeParams != nil {
			typesalias.SetTypeParams(o, typeParams)
		}
		pkg.Scope().Insert(o.Obj())
	} else {
		o := types.NewTypeName(pos, pkg, name, toType(ctx, t.Type))
		pkg.Scope().Insert(o)
	}
}

func loadFunc(ctx *blockCtx, recv *types.Var, name string, d *ast.FuncDecl, genBody bool) {
	if debugLoad {
		if recv == nil {
			log.Println("==> Load func", name)
		} else {
			log.Printf("==> Load method %v.%s\n", recv.Type(), name)
		}
	}
	var pkg = ctx.pkg
	var sigBase *types.Signature
	if d.Shadow {
		if recv != nil && (name == "Main" || name == "MainEntry") {
			if base := ctx.baseClass; base != nil {
				if f := findMethod(base, name); f != nil {
					sigBase = makeMainSig(recv, f)
				}
			}
		}
	} else if d.Operator {
		if recv != nil { // binary op
			if v, ok := binaryGopNames[name]; ok {
				name = v
			}
		} else { // unary op
			if v, ok := unaryGopNames[name]; ok {
				name = v
				at := pkg.Types
				arg1 := d.Type.Params.List[0]
				typ := toType(ctx, arg1.Type)
				recv = types.NewParam(arg1.Pos(), at, arg1.Names[0].Name, typ)
				d.Type.Params.List = nil
			}
		}
	}
	sig := sigBase
	if sig == nil {
		sig = toFuncType(ctx, d.Type, recv, d)
	}
	fn, err := pkg.NewFuncWith(d.Name.Pos(), name, sig, func() token.Pos {
		return d.Recv.List[0].Type.Pos()
	})
	if err != nil {
		ctx.handleErr(err)
		return
	}
	commentFunc(ctx, fn, d)
	if rec := ctx.recorder(); rec != nil {
		rec.Def(d.Name, fn.Func)
		if recv == nil && name != "_" {
			ctx.fileScope.Insert(fn.Func)
		}
	}
	if genBody {
		if body := d.Body; body != nil {
			if recv != nil {
				file := pkg.CurFile()
				ctx.inits = append(ctx.inits, func() { // interface issue: #795
					old := pkg.RestoreCurFile(file)
					loadFuncBody(ctx, fn, body, sigBase, d)
					pkg.RestoreCurFile(old)
				})
			} else {
				loadFuncBody(ctx, fn, body, nil, d)
			}
		}
	}
}

var binaryGopNames = map[string]string{
	"+": "Gop_Add",
	"-": "Gop_Sub",
	"*": "Gop_Mul",
	"/": "Gop_Quo",
	"%": "Gop_Rem",

	"&":  "Gop_And",
	"|":  "Gop_Or",
	"^":  "Gop_Xor",
	"<<": "Gop_Lsh",
	">>": "Gop_Rsh",
	"&^": "Gop_AndNot",

	"+=": "Gop_AddAssign",
	"-=": "Gop_SubAssign",
	"*=": "Gop_MulAssign",
	"/=": "Gop_QuoAssign",
	"%=": "Gop_RemAssign",

	"&=":  "Gop_AndAssign",
	"|=":  "Gop_OrAssign",
	"^=":  "Gop_XorAssign",
	"<<=": "Gop_LshAssign",
	">>=": "Gop_RshAssign",
	"&^=": "Gop_AndNotAssign",

	"==": "Gop_EQ",
	"!=": "Gop_NE",
	"<=": "Gop_LE",
	"<":  "Gop_LT",
	">=": "Gop_GE",
	">":  "Gop_GT",

	"->": "Gop_PointTo",
	"<>": "Gop_PointBi",

	"&&": "Gop_LAnd",
	"||": "Gop_LOr",

	"<-": "Gop_Send",
}

var unaryGopNames = map[string]string{
	"++": "Gop_Inc",
	"--": "Gop_Dec",
	"-":  "Gop_Neg",
	"+":  "Gop_Dup",
	"^":  "Gop_Not",
	"!":  "Gop_LNot",
	"<-": "Gop_Recv",
}

func loadFuncBody(ctx *blockCtx, fn *gogen.Func, body *ast.BlockStmt, sigBase *types.Signature, src ast.Node) {
	cb := fn.BodyStart(ctx.pkg, body)
	cb.SetComments(nil, false)
	if sigBase != nil {
		// this.Sprite.Main(...) or this.Game.MainEntry(...)
		cb.VarVal("this").MemberVal(ctx.baseClass.Name()).MemberVal(fn.Name())
		params := sigBase.Params()
		n := params.Len()
		for i := 0; i < n; i++ {
			cb.Val(params.At(i))
		}
		cb.Call(n).EndStmt()
	}
	compileStmts(ctx, body.List)
	if rec := ctx.recorder(); rec != nil {
		switch fn := src.(type) {
		case *ast.FuncDecl:
			rec.Scope(fn.Type, cb.Scope())
		case *ast.FuncLit:
			rec.Scope(fn.Type, cb.Scope())
		}
	}
	cb.End(src)
}

func loadImport(ctx *blockCtx, spec *ast.ImportSpec) {
	if enableRecover {
		defer func() {
			if e := recover(); e != nil {
				ctx.handleRecover(e, spec)
			}
		}()
	}
	pkgPath := simplifyPkgPath(toString(spec.Path))
	pkg := ctx.pkg.Import(pkgPath, spec)

	var pos token.Pos
	var name string
	var specName = spec.Name
	if specName != nil {
		name, pos = specName.Name, specName.Pos()
	} else {
		name, pos = pkg.Types.Name(), spec.Path.Pos()
	}
	pkgName := types.NewPkgName(pos, ctx.pkg.Types, name, pkg.Types)
	if rec := ctx.recorder(); rec != nil {
		if specName != nil {
			rec.Def(specName, pkgName)
		} else {
			rec.Implicit(spec, pkgName)
		}
		if name != "_" {
			ctx.fileScope.Insert(pkgName)
		}
	}

	if specName != nil {
		if name == "." {
			ctx.lookups = append(ctx.lookups, pkg)
			return
		}
		if name == "_" {
			pkg.MarkForceUsed(ctx.pkg)
			return
		}
	}
	ctx.imports[name] = pkgImp{pkg, pkgName}
}

func loadConstSpecs(ctx *blockCtx, cdecl *gogen.ConstDefs, specs []ast.Spec) {
	for iotav, spec := range specs {
		vSpec := spec.(*ast.ValueSpec)
		loadConsts(ctx, cdecl, vSpec, iotav)
	}
}

func loadConsts(ctx *blockCtx, cdecl *gogen.ConstDefs, v *ast.ValueSpec, iotav int) {
	vNames := v.Names
	names := makeNames(vNames)
	if v.Values == nil {
		if debugLoad {
			log.Println("==> Load const", names)
		}
		cdecl.Next(iotav, v.Pos(), names...)
		defNames(ctx, vNames, nil)
		return
	}
	var typ types.Type
	if v.Type != nil {
		typ = toType(ctx, v.Type)
	}
	if debugLoad {
		log.Println("==> Load const", names, typ)
	}
	fn := func(cb *gogen.CodeBuilder) int {
		for _, val := range v.Values {
			compileExpr(ctx, val)
		}
		return len(v.Values)
	}
	cdecl.New(fn, iotav, v.Pos(), typ, names...)
	defNames(ctx, v.Names, nil)
}

func loadVars(ctx *blockCtx, v *ast.ValueSpec, doc *ast.CommentGroup, global bool) {
	var typ types.Type
	if v.Type != nil {
		typ = toType(ctx, v.Type)
	}
	names := makeNames(v.Names)
	if debugLoad {
		log.Println("==> Load var", typ, names)
	}
	var scope *types.Scope
	if global {
		scope = ctx.pkg.Types.Scope()
	} else {
		scope = ctx.cb.Scope()
	}
	varDefs := ctx.pkg.NewVarDefs(scope).SetComments(doc)
	varDecl := varDefs.New(v.Names[0].Pos(), typ, names...)
	if nv := len(v.Values); nv > 0 {
		cb := varDecl.InitStart(ctx.pkg)
		if enableRecover {
			defer func() {
				if e := recover(); e != nil {
					cb.ResetInit()
					panic(e)
				}
			}()
		}
		if nv == 1 && len(names) == 2 {
			compileExpr(ctx, v.Values[0], clCallWithTwoValue)
		} else {
			for _, val := range v.Values {
				switch e := val.(type) {
				case *ast.LambdaExpr, *ast.LambdaExpr2:
					if len(v.Values) == 1 {
						sig, err := checkLambdaFuncType(ctx, e, typ, clLambaAssign, v.Names[0])
						if err != nil {
							panic(err)
						}
						compileLambda(ctx, e, sig)
					} else {
						panic(ctx.newCodeErrorf(e.Pos(), e.End(), "lambda unsupport multiple assignment"))
					}
				case *ast.SliceLit:
					compileSliceLit(ctx, e, typ)
				case *ast.CompositeLit:
					compileCompositeLit(ctx, e, typ, false)
				default:
					compileExpr(ctx, val)
				}
			}
		}
		cb.EndInit(nv)
	}
	defNames(ctx, v.Names, scope)
}

func defNames(ctx *blockCtx, names []*ast.Ident, scope *types.Scope) {
	if rec := ctx.recorder(); rec != nil {
		if scope == nil {
			scope = ctx.cb.Scope()
		}
		for _, name := range names {
			if o := scope.Lookup(name.Name); o != nil {
				rec.Def(name, o)
			}
		}
	}
}

func makeNames(vals []*ast.Ident) []string {
	names := make([]string, len(vals))
	for i, v := range vals {
		names[i] = v.Name
	}
	return names
}

func removeNames(syms map[string]loader, names []*ast.Ident) {
	for _, name := range names {
		delete(syms, name.Name)
	}
}

func setNamesLoader(ctx *pkgCtx, syms map[string]loader, names []*ast.Ident, load func()) {
	for _, name := range names {
		initLoader(ctx, syms, name.Pos(), name.End(), name.Name, load, true)
	}
}

// -----------------------------------------------------------------------------
