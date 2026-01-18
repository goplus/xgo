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
	"bytes"
	"errors"
	goast "go/ast"
	gotoken "go/token"
	"go/types"
	"log"
	"math/big"
	"strconv"
	"strings"
	"syscall"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/printer"
	"github.com/goplus/xgo/token"
	tpl "github.com/goplus/xgo/tpl/ast"
	"github.com/qiniu/x/stringutil"
)

/*-----------------------------------------------------------------------------

Name context:
- varVal               (ident)
- varRef = expr        (identLHS)
- pkgRef.member        (selectorExpr)
- pkgRef.member = expr (selectorExprLHS)
- pkgRef.fn(args)      (callExpr)
- fn(args)             (callExpr)
- spx.fn(args)         (callExpr)
- this.member          (classMember)
- this.method(args)    (classMember)

Name lookup:
- local variables
- $recv members (only in class files)
- package globals (variables, constants, types, imported packages etc.)
- $spx package exports (only in class files)
- $universe package exports (including builtins)

// ---------------------------------------------------------------------------*/

const (
	clIdentCanAutoCall = 1 << iota // allow auto property
	clIdentAllowBuiltin
	clIdentLHS
	clIdentSelectorExpr // this ident is X (not Sel) of ast.SelectorExpr
	clIdentGoto
	clCallWithTwoValue
	clCommandWithoutArgs // this expr is a command without args (eg. ls)
	clCommandIdent       // this expr is a command and an ident (eg. mkdir "abc")
	clIdentInStringLitEx // this expr is an ident in a string extended literal (eg. ${PATH})
	clInCallExpr
)

const (
	objNormal = iota
	objPkgRef
	objXGoExecOrEnv

	objXGoEnv  = objXGoExecOrEnv
	objXGoExec = objXGoExecOrEnv
)

func compileIdent(ctx *blockCtx, ident *ast.Ident, flags int) (pkg gogen.PkgRef, kind int) {
	fvalue := (flags&clIdentSelectorExpr) != 0 || (flags&clIdentLHS) == 0
	cb := ctx.cb
	name := ident.Name
	if name == "_" {
		if fvalue {
			panic(ctx.newCodeError(ident.Pos(), ident.End(), "cannot use _ as value"))
		}
		cb.VarRef(nil)
		return
	}

	var recv *types.Var
	var oldo types.Object
	scope := ctx.pkg.Types.Scope()
	at, o := cb.Scope().LookupParent(name, token.NoPos)
	if o != nil {
		if at != scope && at != types.Universe { // local object
			goto find
		}
	}

	if ctx.isClass { // in an XGo class file
		if recv = classRecv(cb); recv != nil {
			cb.Val(recv)
			chkFlag := flags
			if chkFlag&clIdentSelectorExpr != 0 { // TODO: remove this condition
				chkFlag = clIdentCanAutoCall
			}
			if compileMember(ctx, ident, name, chkFlag) == nil { // class member object
				return
			}
			cb.InternalStack().PopN(1)
		}
	}

	// global object
	if ctx.loadSymbol(name) {
		o, at = scope.Lookup(name), scope
	}
	if o != nil && at != types.Universe {
		goto find
	}

	// pkgRef object
	if (flags & clIdentSelectorExpr) != 0 {
		if pi, ok := ctx.findImport(name); ok {
			if rec := ctx.recorder(); rec != nil {
				rec.Use(ident, pi.pkgName)
			}
			return pi.PkgRef, objPkgRef
		}
	}

	// function alias
	if compileFuncAlias(ctx, scope, ident, flags) {
		return
	}

	// object from import . "xxx"
	if compilePkgRef(ctx, gogen.PkgRef{}, ident, flags, objPkgRef) {
		return
	}

	// universe object
	if obj := ctx.pkg.Builtin().TryRef(name); obj != nil {
		if (flags&clIdentAllowBuiltin) == 0 && isBuiltin(o) && !strings.HasPrefix(o.Name(), "print") {
			panic(ctx.newCodeErrorf(ident.Pos(), ident.End(), "use of builtin %s not in function call", name))
		}
		oldo, o = o, obj
	} else if o == nil {
		// for support XGo_Exec, see TestSpxXGoExec
		if (clCommandIdent&flags) != 0 && recv != nil && xgoOp(cb, recv, "Exec", ident) == nil {
			kind = objXGoExec
			return
		}
		// for support XGo_Env, see TestSpxGopEnv
		if (clIdentInStringLitEx&flags) != 0 && recv != nil && xgoOp(cb, recv, "Env", ident) == nil {
			kind = objXGoEnv
			return
		}
		if (clIdentGoto & flags) != 0 {
			l := ident.Obj.Data.(*ast.Ident)
			panic(ctx.newCodeErrorf(l.Pos(), l.End(), "label %v is not defined", l.Name))
		}
		panic(ctx.newCodeErrorf(ident.Pos(), ident.End(), "undefined: %s", name))
	}

find:
	if fvalue {
		cb.Val(o, ident)
	} else {
		cb.VarRef(o, ident)
	}
	if rec := ctx.recorder(); rec != nil {
		e := cb.Get(-1)
		if oldo != nil && gogen.IsTypeEx(e.Type) { // for builtin object
			rec.recordIdent(ident, oldo)
			return
		}
		rec.recordIdent(ident, o)
	}
	return
}

/*
func compileMatrixLit(ctx *blockCtx, v *ast.MatrixLit) {
	cb := ctx.cb
	ncol := -1
	for _, elts := range v.Elts {
		switch n := len(elts); n {
		case 1:
			elt := elts[0]
			if e, ok := elt.(*ast.Ellipsis); ok {
				compileExpr(ctx, e.Elt)
				panic("TODO") // TODO(xsw): matrixLit with ellipsis
			}
			fallthrough
		default:
			if ncol < 0 {
				ncol = n
			} else if ncol != n {
				ctx.handleErrorf(elts[0].Pos(), "inconsistent matrix column count: got %v, want %v", n, ncol)
			}
			for _, elt := range elts {
				compileExpr(ctx, elt)
			}
			cb.SliceLitEx(...)
		}
	}
}
*/

func compileEnvExpr(ctx *blockCtx, v *ast.EnvExpr) {
	cb := ctx.cb
	if ctx.isClass { // in an XGo class file
		if recv := classRecv(cb); recv != nil {
			if xgoOp(cb, recv, "Env", v) == nil {
				name := v.Name
				cb.Val(name.Name, name).CallWith(1, 0, v)
				return
			}
		}
	}
	invalidVal(cb)
	ctx.handleErrorf(v.Pos(), v.End(), "operator $%v undefined", v.Name)
}

func classRecv(cb *gogen.CodeBuilder) *types.Var {
	if fn := cb.Func(); fn != nil {
		sig := fn.Ancestor().Type().(*types.Signature)
		return sig.Recv()
	}
	return nil
}

// op = Exec | Env
func xgoOp(cb *gogen.CodeBuilder, recv *types.Var, op string, src ...ast.Node) error {
	cb.Val(recv)
	kind, e := cb.Member("XGo_"+op, gogen.MemberFlagVal, src...)
	if kind == gogen.MemberInvalid {
		_, e = cb.Member("Gop_"+op, gogen.MemberFlagVal, src...)
	}
	return e
}

func isBuiltin(o types.Object) bool {
	if _, ok := o.(*types.Builtin); ok {
		return ok
	}
	return false
}

func compileMember(ctx *blockCtx, v ast.Node, name string, flags int) error {
	var mflag gogen.MemberFlag
	switch {
	case (flags & clIdentLHS) != 0:
		mflag = gogen.MemberFlagRef
	case (flags & clIdentCanAutoCall) != 0:
		mflag = gogen.MemberFlagAutoProperty
	default:
		mflag = gogen.MemberFlagMethodAlias
	}
	_, err := ctx.cb.Member(name, mflag, v)
	return err
}

func compileExprLHS(ctx *blockCtx, expr ast.Expr) {
	switch v := expr.(type) {
	case *ast.Ident:
		compileIdent(ctx, v, clIdentLHS)
	case *ast.IndexExpr:
		compileIndexExprLHS(ctx, v)
	case *ast.SelectorExpr:
		compileSelectorExprLHS(ctx, v)
	case *ast.StarExpr:
		compileStarExprLHS(ctx, v)
	default:
		panic(ctx.newCodeErrorf(v.Pos(), v.End(), "compileExprLHS failed: unknown - %T", expr))
	}
	if rec := ctx.recorder(); rec != nil {
		rec.recordExpr(ctx, expr, true)
	}
}

func twoValue(inFlags []int) bool {
	return inFlags != nil && (inFlags[0]&clCallWithTwoValue) != 0
}

func identOrSelectorFlags(inFlags []int) (flags int, cmdNoArgs bool) {
	if inFlags == nil {
		return clIdentCanAutoCall, false
	}
	flags = inFlags[0]
	if flags&clInCallExpr != 0 {
		return
	}
	if cmdNoArgs = (flags & clCommandWithoutArgs) != 0; cmdNoArgs {
		flags &^= clCommandWithoutArgs
	} else {
		flags |= clIdentCanAutoCall
	}
	return
}

func callCmdNoArgs(ctx *blockCtx, src ast.Node, panicErr bool) (err error) {
	if gogen.IsFunc(ctx.cb.InternalStack().Get(-1).Type) {
		if err = ctx.cb.CallWithEx(0, 0, src); err != nil {
			if panicErr {
				panic(err)
			}
		}
	}
	return
}

func compileExpr(ctx *blockCtx, expr ast.Expr, inFlags ...int) {
	switch v := expr.(type) {
	case *ast.Ident:
		flags, cmdNoArgs := identOrSelectorFlags(inFlags)
		if cmdNoArgs {
			flags |= clCommandIdent // for support XGo_Exec, see TestSpxXGoExec
		}
		_, kind := compileIdent(ctx, v, flags)
		if cmdNoArgs || kind == objXGoExecOrEnv {
			cb := ctx.cb
			if kind == objXGoExecOrEnv {
				cb.Val(v.Name, v)
			} else {
				err := callCmdNoArgs(ctx, expr, false)
				if err == nil {
					return
				}
				if !(ctx.isClass && tryXGoExec(cb, v)) {
					panic(err)
				}
			}
			cb.CallWith(1, 0, v)
		}
	case *ast.BasicLit:
		compileBasicLit(ctx, v)
	case *ast.CallExpr:
		flags := 0
		if inFlags != nil {
			flags = inFlags[0]
		}
		compileCallExpr(ctx, v, flags)
	case *ast.SelectorExpr:
		flags, cmdNoArgs := identOrSelectorFlags(inFlags)
		compileSelectorExpr(ctx, v, flags)
		if cmdNoArgs {
			callCmdNoArgs(ctx, expr, true)
			return
		}
	case *ast.BinaryExpr:
		compileBinaryExpr(ctx, v)
	case *ast.UnaryExpr:
		compileUnaryExpr(ctx, v, twoValue(inFlags))
	case *ast.FuncLit:
		compileFuncLit(ctx, v)
	case *ast.CompositeLit:
		compileCompositeLit(ctx, v, nil, false)
	case *ast.SliceLit:
		compileSliceLit(ctx, v, nil)
	case *ast.RangeExpr:
		compileRangeExpr(ctx, v)
	case *ast.IndexExpr:
		compileIndexExpr(ctx, v, inFlags...)
	case *ast.IndexListExpr:
		compileIndexListExpr(ctx, v, inFlags...)
	case *ast.SliceExpr:
		compileSliceExpr(ctx, v)
	case *ast.StarExpr:
		compileStarExpr(ctx, v)
	case *ast.ArrayType:
		ctx.cb.Typ(toArrayType(ctx, v), v)
	case *ast.MapType:
		ctx.cb.Typ(toMapType(ctx, v), v)
	case *ast.StructType:
		ctx.cb.Typ(toStructType(ctx, v), v)
	case *ast.ChanType:
		ctx.cb.Typ(toChanType(ctx, v), v)
	case *ast.InterfaceType:
		ctx.cb.Typ(toInterfaceType(ctx, v), v)
	case *ast.ComprehensionExpr:
		compileComprehensionExpr(ctx, v, twoValue(inFlags))
	case *ast.TypeAssertExpr:
		compileTypeAssertExpr(ctx, v, twoValue(inFlags))
	case *ast.ParenExpr:
		compileExpr(ctx, v.X, inFlags...)
	case *ast.ErrWrapExpr:
		compileErrWrapExpr(ctx, v, 0)
	case *ast.FuncType:
		ctx.cb.Typ(toFuncType(ctx, v, nil, nil), v)
	case *ast.EnvExpr:
		compileEnvExpr(ctx, v)
	/* case *ast.MatrixLit:
	compileMatrixLit(ctx, v) */
	case *ast.DomainTextLit:
		compileDomainTextLit(ctx, v)
	default:
		panic(ctx.newCodeErrorf(v.Pos(), v.End(), "compileExpr failed: unknown - %T", v))
	}
	if rec := ctx.recorder(); rec != nil {
		rec.recordExpr(ctx, expr, false)
	}
}

func compileExprOrNone(ctx *blockCtx, expr ast.Expr) {
	if expr != nil {
		compileExpr(ctx, expr)
	} else {
		ctx.cb.None()
	}
}

func compileUnaryExpr(ctx *blockCtx, v *ast.UnaryExpr, twoValue bool) {
	compileExpr(ctx, v.X)
	ctx.cb.UnaryOp(gotoken.Token(v.Op), twoValue, v)
}

func compileBinaryExpr(ctx *blockCtx, v *ast.BinaryExpr) {
	compileExpr(ctx, v.X)
	compileExpr(ctx, v.Y)
	ctx.cb.BinaryOp(gotoken.Token(v.Op), v)
}

func compileIndexExprLHS(ctx *blockCtx, v *ast.IndexExpr) {
	compileExpr(ctx, v.X)
	compileExpr(ctx, v.Index)
	ctx.cb.IndexRef(1, v)
}

func compileStarExprLHS(ctx *blockCtx, v *ast.StarExpr) { // *x = ...
	compileExpr(ctx, v.X)
	ctx.cb.ElemRef()
}

func compileStarExpr(ctx *blockCtx, v *ast.StarExpr) { // ... = *x
	compileExpr(ctx, v.X)
	ctx.cb.Star(v)
}

func compileTypeAssertExpr(ctx *blockCtx, v *ast.TypeAssertExpr, twoValue bool) {
	compileExpr(ctx, v.X)
	if v.Type == nil {
		panic("TODO: x.(type) is only used in type switch")
	}
	typ := toType(ctx, v.Type)
	ctx.cb.TypeAssert(typ, twoValue, v)
}

func compileIndexExpr(ctx *blockCtx, v *ast.IndexExpr, inFlags ...int) { // x[i]
	compileExpr(ctx, v.X, inFlags...)
	compileExpr(ctx, v.Index)
	ctx.cb.Index(1, twoValue(inFlags), v)
}

func compileIndexListExpr(ctx *blockCtx, v *ast.IndexListExpr, inFlags ...int) { // fn[t1,t2]
	compileExpr(ctx, v.X, inFlags...)
	n := len(v.Indices)
	for i := 0; i < n; i++ {
		compileExpr(ctx, v.Indices[i])
	}
	ctx.cb.Index(n, twoValue(inFlags), v)
}

func compileSliceExpr(ctx *blockCtx, v *ast.SliceExpr) { // x[i:j:k]
	compileExpr(ctx, v.X)
	compileExprOrNone(ctx, v.Low)
	compileExprOrNone(ctx, v.High)
	if v.Slice3 {
		compileExprOrNone(ctx, v.Max)
	}
	ctx.cb.Slice(v.Slice3, v)
}

func compileSelectorExprLHS(ctx *blockCtx, v *ast.SelectorExpr) {
	switch x := v.X.(type) {
	case *ast.Ident:
		if at, kind := compileIdent(ctx, x, clIdentLHS|clIdentSelectorExpr); kind != objNormal {
			ctx.cb.VarRef(at.Ref(v.Sel.Name))
			return
		}
	default:
		compileExpr(ctx, v.X)
	}
	ctx.cb.MemberRef(v.Sel.Name, v)
}

func compileSelectorExpr(ctx *blockCtx, v *ast.SelectorExpr, flags int) {
	switch x := v.X.(type) {
	case *ast.Ident:
		if at, kind := compileIdent(ctx, x, flags|clIdentCanAutoCall|clIdentSelectorExpr); kind != objNormal {
			if compilePkgRef(ctx, at, v.Sel, flags, kind) {
				return
			}
			if token.IsExported(v.Sel.Name) {
				panic(ctx.newCodeErrorf(x.Pos(), x.End(), "undefined: %s.%s", x.Name, v.Sel.Name))
			}
			panic(ctx.newCodeErrorf(x.Pos(), x.End(), "cannot refer to unexported name %s.%s", x.Name, v.Sel.Name))
		}
	default:
		compileExpr(ctx, v.X)
	}
	if err := compileMember(ctx, v, v.Sel.Name, flags); err != nil {
		panic(err)
	}
}

func compileFuncAlias(ctx *blockCtx, scope *types.Scope, x *ast.Ident, flags int) bool {
	name := x.Name
	if c := name[0]; c >= 'a' && c <= 'z' {
		name = string(rune(c)+('A'-'a')) + name[1:]
		o := scope.Lookup(name)
		if o == nil && ctx.loadSymbol(name) {
			o = scope.Lookup(name)
		}
		if o != nil {
			return identVal(ctx, x, flags, o, true)
		}
	}
	return false
}

func pkgRef(at gogen.PkgRef, name string) (o types.Object, alias bool) {
	if c := name[0]; c >= 'a' && c <= 'z' {
		name = string(rune(c)+('A'-'a')) + name[1:]
		if v := at.TryRef(name); v != nil && gogen.IsFunc(v.Type()) {
			return v, true
		}
		return
	}
	return at.TryRef(name), false
}

// allow pkg.Types to be nil
func lookupPkgRef(ctx *blockCtx, pkg gogen.PkgRef, x *ast.Ident, pkgKind int) (o types.Object, alias bool) {
	if pkg.Types != nil {
		return pkgRef(pkg, x.Name)
	}
	if pkgKind == objPkgRef {
		for _, at := range ctx.lookups {
			if o2, alias2 := pkgRef(at, x.Name); o2 != nil {
				if o != nil {
					panic(ctx.newCodeErrorf(
						x.Pos(), x.End(), "confliction: %s declared both in \"%s\" and \"%s\"",
						x.Name, at.Types.Path(), pkg.Types.Path()))
				}
				pkg, o, alias = at, o2, alias2
			}
		}
	}
	return
}

// allow at.Types to be nil
func compilePkgRef(ctx *blockCtx, at gogen.PkgRef, x *ast.Ident, flags, pkgKind int) bool {
	if v, alias := lookupPkgRef(ctx, at, x, pkgKind); v != nil {
		if (flags & clIdentLHS) != 0 {
			if rec := ctx.recorder(); rec != nil {
				rec.Use(x, v)
			}
			ctx.cb.VarRef(v, x)
			return true
		}
		return identVal(ctx, x, flags, v, alias)
	}
	return false
}

func identVal(ctx *blockCtx, x *ast.Ident, flags int, v types.Object, alias bool) bool {
	autocall := false
	if alias {
		if autocall = (flags & clIdentCanAutoCall) != 0; autocall {
			if !gogen.HasAutoProperty(v.Type()) {
				return false
			}
		}
	}
	if rec := ctx.recorder(); rec != nil {
		rec.Use(x, v)
	}
	cb := ctx.cb.Val(v, x)
	if autocall {
		cb.CallWith(0, 0, x)
	}
	return true
}

type fnType struct {
	next         *fnType
	params       *types.Tuple
	sig          *types.Signature
	base         int
	size         int
	variadic     bool
	typetype     bool
	typeparam    bool
	typeAsParams bool
}

func (p *fnType) arg(i int, ellipsis bool) types.Type {
	if i+p.base < p.size {
		return p.params.At(i + p.base).Type()
	}
	if p.variadic {
		t := p.params.At(p.size).Type()
		if ellipsis {
			return t
		}
		return t.(*types.Slice).Elem()
	}
	return nil
}

func (p *fnType) init(base int, t *types.Signature, typeAsParams bool) {
	p.base = base
	p.sig = t
	p.typeAsParams = typeAsParams
	p.params, p.variadic, p.typeparam = t.Params(), t.Variadic(), t.TypeParams() != nil
	p.size = p.params.Len()
	if p.variadic {
		p.size--
	}
}

func (p *fnType) initTypeType(t *gogen.TypeType) {
	param := types.NewParam(0, nil, "", t.Type())
	p.params, p.typetype = types.NewTuple(param), true
	p.size = 1
}

func (p *fnType) load(fnt types.Type) {
	switch v := fnt.(type) {
	case *gogen.TypeType:
		p.initTypeType(v)
	case *types.Signature:
		typ, objs := gogen.CheckSigFuncExObjects(v)
		switch typ.(type) {
		case *gogen.TyOverloadFunc, *gogen.TyOverloadMethod:
			p.initFuncs(0, objs, false)
			return
		case *gogen.TyTemplateRecvMethod:
			p.initFuncs(1, objs, false)
			return
		case *gogen.TyTypeAsParams:
			p.initFuncs(1, objs, true)
			return
		}
		p.init(0, v, false)
	}
}

func (p *fnType) initFuncs(base int, funcs []types.Object, typeAsParams bool) {
	for i, obj := range funcs {
		if sig, ok := obj.Type().(*types.Signature); ok {
			if i == 0 {
				p.init(base, sig, typeAsParams)
			} else {
				fn := &fnType{}
				fn.init(base, sig, typeAsParams)
				p.next = fn
				p = p.next
			}
		}
	}
}

func compileCallExpr(ctx *blockCtx, v *ast.CallExpr, inFlags int) {
	// If you need to confirm the callExpr format, you can turn on
	// if !v.NoParenEnd.IsValid() && !v.Rparen.IsValid() {
	// 	panic("unexpected invalid Rparen and NoParenEnd in CallExpr")
	// }
	var ifn *ast.Ident
	switch fn := v.Fun.(type) {
	case *ast.Ident:
		if v.IsCommand() { // for support XGo_Exec, see TestSpxXGoExec
			inFlags |= clCommandIdent
		}
		if _, kind := compileIdent(ctx, fn, clIdentAllowBuiltin|inFlags); kind == objXGoExec {
			args := make([]ast.Expr, 1, len(v.Args)+1)
			args[0] = toBasicLit(fn)
			args = append(args, v.Args...)
			v = &ast.CallExpr{Fun: fn, Args: args, Ellipsis: v.Ellipsis, NoParenEnd: v.NoParenEnd}
		} else {
			ifn = fn
		}
	case *ast.SelectorExpr:
		compileSelectorExpr(ctx, fn, 0)
	case *ast.ErrWrapExpr:
		if v.IsCommand() {
			callExpr := *v
			callExpr.Fun = fn.X
			ewExpr := *fn
			ewExpr.X = &callExpr
			compileErrWrapExpr(ctx, &ewExpr, inFlags)
			return
		}
		compileErrWrapExpr(ctx, fn, 0)
	default:
		compileExpr(ctx, fn, clInCallExpr)
	}
	var err error
	var stk = ctx.cb.InternalStack()
	var base = stk.Len()
	var flags gogen.InstrFlags
	var ellipsis = v.Ellipsis != gotoken.NoPos
	if ellipsis {
		flags = gogen.InstrFlagEllipsis
	}
	if (inFlags & clCallWithTwoValue) != 0 {
		flags |= gogen.InstrFlagTwoValue
	}
	pfn := stk.Get(-1)
	fnt := pfn.Type
	fn := &fnType{}
	fn.load(fnt)
	if len(v.Kwargs) > 0 { // https://github.com/goplus/xgo/issues/2443
		n := len(v.Args)
		args := make([]ast.Expr, n+1)
		if fn.variadic { // has variadic parameter
			idx := fn.size - 1
			if idx < 0 {
				panic(ctx.newCodeError(v.Pos(), v.End(), msgNoKwargsOVF))
			}
			if len(v.Args) < idx {
				panic(ctx.newCodeError(v.Pos(), v.End(), msgNoEnoughArgToKwargs))
			}
			copy(args, v.Args[:idx])
			args[idx] = mergeKwargs(ctx, v, fn.params.At(idx).Type())
			copy(args[idx+1:], v.Args[idx:])
		} else {
			copy(args, v.Args)
			args[n] = mergeKwargs(ctx, v, fn.arg(n, false))
		}
		ne := *v
		ne.Args, ne.Kwargs = args, nil
		v = &ne
	}
	for fn != nil {
		if err = compileCallArgs(ctx, pfn, fn, v, ellipsis, flags); err == nil {
			if rec := ctx.recorder(); rec != nil {
				rec.recordCallExpr(ctx, v, fnt)
			}
			return
		}
		stk.SetLen(base)
		fn = fn.next
	}
	if ifn != nil && builtinOrXGoExec(ctx, ifn, v, flags) == nil {
		return
	}
	panic(err)
}

const (
	msgNoKwargsOVF         = "keyword arguments are not supported for a function with only variadic parameters"
	msgNoEnoughArgToKwargs = "not enough arguments for function call with keyword arguments"
	msgUnexpectedKwargs    = "keyword arguments can only be used for struct or map[string]T types, but got %v"
)

func inThisPkg(ctx *blockCtx, t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		if named.Obj().Pkg() == ctx.pkg.Types {
			return true
		}
	}
	return false
}

func mergeKwargs(ctx *blockCtx, v *ast.CallExpr, t types.Type) ast.Expr {
	if t != nil {
		switch u := t.Underlying().(type) {
		case *types.Pointer:
			t = u.Elem()
			if u, ok := t.Underlying().(*types.Struct); ok {
				return mergeStructKwargs(v.Kwargs, u, inThisPkg(ctx, t))
			}
			panic(ctx.newCodeErrorf(v.Pos(), v.End(), msgUnexpectedKwargs, t))
		case *types.Struct:
			return mergeStructKwargs(v.Kwargs, u, inThisPkg(ctx, t))
		}
	}
	return mergeStringMapKwargs(v.Kwargs) // fallback to map[string]T
}

func mergeStringMapKwargs(kwargs []*ast.KwargExpr) ast.Expr {
	n := len(kwargs)
	elts := make([]ast.Expr, n)
	for i, arg := range kwargs {
		elts[i] = &ast.KeyValueExpr{
			Key:   toBasicLit(arg.Name),
			Value: arg.Value,
		}
	}
	return &ast.CompositeLit{
		Lbrace: kwargs[0].Pos() - 1,
		Elts:   elts,
		Rbrace: kwargs[n-1].End(),
	}
}

func mergeStructKwargs(kwargs []*ast.KwargExpr, u *types.Struct, inPkg bool) ast.Expr {
	n := len(kwargs)
	elts := make([]ast.Expr, n)
	for i, arg := range kwargs {
		elts[i] = &ast.KeyValueExpr{
			Key:   getFldName(arg.Name, u, inPkg),
			Value: arg.Value,
		}
	}
	return &ast.CompositeLit{
		Lbrace: kwargs[0].Pos() - 1,
		Elts:   elts,
		Rbrace: kwargs[n-1].End(),
	}
}

func getFldName(name *ast.Ident, u *types.Struct, inPkg bool) *ast.Ident {
	if name.IsExported() {
		return name
	}
	capName := stringutil.Capitalize(name.Name)
	if !inPkg {
		return &ast.Ident{NamePos: name.NamePos, Name: capName}
	}
	for i, n := 0, u.NumFields(); i < n; i++ {
		fld := u.Field(i)
		if fld.Name() == name.Name {
			return name
		}
		if fld.Exported() && fld.Name() == capName {
			return &ast.Ident{NamePos: name.NamePos, Name: capName}
		}
	}
	return name // fallback to origin name
}

func toBasicLit(fn *ast.Ident) *ast.BasicLit {
	return &ast.BasicLit{ValuePos: fn.NamePos, Kind: token.STRING, Value: strconv.Quote(fn.Name)}
}

// maybe builtin new/delete: see TestSpxNewObj, TestMayBuiltinDelete
// maybe XGo_Exec: see TestSpxXGoExec
func builtinOrXGoExec(ctx *blockCtx, ifn *ast.Ident, v *ast.CallExpr, flags gogen.InstrFlags) error {
	cb := ctx.cb
	switch name := ifn.Name; name {
	case "new", "delete":
		cb.InternalStack().PopN(1)
		cb.Val(ctx.pkg.Builtin().Ref(name), ifn)
		return fnCall(ctx, v, flags, 0)
	default:
		// for support XGo_Exec, see TestSpxXGoExec
		if v.IsCommand() && ctx.isClass && tryXGoExec(cb, ifn) {
			return fnCall(ctx, v, flags, 1)
		}
	}
	return syscall.ENOENT
}

func tryXGoExec(cb *gogen.CodeBuilder, ifn *ast.Ident) bool {
	if recv := classRecv(cb); recv != nil {
		cb.InternalStack().PopN(1)
		if xgoOp(cb, recv, "Exec", ifn) == nil {
			cb.Val(ifn.Name, ifn)
			return true
		}
	}
	return false
}

func fnCall(ctx *blockCtx, v *ast.CallExpr, flags gogen.InstrFlags, extra int) error {
	for _, arg := range v.Args {
		compileExpr(ctx, arg)
	}
	return ctx.cb.CallWithEx(len(v.Args)+extra, flags, v)
}

func compileCallArgs(ctx *blockCtx, pfn *gogen.Element, fn *fnType, v *ast.CallExpr, ellipsis bool, flags gogen.InstrFlags) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = ctx.recoverErr(r, v)
		}
	}()

	vargs := v.Args
	if fn.typeAsParams && fn.typeparam {
		n := fn.sig.TypeParams().Len()
		for i := 0; i < n; i++ {
			compileExpr(ctx, vargs[i])
		}
		args := ctx.cb.InternalStack().GetArgs(n)
		var targs []types.Type
		for i, arg := range args {
			typ := arg.Type
			t, ok := typ.(*gogen.TypeType)
			if !ok {
				return ctx.newCodeErrorf(vargs[i].Pos(), vargs[i].End(), "%v not type", ctx.LoadExpr(vargs[i]))
			}
			targs = append(targs, t.Type())
		}
		ret, err := types.Instantiate(nil, fn.sig, targs, true)
		if err != nil {
			return ctx.newCodeError(v.Pos(), v.End(), err.Error())
		}
		fn.init(1, ret.(*types.Signature), false)
		vargs = vargs[n:]
	}

	var needInferFunc bool
	for i, arg := range vargs {
		t := fn.arg(i, ellipsis)
		switch expr := arg.(type) {
		case *ast.LambdaExpr:
			if fn.typeparam {
				needInferFunc = true
				compileIdent(ctx, ast.NewIdent("nil"), 0)
				continue
			}
			sig, e := checkLambdaFuncType(ctx, expr, t, clLambaArgument, v.Fun)
			if e != nil {
				return e
			}
			if err = compileLambdaExpr(ctx, expr, sig); err != nil {
				return
			}
		case *ast.LambdaExpr2:
			if fn.typeparam {
				needInferFunc = true
				compileIdent(ctx, ast.NewIdent("nil"), 0)
				continue
			}
			sig, e := checkLambdaFuncType(ctx, expr, t, clLambaArgument, v.Fun)
			if e != nil {
				return e
			}
			if err = compileLambdaExpr2(ctx, expr, sig); err != nil {
				return
			}
		case *ast.CompositeLit:
			if err = compileCompositeLitEx(ctx, expr, t, true); err != nil {
				return
			}
		case *ast.SliceLit:
			switch t.(type) {
			case *types.Slice:
			case *types.Named:
				if _, ok := getUnderlying(ctx, t).(*types.Slice); !ok {
					t = nil
				}
			default:
				t = nil
			}
			typetype := fn.typetype && t != nil
			if typetype {
				ctx.cb.InternalStack().PopN(1)
			}
			if err = compileSliceLit(ctx, expr, t, true); err != nil {
				return
			}
			if typetype {
				return
			}
		case *ast.NumberUnitLit:
			compileNumberUnitLit(ctx, expr, t)
		default:
			compileExpr(ctx, arg)
			if sigParamLen(t) == 0 {
				cb := ctx.cb
				if nonClosure(cb.Get(-1).Type) {
					cb.ConvertToClosure()
				}
			}
		}
	}
	if needInferFunc {
		args := ctx.cb.InternalStack().GetArgs(len(v.Args))
		typ, err := gogen.InferFunc(ctx.pkg, pfn, fn.sig, nil, args, flags)
		if err != nil {
			return err
		}
		next := &fnType{}
		next.init(fn.base, typ.(*types.Signature), false)
		next.next = fn.next
		fn.next = next
		return errCallNext
	}
	return ctx.cb.CallWithEx(len(v.Args), flags, v)
}

var (
	errCallNext = errors.New("call next")
)

type clLambaFlag string

const (
	clLambaAssign   clLambaFlag = "assignment"
	clLambaField    clLambaFlag = "field value"
	clLambaArgument clLambaFlag = "argument"
)

// check lambda func type
func checkLambdaFuncType(ctx *blockCtx, lambda ast.Expr, ftyp types.Type, flag clLambaFlag, toNode ast.Node) (*types.Signature, error) {
	typ := ftyp
retry:
	switch t := typ.(type) {
	case *types.Signature:
		if l, ok := lambda.(*ast.LambdaExpr); ok {
			if len(l.Rhs) != t.Results().Len() {
				break
			}
		}
		return t, nil
	case *types.Named:
		typ = t.Underlying()
		goto retry
	}
	var to string
	if toNode != nil {
		to = " to " + ctx.LoadExpr(toNode)
	}
	return nil, ctx.newCodeErrorf(lambda.Pos(), lambda.End(), "cannot use lambda literal as type %v in %v%v", ftyp, flag, to)
}

func sigParamLen(typ types.Type) int {
retry:
	switch t := typ.(type) {
	case *types.Signature:
		return t.Params().Len()
	case *types.Named:
		typ = t.Underlying()
		goto retry
	}
	return -1
}

func nonClosure(typ types.Type) bool {
retry:
	switch t := typ.(type) {
	case *types.Signature:
		return false
	case *types.Basic:
		if t.Kind() == types.UntypedNil {
			return false
		}
	case *types.Named:
		typ = t.Underlying()
		goto retry
	}
	return true
}

func compileLambda(ctx *blockCtx, lambda ast.Expr, sig *types.Signature) {
	switch expr := lambda.(type) {
	case *ast.LambdaExpr2:
		if err := compileLambdaExpr2(ctx, expr, sig); err != nil {
			panic(err)
		}
	case *ast.LambdaExpr:
		if err := compileLambdaExpr(ctx, expr, sig); err != nil {
			panic(err)
		}
	}
}

func makeLambdaParams(ctx *blockCtx, pos, end token.Pos, lhs []*ast.Ident, in *types.Tuple) (*types.Tuple, error) {
	pkg := ctx.pkg
	n := len(lhs)
	if nin := in.Len(); n != nin {
		fewOrMany := "few"
		if n > nin {
			fewOrMany = "many"
		}
		has := make([]string, n)
		for i, v := range lhs {
			has[i] = v.Name
		}
		return nil, ctx.newCodeErrorf(
			pos, end, "too %s arguments in lambda expression\n\thave (%s)\n\twant %v", fewOrMany, strings.Join(has, ", "), in)
	}
	if n == 0 {
		return nil, nil
	}
	params := make([]*types.Var, n)
	for i, name := range lhs {
		param := pkg.NewParam(name.Pos(), name.Name, in.At(i).Type())
		params[i] = param
		if rec := ctx.recorder(); rec != nil {
			rec.Def(name, param)
		}
	}
	return types.NewTuple(params...), nil
}

func makeLambdaResults(pkg *gogen.Package, out *types.Tuple) *types.Tuple {
	nout := out.Len()
	if nout == 0 {
		return nil
	}
	results := make([]*types.Var, nout)
	for i := 0; i < nout; i++ {
		results[i] = pkg.NewParam(token.NoPos, "", out.At(i).Type())
	}
	return types.NewTuple(results...)
}

func compileLambdaExpr(ctx *blockCtx, v *ast.LambdaExpr, sig *types.Signature) error {
	pkg := ctx.pkg
	params, err := makeLambdaParams(ctx, v.Pos(), v.End(), v.Lhs, sig.Params())
	if err != nil {
		return err
	}
	results := makeLambdaResults(pkg, sig.Results())
	ctx.cb.NewClosure(params, results, false).BodyStart(pkg)
	if len(v.Lhs) > 0 {
		defNames(ctx, v.Lhs, ctx.cb.Scope())
	}
	for _, v := range v.Rhs {
		compileExpr(ctx, v)
	}
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, ctx.cb.Scope())
	}
	ctx.cb.Return(len(v.Rhs)).End(v)
	return nil
}

func compileLambdaExpr2(ctx *blockCtx, v *ast.LambdaExpr2, sig *types.Signature) error {
	pkg := ctx.pkg
	params, err := makeLambdaParams(ctx, v.Pos(), v.End(), v.Lhs, sig.Params())
	if err != nil {
		return err
	}
	results := makeLambdaResults(pkg, sig.Results())
	comments, once := ctx.cb.BackupComments()
	fn := ctx.cb.NewClosure(params, results, false)
	cb := fn.BodyStart(ctx.pkg, v.Body)
	if len(v.Lhs) > 0 {
		defNames(ctx, v.Lhs, cb.Scope())
	}
	compileStmts(ctx, v.Body.List)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, ctx.cb.Scope())
	}
	cb.End(v)
	ctx.cb.SetComments(comments, once)
	return nil
}

func compileFuncLit(ctx *blockCtx, v *ast.FuncLit) {
	cb := ctx.cb
	comments, once := cb.BackupComments()
	sig := toFuncType(ctx, v.Type, nil, nil)
	if rec := ctx.recorder(); rec != nil {
		rec.recordFuncLit(v, sig)
	}
	fn := cb.NewClosureWith(sig)
	if body := v.Body; body != nil {
		loadFuncBody(ctx, fn, body, nil, v)
		cb.SetComments(comments, once)
	}
}

func compileNumberUnitLit(ctx *blockCtx, v *ast.NumberUnitLit, expected types.Type) {
	ctx.cb.ValWithUnit(
		&goast.BasicLit{ValuePos: v.ValuePos, Kind: gotoken.Token(v.Kind), Value: v.Value},
		expected, v.Unit)
}

func compileBasicLit(ctx *blockCtx, v *ast.BasicLit) {
	cb := ctx.cb
	switch kind := v.Kind; kind {
	case token.RAT:
		val := v.Value
		bi, _ := new(big.Int).SetString(val[:len(val)-1], 10) // remove r suffix
		cb.UntypedBigInt(bi, v)
	case token.CSTRING, token.PYSTRING:
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			log.Panicln("compileBasicLit:", err)
		}
		var xstr gogen.Ref
		switch kind {
		case token.CSTRING:
			xstr = ctx.cstr()
		default:
			xstr = ctx.pystr()
		}
		cb.Val(xstr).Val(s).Call(1)
	default:
		if v.Extra == nil {
			basicLit(cb, v)
			return
		}
		compileStringLitEx(ctx, cb, v)
	}
}

func invalidVal(cb *gogen.CodeBuilder) {
	cb.Val(&gogen.Element{Type: types.Typ[types.Invalid]})
}

func basicLit(cb *gogen.CodeBuilder, v *ast.BasicLit) {
	cb.Val(&goast.BasicLit{Kind: gotoken.Token(v.Kind), Value: v.Value}, v)
}

const (
	stringutilPkgPath = "github.com/qiniu/x/stringutil"
)

func compileStringLitEx(ctx *blockCtx, cb *gogen.CodeBuilder, lit *ast.BasicLit) {
	pos := lit.ValuePos + 1
	quote := lit.Value[:1]
	parts := lit.Extra.Parts
	n := len(parts)
	if n != 1 {
		cb.Val(ctx.pkg.Import(stringutilPkgPath).Ref("Concat"))
	}
	for _, part := range parts {
		switch v := part.(type) {
		case string: // normal string literal or end with "$$"
			next := pos + token.Pos(len(v))
			if strings.HasSuffix(v, "$$") {
				v = v[:len(v)-1]
			}
			basicLit(cb, &ast.BasicLit{ValuePos: pos - 1, Value: quote + v + quote, Kind: token.STRING})
			pos = next
		case ast.Expr:
			flags := 0
			if _, ok := v.(*ast.Ident); ok {
				flags = clIdentInStringLitEx
			}
			compileExpr(ctx, v, flags)
			t := cb.Get(-1).Type
			if t.Underlying() != types.Typ[types.String] {
				if _, err := cb.Member("string", gogen.MemberFlagAutoProperty); err != nil {
					if _, e2 := cb.Member("error", gogen.MemberFlagAutoProperty); e2 != nil {
						if e, ok := err.(*gogen.CodeError); ok {
							err = ctx.newCodeErrorf(v.Pos(), v.End(), "%s.string%s", ctx.LoadExpr(v), e.Msg)
						}
						ctx.handleErr(err)
					}
				}
			}
			pos = v.End()
		default:
			panic("compileStringLitEx TODO: unexpected part")
		}
	}
	if n != 1 {
		cb.CallWith(n, 0, lit)
	}
}

const (
	tplPkgPath = "github.com/goplus/xgo/tpl"
)

// A DomainTextLit node represents a domain-specific text literal.
// https://github.com/goplus/xgo/issues/2143
//
//	domainTag`...`
//	domainTag`> arg1, arg2, ...
//	  ...
//	`
func compileDomainTextLit(ctx *blockCtx, v *ast.DomainTextLit) {
	var cb = ctx.cb
	var imp gogen.PkgRef
	var name = v.Domain.Name
	var path string
	if pi, ok := ctx.findImport(name); ok {
		imp = pi.PkgRef
		path = pi.Path()
		if path == "golang.org/x/net/html" {
			// html`...` => html.Parse(strings.NewReader(`...`))
			cb.Val(imp.Ref("Parse")).
				Val(ctx.pkg.Import("strings").Ref("NewReader")).
				Val(&goast.BasicLit{Kind: gotoken.STRING, Value: v.Value}, v).
				CallWith(1, 0, v).
				CallWith(1, 0, v)
			return
		}
	} else {
		if name == "tpl" {
			path = tplPkgPath
		} else {
			path = tplPkgPath + "/encoding/" + name
		}
		imp = ctx.pkg.Import(path)
		/* TODO(xsw):
		if imp = ctx.pkg.TryImport(path); imp.Types == nil {
			panic("compileDomainTextLit TODO: unknown domain: " + name)
		}
		*/
	}

	n := 1
	if path == tplPkgPath {
		pos := ctx.fset.Position(v.ValuePos)
		filename := relFile(ctx.relBaseDir, pos.Filename)
		cb.Val(imp.Ref("NewEx")).
			Val(&goast.BasicLit{Kind: gotoken.STRING, Value: v.Value}, v).
			Val(filename).Val(pos.Line).Val(pos.Column)
		n += 3
		if f, ok := v.Extra.(*tpl.File); ok {
			decls := f.Decls
			for _, decl := range decls {
				if r, ok := decl.(*tpl.Rule); ok {
					if expr, ok := r.RetProc.(*ast.LambdaExpr2); ok {
						cb.Val(r.Name.Name)
						sig := sigRetFunc(ctx.pkg, r.IsList())
						compileLambdaExpr2(ctx, lambdaRetFunc(expr), sig)
						n += 2
					}
				}
			}
		}
	} else {
		cb.Val(imp.Ref("New"))
		if lit, ok := v.Extra.(*ast.DomainTextLitEx); ok {
			cb.Val(lit.Raw)
			for _, arg := range lit.Args {
				compileExpr(ctx, arg)
			}
			n += len(lit.Args)
		} else {
			cb.Val(&goast.BasicLit{Kind: gotoken.STRING, Value: v.Value}, v)
		}
	}
	cb.CallWith(n, 0, v)
}

func lambdaRetFunc(expr *ast.LambdaExpr2) *ast.LambdaExpr2 {
	v := *expr
	v.Lhs = []*ast.Ident{
		{NamePos: expr.Pos(), Name: "self"},
	}
	return &v
}

func sigRetFunc(pkg *gogen.Package, isList bool) *types.Signature {
	rets := types.NewTuple(anyParam(pkg))
	var args *types.Tuple
	if isList {
		args = types.NewTuple(anySliceParam(pkg))
	} else {
		args = rets
	}
	return types.NewSignatureType(nil, nil, nil, args, rets, false)
}

func anyParam(pkg *gogen.Package) *types.Var {
	return pkg.NewParam(token.NoPos, "", gogen.TyEmptyInterface)
}

func anySliceParam(pkg *gogen.Package) *types.Var {
	return pkg.NewParam(token.NoPos, "", types.NewSlice(gogen.TyEmptyInterface))
}

const (
	compositeLitVal    = 0
	compositeLitKeyVal = 1
)

func checkCompositeLitElts(elts []ast.Expr) (kind int) {
	for _, elt := range elts {
		if _, ok := elt.(*ast.KeyValueExpr); ok {
			return compositeLitKeyVal
		}
	}
	return compositeLitVal
}

func compileCompositeLitElts(ctx *blockCtx, elts []ast.Expr, kind int, expected *kvType) error {
	for _, elt := range elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.CompositeLit); ok && key.Type == nil {
				compileCompositeLit(ctx, key, expected.Key(), false)
			} else {
				compileExpr(ctx, kv.Key)
			}
			err := compileCompositeLitElt(ctx, kv.Value, expected.Elem(), clLambaAssign, kv.Key)
			if err != nil {
				return err
			}
		} else {
			if kind == compositeLitKeyVal {
				ctx.cb.None()
			}
			err := compileCompositeLitElt(ctx, elt, expected.Elem(), clLambaAssign, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func compileCompositeLitElt(ctx *blockCtx, e ast.Expr, typ types.Type, flag clLambaFlag, toNode ast.Node) error {
	switch v := unparen(e).(type) {
	case *ast.LambdaExpr, *ast.LambdaExpr2:
		sig, err := checkLambdaFuncType(ctx, v, typ, flag, toNode)
		if err != nil {
			return err
		}
		compileLambda(ctx, v, sig)
	case *ast.SliceLit:
		compileSliceLit(ctx, v, typ)
	case *ast.CompositeLit:
		compileCompositeLit(ctx, v, typ, false)
	default:
		compileExpr(ctx, v)
	}
	return nil
}

func unparen(x ast.Expr) ast.Expr {
	if e, ok := x.(*ast.ParenExpr); ok {
		return e.X
	}
	return x
}

func compileStructLit(ctx *blockCtx, elts []ast.Expr, t *types.Struct, typ types.Type, src *ast.CompositeLit) error {
	for idx, elt := range elts {
		if idx >= t.NumFields() {
			return ctx.newCodeErrorf(elt.Pos(), elt.End(), "too many values in %v{...}", typ)
		}
		err := compileCompositeLitElt(ctx, elt, t.Field(idx).Type(), clLambaField, nil)
		if err != nil {
			return err
		}
	}
	ctx.cb.StructLit(typ, len(elts), false, src)
	return nil
}

func compileStructLitInKeyVal(ctx *blockCtx, elts []ast.Expr, t *types.Struct, typ types.Type, src *ast.CompositeLit) error {
	for _, elt := range elts {
		kv := elt.(*ast.KeyValueExpr)
		name := kv.Key.(*ast.Ident)
		idx := lookupField(t, name.Name)
		if idx >= 0 {
			ctx.cb.Val(idx)
		} else {
			src := ctx.LoadExpr(name)
			return ctx.newCodeErrorf(name.Pos(), name.End(), "%s undefined (type %v has no field or method %s)", src, typ, name.Name)
		}
		if rec := ctx.recorder(); rec != nil {
			rec.Use(name, t.Field(idx))
		}
		err := compileCompositeLitElt(ctx, kv.Value, t.Field(idx).Type(), clLambaField, kv.Key)
		if err != nil {
			return err
		}
	}
	ctx.cb.StructLit(typ, len(elts)<<1, true, src)
	return nil
}

func lookupField(t *types.Struct, name string) int {
	for i, n := 0, t.NumFields(); i < n; i++ {
		if fld := t.Field(i); fld.Name() == name {
			return i
		}
	}
	return -1
}

type kvType struct {
	underlying types.Type
	key, val   types.Type
	cached     bool
}

func (p *kvType) required() *kvType {
	if !p.cached {
		p.cached = true
		switch t := p.underlying.(type) {
		case *types.Slice:
			p.key, p.val = types.Typ[types.Int], t.Elem()
		case *types.Array:
			p.key, p.val = types.Typ[types.Int], t.Elem()
		case *types.Map:
			p.key, p.val = t.Key(), t.Elem()
		}
	}
	return p
}

func (p *kvType) Key() types.Type {
	return p.required().key
}

func (p *kvType) Elem() types.Type {
	return p.required().val
}

func getUnderlying(ctx *blockCtx, typ types.Type) types.Type {
	u := typ.Underlying()
	if u == nil {
		if t, ok := typ.(*types.Named); ok {
			ctx.loadNamed(ctx.pkg, t)
			u = t.Underlying()
		}
	}
	return u
}

func compileCompositeLit(ctx *blockCtx, v *ast.CompositeLit, expected types.Type, mapOrStructOnly bool) {
	if err := compileCompositeLitEx(ctx, v, expected, mapOrStructOnly); err != nil {
		panic(err)
	}
}

// mapOrStructOnly means only map/struct can omit type
func compileCompositeLitEx(ctx *blockCtx, v *ast.CompositeLit, expected types.Type, mapOrStructOnly bool) error {
	var hasPtr bool
	var typ, underlying types.Type
	var kind = checkCompositeLitElts(v.Elts)
	if v.Type != nil {
		typ = toType(ctx, v.Type)
		underlying = getUnderlying(ctx, typ)
		// Auto-reference typed composite literal when expected type is pointer
		if expected != nil {
			if t, ok := expected.(*types.Pointer); ok {
				telem := t.Elem()
				if types.Identical(typ, telem) {
					hasPtr = true
				}
			}
		}
	} else if expected != nil {
		if t, ok := expected.(*types.Pointer); ok {
			telem := t.Elem()
			tu := getUnderlying(ctx, telem)
			if _, ok := tu.(*types.Struct); ok { // struct pointer
				typ, underlying, hasPtr = telem, tu, true
			}
		} else if tu := getUnderlying(ctx, expected); !mapOrStructOnly || isMapOrStruct(tu) {
			typ, underlying = expected, tu
		}
	}
	if t, ok := underlying.(*types.Struct); ok {
		var err error
		if kind == compositeLitKeyVal {
			err = compileStructLitInKeyVal(ctx, v.Elts, t, typ, v)
		} else {
			err = compileStructLit(ctx, v.Elts, t, typ, v)
		}
		if err != nil {
			return err
		}
	} else {
		err := compileCompositeLitElts(ctx, v.Elts, kind, &kvType{underlying: underlying})
		if err != nil {
			return err
		}
		n := len(v.Elts)
		switch underlying.(type) {
		case *types.Slice:
			ctx.cb.SliceLitEx(typ, n<<kind, kind == compositeLitKeyVal, v)
		case *types.Array:
			ctx.cb.ArrayLitEx(typ, n<<kind, kind == compositeLitKeyVal, v)
		case *types.Map:
			if kind == compositeLitVal && n > 0 {
				return ctx.newCodeError(v.Pos(), v.End(), "missing key in map literal")
			}
			if err := compileMapLitEx(ctx, typ, n, v); err != nil {
				return err
			}
		default:
			if kind == compositeLitVal && n > 0 {
				return ctx.newCodeErrorf(v.Pos(), v.End(), "invalid composite literal type %v", typ)
			}
			if err := compileMapLitEx(ctx, nil, n, v); err != nil {
				return err
			}
		}
	}
	if hasPtr {
		ctx.cb.UnaryOp(gotoken.AND)
		typ = expected
	}
	if rec := ctx.recorder(); rec != nil {
		rec.recordCompositeLit(v, typ)
	}
	return nil
}

func compileMapLitEx(ctx *blockCtx, typ types.Type, n int, v *ast.CompositeLit) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ctx.newCodeError(v.Pos(), v.End(), "invalid map literal")
		}
	}()
	err = ctx.cb.MapLitEx(typ, n<<1, v)
	return
}

func isMapOrStruct(tu types.Type) bool {
	switch tu.(type) {
	case *types.Struct:
		return true
	case *types.Map:
		return true
	}
	return false
}

func compileSliceLit(ctx *blockCtx, v *ast.SliceLit, typ types.Type, noPanic ...bool) (err error) {
	if noPanic != nil {
		defer func() {
			if e := recover(); e != nil { // TODO: don't use defer to capture error
				err = ctx.recoverErr(e, v)
			}
		}()
	}
	n := len(v.Elts)
	for _, elt := range v.Elts {
		compileExpr(ctx, elt)
	}
	if isSpecificSliceType(ctx, typ) {
		ctx.cb.SliceLitEx(typ, n, false, v)
	} else {
		ctx.cb.SliceLitEx(nil, n, false, v)
	}
	return
}

func compileRangeExpr(ctx *blockCtx, v *ast.RangeExpr) {
	pkg, cb := ctx.pkg, ctx.cb
	cb.Val(pkg.Builtin().Ref("newRange"))
	if v.First == nil {
		ctx.cb.Val(0, v)
	} else {
		compileExpr(ctx, v.First)
	}
	compileExpr(ctx, v.Last)
	if v.Expr3 == nil {
		ctx.cb.Val(1, v)
	} else {
		compileExpr(ctx, v.Expr3)
	}
	cb.Call(3)
}

const (
	comprehensionInvalid = iota
	comprehensionList
	comprehensionMap
	comprehensionSelect
)

func comprehensionKind(v *ast.ComprehensionExpr) int {
	switch v.Tok {
	case token.LBRACK: // [
		return comprehensionList
	case token.LBRACE: // {
		if _, ok := v.Elt.(*ast.KeyValueExpr); ok {
			return comprehensionMap
		}
		return comprehensionSelect
	}
	panic("TODO: invalid comprehensionExpr")
}

// [expr for k, v <- container, cond]
// {for k, v <- container, cond}
// {expr for k, v <- container, cond}
// {kexpr: vexpr for k, v <- container, cond}
func compileComprehensionExpr(ctx *blockCtx, v *ast.ComprehensionExpr, twoValue bool) {
	const (
		nameOk  = "_xgo_ok"
		nameRet = "_xgo_ret"
	)
	kind := comprehensionKind(v)
	pkg, cb := ctx.pkg, ctx.cb
	var results *types.Tuple
	var ret *gogen.Param
	if v.Elt == nil {
		boolean := pkg.NewParam(token.NoPos, nameOk, types.Typ[types.Bool])
		results = types.NewTuple(boolean)
	} else {
		ret = pkg.NewAutoParam(nameRet)
		if kind == comprehensionSelect && twoValue {
			boolean := pkg.NewParam(token.NoPos, nameOk, types.Typ[types.Bool])
			results = types.NewTuple(ret, boolean)
		} else {
			results = types.NewTuple(ret)
		}
	}
	cb.NewClosure(nil, results, false).BodyStart(pkg)
	if kind == comprehensionMap {
		cb.VarRef(ret).ZeroLit(ret.Type()).Assign(1)
	}
	end := 0
	for i := len(v.Fors) - 1; i >= 0; i-- {
		names := make([]string, 0, 2)
		defineNames := make([]*ast.Ident, 0, 2)
		forStmt := v.Fors[i]
		if forStmt.Key != nil {
			names = append(names, forStmt.Key.Name)
			defineNames = append(defineNames, forStmt.Key)
		} else {
			names = append(names, "_")
		}
		names = append(names, forStmt.Value.Name)
		defineNames = append(defineNames, forStmt.Value)
		cb.ForRange(names...)
		compileExpr(ctx, forStmt.X)
		cb.RangeAssignThen(forStmt.TokPos)
		defNames(ctx, defineNames, cb.Scope())
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(forStmt, cb.Scope())
		}
		if forStmt.Cond != nil {
			cb.If()
			if forStmt.Init != nil {
				compileStmt(ctx, forStmt.Init)
			}
			compileExpr(ctx, forStmt.Cond)
			cb.Then()
			end++
		}
		end++
	}
	switch kind {
	case comprehensionList:
		// _xgo_ret = append(_xgo_ret, elt)
		cb.VarRef(ret)
		cb.Val(pkg.Builtin().Ref("append"))
		cb.Val(ret)
		compileExpr(ctx, v.Elt)
		cb.Call(2).Assign(1)
	case comprehensionMap:
		// _xgo_ret[key] = val
		cb.Val(ret)
		kv := v.Elt.(*ast.KeyValueExpr)
		compileExpr(ctx, kv.Key)
		cb.IndexRef(1)
		compileExpr(ctx, kv.Value)
		cb.Assign(1)
	default:
		if v.Elt == nil {
			// return true
			cb.Val(true)
			cb.Return(1)
		} else {
			// return elt, true
			compileExpr(ctx, v.Elt)
			n := 1
			if twoValue {
				cb.Val(true)
				n++
			}
			cb.Return(n)
		}
	}
	for i := 0; i < end; i++ {
		cb.End()
	}
	cb.Return(0).End().Call(0)
}

const (
	errorPkgPath = "github.com/qiniu/x/errors"
)

var (
	tyError = types.Universe.Lookup("error").Type()
)

func compileErrWrapExpr(ctx *blockCtx, v *ast.ErrWrapExpr, inFlags int) {
	const (
		nameErr = "_xgo_err"
		nameRet = "_xgo_ret"
	)
	pkg, cb := ctx.pkg, ctx.cb
	useClosure := v.Tok == token.NOT || v.Default != nil
	if !useClosure && (cb.Scope().Parent() == types.Universe) {
		panic("TODO: can't use expr? in global")
	}
	expr := v.X
	switch expr.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		expr = &ast.CallExpr{Fun: expr, NoParenEnd: expr.End()}
	}
	compileExpr(ctx, expr, inFlags)
	x := cb.InternalStack().Pop()
	n := 0
	results, ok := x.Type.(*types.Tuple)
	if ok {
		n = results.Len() - 1
	}

	var ret []*types.Var
	if n > 0 {
		i, retName := 0, nameRet
		ret = make([]*gogen.Param, n)
		for {
			ret[i] = pkg.NewAutoParam(retName)
			i++
			if i >= n {
				break
			}
			retName = nameRet + strconv.Itoa(i+1)
		}
	}
	sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(ret...), false)
	if useClosure {
		cb.NewClosureWith(sig).BodyStart(pkg)
	} else {
		cb.CallInlineClosureStart(sig, 0, false)
	}

	cb.NewVar(tyError, nameErr)
	err := cb.Scope().Lookup(nameErr)

	for _, retVar := range ret {
		cb.VarRef(retVar)
	}
	cb.VarRef(err)
	cb.InternalStack().Push(x)
	cb.Assign(n+1, 1)

	cb.If().Val(err).CompareNil(gotoken.NEQ).Then()
	if v.Default == nil {
		pos := pkg.Fset.Position(v.Pos())
		curFn := cb.Func().Ancestor()
		curFnName := curFn.Name()
		if curFnName == "" {
			curFnName = "main"
		}

		cb.VarRef(err).
			Val(pkg.Import(errorPkgPath).Ref("NewFrame")).
			Val(err).
			Val(sprintAst(pkg.Fset, v.X)).
			Val(relFile(ctx.relBaseDir, pos.Filename)).
			Val(pos.Line).
			Val(curFn.Pkg().Name() + "." + curFnName).
			Call(5).
			Assign(1)
	}

	if v.Tok == token.NOT { // expr!
		cb.Val(pkg.Builtin().Ref("panic")).Val(err).Call(1).EndStmt()
	} else if v.Default == nil { // expr?
		cb.Val(err).ReturnErr(true)
	} else { // expr?:val
		compileExpr(ctx, v.Default)
		cb.Return(1)
	}
	cb.End().Return(0).End()
	if useClosure {
		cb.Call(0)
	}
}

func sprintAst(fset *token.FileSet, x ast.Node) string {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, fset, x)
	if err != nil {
		panic("Unexpected error: " + err.Error())
	}

	return buf.String()
}

// -----------------------------------------------------------------------------
