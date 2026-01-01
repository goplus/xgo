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

package cl

import (
	goast "go/ast"
	"go/constant"
	gotoken "go/token"
	"go/types"
	"log"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/mod/modfile"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
	"github.com/qiniu/x/stringutil"
)

// -----------------------------------------------------------------------------

type gmxClass struct {
	name    string // class type, empty for default project class
	clsfile string
	ext     string
	proj    *gmxProject
	sp      *spxObj
}

func (p *gmxClass) getName(ctx *pkgCtx) string {
	tname := p.name
	if tname == "" { // use default project class
		tname = p.proj.getGameClass(ctx)
	}
	return tname
}

type spxObj struct {
	obj    gogen.Ref // work base class
	ext    string
	proto  string           // work class prototype
	prefix string           // work class prefix
	feats  spriteFeat       // work class features
	clone  *types.Signature // prototype of Classclone
	types  []string         // work classes (ie. <type>.spx)
}

func spriteByProto(sprites []*spxObj, proto string) *spxObj {
	for _, sp := range sprites {
		if sp.proto == proto {
			return sp
		}
	}
	return nil
}

type gmxProject struct {
	gameClass_ string    // <gmtype>.gmx
	game       gogen.Ref // Game (project base class)
	sprites    []*spxObj // .spx => Sprite
	scheds     []string
	schedStmts []goast.Stmt // nil or len(scheds) == 2 (delayload)
	pkgImps    []gogen.PkgRef
	pkgPaths   []string
	autoimps   map[string]pkgImp // auto-import statement in gox.mod
	gt         *Project
	hasScheds  bool
	gameIsPtr  bool
	isTest     bool
	hasMain_   bool
}

func (p *gmxProject) embed(chk func(name string) bool, flds []*types.Var, pkg *gogen.Package) []*types.Var {
	for _, sp := range p.sprites {
		if sp.feats&spriteEmbedded != 0 {
			for _, spt := range sp.types {
				spto := pkg.Ref(spt) // work class
				if chk != nil && !chk(spto.Name()) {
					pt := types.NewPointer(spto.Type()) // pointer to work class
					flds = append(flds, types.NewField(token.NoPos, pkg.Types, spto.Name(), pt, false))
				}
			}
		}
	}
	return flds
}

type spriteFeat uint

const (
	spriteClassfname spriteFeat = 1 << iota
	spriteClassclone

	spriteEmbedded spriteFeat = 0x80
)

func spriteFeature(elt types.Type, sp *spxObj) {
	if intf, ok := elt.(*types.Interface); ok {
		for i, n := 0, intf.NumMethods(); i < n; i++ {
			switch m := intf.Method(i); m.Name() {
			case "Classfname":
				sp.feats |= spriteClassfname
			case "Classclone":
				sp.clone = m.Type().(*types.Signature)
				sp.feats |= spriteClassclone
			}
		}
	}
}

func spriteFeatures(game gogen.Ref, sprites []*spxObj) {
	if mainFn := findMethod(game, "Main"); mainFn != nil {
		sig := mainFn.Type().(*types.Signature)
		if t, ok := gogen.CheckSigFuncEx(sig); ok {
			if t, ok := t.(*gogen.TyTemplateRecvMethod); ok {
				sig = t.Func.Type().(*types.Signature)
			}
		}
		if n := len(sprites); n == 1 && sig.Variadic() {
			// single work class
			in := sig.Params()
			last := in.At(in.Len() - 1)
			elt := last.Type().(*types.Slice).Elem()
			if tn, ok := elt.(*types.Named); ok {
				elt = tn.Underlying()
			}
			spriteFeature(elt, sprites[0])
		} else {
			// multiple work classes
			in := sig.Params()
			for i, narg := 1, in.Len(); i < narg; i++ { // TODO(xsw): error handling
				tslice := in.At(i).Type().(*types.Slice)
				tn := tslice.Elem().(*types.Named)
				sp := spriteByProto(sprites, tn.Obj().Name())
				spriteFeature(tn.Underlying(), sp)
			}
		}
	}
}

func (p *gmxProject) getGameClass(ctx *pkgCtx) string {
	tname := p.gameClass_ // project class
	if tname != "" && tname != "main" {
		return tname
	}
	gt := p.gt
	tname = gt.Class // project base class
	if p.gameIsPtr {
		tname = tname[1:]
	}
	if ctx.nproj > 1 && !p.hasMain_ {
		tname = stringutil.Capitalize(path.Base(gt.PkgPaths[0])) + tname
	}
	return tname
}

func (p *gmxProject) hasMain() bool {
	if !p.hasMain_ {
		imps := p.pkgImps
		p.hasMain_ = len(imps) > 0 && imps[0].TryRef("GopTestClass") != nil
	}
	return p.hasMain_
}

func (p *gmxProject) getScheds(cb *gogen.CodeBuilder) []goast.Stmt {
	if p == nil || !p.hasScheds {
		return nil
	}
	if p.schedStmts == nil {
		p.schedStmts = make([]goast.Stmt, 2)
		for i, v := range p.scheds {
			fn := cb.Val(spxLookup(p.pkgImps, v)).Call(0).InternalStack().Pop().Val
			p.schedStmts[i] = &goast.ExprStmt{X: fn}
		}
		if len(p.scheds) < 2 {
			p.schedStmts[1] = p.schedStmts[0]
		}
	}
	return p.schedStmts
}

var (
	repl = strings.NewReplacer(":", "", "#", "", "-", "_", ".", "_")
)

func ClassNameAndExt(file string) (name, clsfile, ext string) {
	fname := filepath.Base(file)
	clsfile, ext = modfile.SplitFname(fname)
	name = clsfile
	if strings.ContainsAny(name, ":#-.") {
		name = repl.Replace(name)
	}
	return
}

// GetFileClassType get ast.File classType
// TODO(xsw): to refactor
//
// Deprecated: Don't use it
func GetFileClassType(file *ast.File, filename string, lookupClass func(ext string) (c *Project, ok bool)) (classType string, isTest bool) {
	if file.IsClass {
		var ext string
		classType, _, ext = ClassNameAndExt(filename)
		if file.IsNormalGox {
			isTest = strings.HasSuffix(ext, "_test.gox")
			if !isTest && classType == "main" {
				classType = "_main"
			}
		} else {
			isTest = strings.HasSuffix(ext, "test.gox")
		}
		if file.IsProj && classType == "main" {
			if gt, ok := lookupClass(ext); ok {
				classType = gt.Class
			}
		} else if isTest {
			classType = casePrefix + testNameSuffix(classType)
		}
	} else if strings.HasSuffix(filename, "_test.xgo") || strings.HasSuffix(filename, "_test.gop") {
		isTest = true
	}
	return
}

func isGoxTestFile(ext string) bool {
	return strings.HasSuffix(ext, "test.gox")
}

func loadClass(ctx *pkgCtx, pkg *gogen.Package, file string, f *ast.File, conf *Config) *gmxProject {
	tname, clsfile, ext := ClassNameAndExt(file)
	gt, ok := conf.LookupClass(ext)
	if !ok {
		panic("class not found: " + ext)
	}
	p, ok := ctx.projs[gt.Ext]
	if !ok {
		pkgPaths := gt.PkgPaths
		p = &gmxProject{pkgPaths: pkgPaths, isTest: isGoxTestFile(ext), gt: gt}
		ctx.projs[gt.Ext] = p

		p.pkgImps = make([]gogen.PkgRef, len(pkgPaths))
		for i, pkgPath := range pkgPaths {
			p.pkgImps[i] = pkg.Import(pkgPath)
		}

		if len(gt.Import) > 0 {
			autoimps := make(map[string]pkgImp)
			for _, imp := range gt.Import {
				pkgi := pkg.Import(imp.Path)
				name := imp.Name
				if name == "" {
					name = pkgi.Types.Name()
				}
				pkgName := types.NewPkgName(token.NoPos, pkg.Types, name, pkgi.Types)
				autoimps[name] = pkgImp{pkgi, pkgName}
			}
			p.autoimps = autoimps
		}

		spx := p.pkgImps[0]
		nWork := len(gt.Works)
		sprites := make([]*spxObj, nWork)
		for i, v := range gt.Works {
			if nWork > 1 && v.Proto == "" {
				panic("should have prototype if there are multiple work classes")
			}
			obj, _ := spxRef(spx, v.Class)
			sp := &spxObj{obj: obj, ext: v.Ext, proto: v.Proto, prefix: v.Prefix}
			if v.Embedded {
				sp.feats |= spriteEmbedded
			}
			sprites[i] = sp
		}
		p.sprites = sprites
		if gt.Class != "" {
			p.game, p.gameIsPtr = spxRef(spx, gt.Class)
			spriteFeatures(p.game, sprites)
		}
		if x := getStringConst(spx, "Gop_sched"); x != "" {
			p.scheds, p.hasScheds = strings.SplitN(x, ",", 2), true
		}
	}
	cls := &gmxClass{clsfile: clsfile, ext: ext, proj: p}
	if f.IsProj {
		if p.gameClass_ != "" {
			panic("multiple project files found: " + tname + ", " + p.gameClass_)
		}
		p.gameClass_ = tname
		p.hasMain_ = f.HasShadowEntry()
		if !p.isTest {
			ctx.nproj++
		}
		if tname != "main" {
			cls.name = tname
		}
	} else {
		sp := getSpxObj(p, ext)
		tname := spName(sp, tname)
		sp.types = append(sp.types, tname)
		cls.sp = sp
		cls.name = tname
	}
	ctx.classes[f] = cls
	if debugLoad {
		log.Println("==> InitClass", tname, "isProj:", f.IsProj)
	}
	return p
}

type none = struct{}

var specialNames = map[string]none{
	"init": {}, "main": {}, "go": {}, "goto": {}, "type": {}, "var": {}, "import": {},
	"package": {}, "interface": {}, "struct": {}, "const": {}, "func": {}, "map": {},
	"for": {}, "if": {}, "else": {}, "switch": {}, "case": {}, "select": {}, "defer": {},
	"range": {}, "return": {}, "break": {}, "continue": {}, "fallthrough": {}, "default": {},
}

func spName(sp *spxObj, name string) string {
	if sp.prefix != "" {
		return sp.prefix + name
	}
	if _, ok := specialNames[name]; ok {
		name = "_" + name
	}
	return name
}

func getSpxObj(p *gmxProject, ext string) *spxObj {
	for _, sp := range p.sprites {
		if sp.ext == ext {
			return sp
		}
	}
	return nil
}

func spxLookup(pkgImps []gogen.PkgRef, name string) gogen.Ref {
	for _, pkg := range pkgImps {
		if o := pkg.TryRef(name); o != nil {
			return o
		}
	}
	panic("spxLookup: symbol not found - " + name)
}

func spxTryRef(spx gogen.PkgRef, typ string) (obj types.Object, isPtr bool) {
	if strings.HasPrefix(typ, "*") {
		typ, isPtr = typ[1:], true
	}
	obj = spx.TryRef(typ)
	return
}

func spxRef(spx gogen.PkgRef, typ string) (obj gogen.Ref, isPtr bool) {
	obj, isPtr = spxTryRef(spx, typ)
	if obj == nil {
		panic(spx.Types.Name() + "." + typ + " not found")
	}
	return
}

func getStringConst(spx gogen.PkgRef, name string) string {
	if o := spx.TryRef(name); o != nil {
		if c, ok := o.(*types.Const); ok {
			return constant.StringVal(c.Val())
		}
	}
	return ""
}

func setBodyHandler(ctx *blockCtx) {
	if proj := ctx.proj; proj != nil { // in an XGo class file
		if scheds := proj.getScheds(ctx.cb); scheds != nil {
			ctx.cb.SetBodyHandler(func(body *goast.BlockStmt, kind int) {
				idx := 0
				if len(body.List) == 0 {
					idx = 1
				}
				gogen.InsertStmtFront(body, scheds[idx])
			})
		}
	}
}

const (
	casePrefix = "case"
)

func testNameSuffix(testType string) string {
	if c := testType[0]; c >= 'A' && c <= 'Z' {
		return testType
	}
	return "_" + testType
}

func gmxTestFunc(pkg *gogen.Package, testType string, isProj bool) {
	if isProj {
		genTestFunc(pkg, "TestMain", testType, "m", "M")
	} else {
		name := testNameSuffix(testType)
		genTestFunc(pkg, "Test"+name, casePrefix+name, "t", "T")
	}
}

func genTestFunc(pkg *gogen.Package, name, testType, param, paramType string) {
	testing := pkg.Import("testing")
	objT := testing.Ref(paramType)
	paramT := types.NewParam(token.NoPos, pkg.Types, param, types.NewPointer(objT.Type()))
	params := types.NewTuple(paramT)

	pkg.NewFunc(nil, name, params, nil, false).BodyStart(pkg).
		Val(pkg.Builtin().Ref("new")).Val(pkg.Ref(testType)).Call(1).
		MemberVal("TestMain").Val(paramT).Call(1).EndStmt().
		End()
}

func gmxCheckProjs(pkg *gogen.Package, ctx *pkgCtx) (*gmxProject, bool) {
	var projMain, projNoMain *gmxProject
	var multiMain, multiNoMain bool
	for _, v := range ctx.projs {
		if v.isTest {
			continue
		}
		if v.hasMain() {
			if projMain != nil {
				multiMain = true
			} else {
				projMain = v
			}
		} else {
			if projNoMain != nil {
				multiNoMain = true
			} else {
				projNoMain = v
			}
		}
		if v.game != nil {
			gmxProjMain(pkg, ctx, v)
		}
	}
	if projMain != nil {
		return projMain, multiMain
	}
	return projNoMain, multiNoMain
}

func gmxProjMain(pkg *gogen.Package, parent *pkgCtx, proj *gmxProject) {
	base := proj.game                      // project base class
	classType := proj.getGameClass(parent) // project class
	ld := getTypeLoader(parent, parent.syms, token.NoPos, token.NoPos, classType)
	if ld.typ == nil { // no project class, use default
		ld.typ = func() {
			if debugLoad {
				log.Println("==> Load > NewType", classType)
			}
			old, _ := pkg.SetCurFile(defaultGoFile, true)
			defer pkg.RestoreCurFile(old)

			baseType := base.Type()
			if proj.gameIsPtr {
				baseType = types.NewPointer(baseType)
			}

			flds := proj.embed(nil, []*types.Var{
				types.NewField(token.NoPos, pkg.Types, base.Name(), baseType, true),
			}, pkg)

			decl := pkg.NewTypeDefs().NewType(classType)
			ld.typInit = func() { // decycle
				if debugLoad {
					log.Println("==> Load > InitType", classType)
				}
				old, _ := pkg.SetCurFile(defaultGoFile, true)
				defer pkg.RestoreCurFile(old)

				decl.InitType(pkg, types.NewStruct(flds, nil))
			}
			parent.tylds = append(parent.tylds, ld)
		}
	}
	ld.methods = append(ld.methods, func() {
		old, _ := pkg.SetCurFile(defaultGoFile, true)
		defer pkg.RestoreCurFile(old)
		doInitType(ld)

		t := pkg.Ref(classType).Type()
		recv := types.NewParam(token.NoPos, pkg.Types, "this", types.NewPointer(t))
		sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
		fn, err := pkg.NewFuncWith(token.NoPos, "Main", sig, func() gotoken.Pos {
			// parent.ProjFile() never be nil here
			return parent.ProjFile().Pos()
		})
		if err != nil {
			panic(err)
		}

		parent.inits = append(parent.inits, func() {
			old, _ := pkg.SetCurFile(defaultGoFile, true)
			defer pkg.RestoreCurFile(old)

			cb := fn.BodyStart(pkg).Typ(base.Type()).MemberVal("Main")
			stk := cb.InternalStack()

			// force remove //line comments for main func
			cb.SetComments(nil, false)

			mainFn := stk.Pop()
			sigParams := mainFn.Type.(*types.Signature).Params()
			callMain := func() {
				src := parent.lookupClassNode(proj.gameClass_)
				stk.Push(mainFn)
				if _, isPtr := sigParams.At(0).Type().(*types.Pointer); isPtr {
					cb.Val(recv, src).MemberRef(base.Name()).UnaryOp(gotoken.AND)
				} else {
					cb.Val(recv, src) // template recv method
				}
			}

			iobj := 0
			narg := sigParams.Len()
			if narg > 1 {
				sprites := proj.sprites
				if len(sprites) == 1 && sprites[0].proto == "" { // no work class prototype
					sp := sprites[0]
					narg = 1 + len(sp.types)
					genWorkClasses(pkg, parent, cb, recv, sp, iobj, -1, callMain)
				} else {
					lstNames := make([]string, narg)
					for i := 1; i < narg; i++ {
						tslice := sigParams.At(i).Type()
						tn := tslice.(*types.Slice).Elem().(*types.Named)
						sp := spriteByProto(sprites, tn.Obj().Name()) // work class
						if n := len(sp.types); n > 0 {
							lstNames[i] = genWorkClasses(pkg, parent, cb, recv, sp, iobj, i, nil)
							cb.SliceLitEx(tslice, n, false).EndInit(1)
							iobj += n
						}
					}
					callMain()
					for i := 1; i < narg; i++ {
						if lstName := lstNames[i]; lstName != "" {
							cb.VarVal(lstName)
						} else {
							cb.Val(nil)
						}
					}
				}
			} else {
				callMain()
			}

			cb.Call(narg).EndStmt().End()
		})
	})
}

func genWorkClasses(
	pkg *gogen.Package, parent *pkgCtx, cb *gogen.CodeBuilder, recv *types.Var,
	sp *spxObj, iobj, ilst int, callMain func()) (lstName string) {
	const (
		indexGame     = 1
		objNamePrefix = "_xgo_obj"
		lstNamePrefix = "_xgo_lst"
	)
	embedded := (sp.feats&spriteEmbedded != 0)
	sptypes := sp.types
	for i, spt := range sptypes {
		src := parent.lookupClassNode(spt)
		spto := pkg.Ref(spt)
		objName := objNamePrefix + strconv.Itoa(iobj+i)
		cb.DefineVarStart(token.NoPos, objName).
			Val(indexGame, src).Val(recv, src).StructLit(spto.Type(), 2, true, src).
			UnaryOp(gotoken.AND).EndInit(1)
		if embedded {
			cb.Val(recv, src).MemberRef(spt, src).VarVal(objName, src).Assign(1)
		}
	}
	if ilst > 0 {
		lstName = lstNamePrefix + strconv.Itoa(ilst-1)
		cb.DefineVarStart(token.NoPos, lstName)
	} else {
		callMain()
	}
	for i, spt := range sptypes {
		src := parent.lookupClassNode(spt)
		objName := objNamePrefix + strconv.Itoa(iobj+i)
		cb.VarVal(objName, src)
	}
	return
}

func genMainFunc(pkg *gogen.Package, gameClass string) {
	if o := pkg.TryRef(gameClass); o != nil {
		// force remove //line comments for main func
		pkg.CB().SetComments(nil, false)
		// new(gameClass).Main()
		new := pkg.Builtin().Ref("new")
		pkg.NewFunc(nil, "main", nil, nil, false).BodyStart(pkg).
			Val(new).Val(o).Call(1).MemberVal("Main").Call(0).EndStmt().
			End()
	}
}

func findMethod(o types.Object, name string) *types.Func {
	if obj, ok := o.(*types.TypeName); ok {
		if t, ok := obj.Type().(*types.Named); ok {
			for i, n := 0, t.NumMethods(); i < n; i++ {
				f := t.Method(i)
				if f.Name() == name {
					return f
				}
			}
		}
	}
	return nil
}

func makeMainSig(recv *types.Var, f *types.Func) *types.Signature {
	const (
		namePrefix = "_xgo_arg"
	)
	sig := f.Type().(*types.Signature)
	in := sig.Params()
	nin := in.Len()
	pkg := recv.Pkg()
	params := make([]*types.Var, nin)
	for i := 0; i < nin; i++ {
		paramName := namePrefix + strconv.Itoa(i)
		params[i] = types.NewParam(token.NoPos, pkg, paramName, in.At(i).Type())
	}
	return types.NewSignatureType(recv, nil, nil, types.NewTuple(params...), sig.Results(), false)
}

func genClassfname(ctx *blockCtx, c *gmxClass) {
	pkg := ctx.pkg
	recv := toRecv(ctx, ctx.classRecv)
	ret := types.NewTuple(pkg.NewParam(token.NoPos, "", types.Typ[types.String]))
	pkg.NewFunc(recv, "Classfname", nil, ret, false).BodyStart(pkg).
		Val(c.clsfile).Return(1).
		End()
}

func genClassclone(ctx *blockCtx, classclone *types.Signature) {
	const (
		nameRet = "_xgo_ret"
	)
	pkg := ctx.pkg
	recv := toRecv(ctx, ctx.classRecv)
	ret := classclone.Results()
	pkg.NewFunc(recv, "Classclone", nil, ret, false).BodyStart(pkg).
		DefineVarStart(token.NoPos, nameRet).VarVal("this").Elem().EndInit(1).
		VarVal(nameRet).UnaryOp(gotoken.AND).Return(1).
		End()
}

func genGopInit(ctx *blockCtx, classDecl *ast.GenDecl) {
	pkg := ctx.pkg
	recv := toRecv(ctx, ctx.classRecv)

	cb := pkg.NewFunc(recv, "Gop_Init", nil, nil, false).BodyStart(pkg)
	ctx.cb = cb

	// Generate initialization code for each field with a value
	for _, v := range classDecl.Specs {
		spec := v.(*ast.ValueSpec)
		if len(spec.Values) == 0 {
			continue
		}

		// Handle embedded types (no names)
		if len(spec.Names) == 0 {
			continue
		}

		// For each name-value pair, generate: this.Name = value
		for i, name := range spec.Names {
			if i < len(spec.Values) {
				cb.VarVal("this").MemberVal(name.Name)
				compileExpr(ctx, spec.Values[i])
				cb.Assign(1)
			}
		}
	}

	cb.End()
}

func astEmptyEntrypoint(f *ast.File) {
	var entry = getEntrypoint(f)
	var hasEntry bool
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == entry {
				hasEntry = true
			}
		}
	}
	if !hasEntry {
		f.Decls = append(f.Decls, &ast.FuncDecl{
			Name: &ast.Ident{
				Name: entry,
			},
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
			Body:   &ast.BlockStmt{},
			Shadow: true,
		})
	}
}

func getEntrypoint(f *ast.File) string {
	switch {
	case f.IsProj:
		return "MainEntry"
	case f.IsClass:
		return "Main"
	case inMainPkg(f):
		return "main"
	default:
		return "init"
	}
}

func inMainPkg(f *ast.File) bool {
	return f.Name.Name == "main"
}

// -----------------------------------------------------------------------------
