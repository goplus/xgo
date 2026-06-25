/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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
	"fmt"
	"go/types"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

const decoPrefix = "_xgodeco_"

// lookupDecorator looks up a decorator function by name in the current package
// and framework packages. Follows XGo naming convention (lowercase → uppercase).
func lookupDecorator(ctx *blockCtx, nameIdent *ast.Ident) types.Object {
	name := nameIdent.Name
	scope := ctx.pkg.Types.Scope()

	// Try exact name in current package.
	ctx.loadSymbol(name)
	if o := scope.Lookup(name); o != nil && gogen.IsFunc(o.Type()) {
		return o
	}

	// Try uppercase version (XGo convention: @log → Log).
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		upper := string(rune(name[0])-('a'-'A')) + name[1:]
		ctx.loadSymbol(upper)
		if o := scope.Lookup(upper); o != nil && gogen.IsFunc(o.Type()) {
			return o
		}
	}

	// Try from lookups (class framework packages).
	for _, at := range ctx.lookups {
		if o, _ := pkgRef(at, name); o != nil {
			return o
		}
	}
	return nil
}

// getDecoratorForm returns which form the decorator function uses:
//
//	1  if the last parameter is func()
//	2  if the last parameter is func() error
//	0  if invalid
func getDecoratorForm(decorObj types.Object) int {
	sig, ok := decorObj.Type().(*types.Signature)
	if !ok {
		return 0
	}
	params := sig.Params()
	n := params.Len()
	if n == 0 {
		return 0
	}
	fnSig, ok := params.At(n - 1).Type().(*types.Signature)
	if !ok {
		return 0
	}
	if fnSig.Params().Len() != 0 {
		return 0
	}
	results := fnSig.Results()
	if results.Len() == 0 {
		return 1 // func()
	}
	if results.Len() == 1 && results.At(0).Type() == gogen.TyError {
		return 2 // func() error
	}
	return 0
}

// findErrorResult returns the last error-typed result variable, or nil.
func findErrorResult(results *types.Tuple) *types.Var {
	for i := results.Len() - 1; i >= 0; i-- {
		if v := results.At(i); v.Type() == gogen.TyError {
			return v
		}
	}
	return nil
}

// namedResults returns a *types.Tuple where every result variable is named.
// Anonymous results (_r0, _r1, ...) get auto-generated names so they can be
// referenced inside the wrapper closure.
func namedResults(typPkg *types.Package, results *types.Tuple) *types.Tuple {
	n := results.Len()
	if n == 0 {
		return results
	}
	needRename := false
	for i := 0; i < n; i++ {
		v := results.At(i)
		if v.Name() == "" || v.Name() == "_" {
			needRename = true
			break
		}
	}
	if !needRename {
		return results
	}
	vars := make([]*types.Var, n)
	for i := 0; i < n; i++ {
		v := results.At(i)
		name := v.Name()
		if name == "" || name == "_" {
			name = fmt.Sprintf("_r%d", i)
		}
		vars[i] = types.NewParam(v.Pos(), typPkg, name, v.Type())
	}
	return types.NewTuple(vars...)
}

// loadDecoratedFunc handles the case where d has one or more decorators.
// It creates _xgodeco_<name> with the original body, and <name> as a wrapper.
func loadDecoratedFunc(ctx *blockCtx, recv *types.Var, name string, d *ast.FuncDecl, genBody bool) {
	// Reject generic decorated functions.
	if d.Type.TypeParams != nil && len(d.Type.TypeParams.List) > 0 {
		ctx.handleErrorf(d.Decorators[0].At, d.Name.End(),
			"cannot apply decorators to generic function %s", name)
		return
	}

	pkg := ctx.pkg
	internalName := decoPrefix + name

	// Build the public wrapper signature (with named results for closure capture).
	wrapperSig := buildWrapperSig(ctx, recv, d)

	var recvTypePos func() token.Pos
	if d.Recv != nil && len(d.Recv.List) > 0 {
		recvTypePos = func() token.Pos { return d.Recv.List[0].Type.Pos() }
	}

	if genBody {
		// Create _xgodeco_<name> FIRST so it appears before the wrapper in output.
		internalDecl := &ast.FuncDecl{
			Doc:     d.Doc,
			Recv:    d.Recv,
			Name:    &ast.Ident{NamePos: d.Name.NamePos, Name: internalName},
			Type:    d.Type,
			Body:    d.Body,
			IsClass: d.IsClass,
			Static:  d.Static,
		}
		loadFunc(ctx, recv, internalName, internalDecl, true)
	}

	// Create the public wrapper function.
	wrapperFn, err := pkg.NewFuncWith(d.Name.Pos(), name, wrapperSig, recvTypePos)
	if err != nil {
		ctx.handleErr(err)
		return
	}
	commentFunc(ctx, wrapperFn, d)
	if rec := ctx.recorder(); rec != nil {
		rec.Def(d.Name, wrapperFn.Func)
		if recv == nil && name != "_" {
			ctx.fileScope.Insert(wrapperFn.Func)
		}
	}

	if !genBody {
		return
	}

	// Generate wrapper body.
	doGenWrapper := func() {
		generateDecoratorWrapper(ctx, wrapperFn, wrapperSig, internalName, d)
	}

	if recv != nil {
		file := pkg.CurFile()
		ctx.inits = append(ctx.inits, func() {
			old := pkg.RestoreCurFile(file)
			doGenWrapper()
			pkg.RestoreCurFile(old)
		})
	} else {
		doGenWrapper()
	}
}

// buildWrapperSig constructs the signature for the public wrapper function.
// Named results are ensured so they can be captured by the inner closure.
func buildWrapperSig(ctx *blockCtx, recv *types.Var, d *ast.FuncDecl) *types.Signature {
	sig := toFuncType(ctx, d.Type, recv, d)
	named := namedResults(ctx.pkg.Types, sig.Results())
	if named == sig.Results() {
		return sig // no rename needed
	}
	return types.NewSignatureType(recv, nil, nil, sig.Params(), named, sig.Variadic())
}

// generateDecoratorWrapper builds the body of the public wrapper function:
//
//	func Foo(args...) (rets...) {
//	    Outer(func() {
//	        Inner(constArg, func() error {
//	            rets... = _xgodeco_Foo(args...)
//	            return err / nil
//	        })
//	    })
//	    return
//	}
func generateDecoratorWrapper(ctx *blockCtx, wrapperFn *gogen.Func, sig *types.Signature, internalName string, d *ast.FuncDecl) {
	cb := wrapperFn.BodyStart(ctx.pkg, d)

	decorators := d.Decorators

	// Resolve all decorator objects and forms up-front.
	decorObjs := make([]types.Object, len(decorators))
	forms := make([]int, len(decorators))
	for i, deco := range decorators {
		nameIdent, ok := deco.Fun.(*ast.Ident)
		if !ok {
			ctx.handleErrorf(deco.At, deco.End(),
				"decorator name must be a simple identifier")
			cb.End(d)
			return
		}
		obj := lookupDecorator(ctx, nameIdent)
		if obj == nil {
			ctx.handleErrorf(nameIdent.Pos(), nameIdent.End(),
				"undefined decorator: %s", nameIdent.Name)
			cb.End(d)
			return
		}
		form := getDecoratorForm(obj)
		if form == 0 {
			ctx.handleErrorf(nameIdent.Pos(), nameIdent.End(),
				"invalid decorator %s: last parameter must be func() or func() error",
				nameIdent.Name)
			cb.End(d)
			return
		}
		// Validate argument count.
		decorSig := obj.Type().(*types.Signature)
		nConstArgs := decorSig.Params().Len() - 1
		if nConstArgs != len(deco.Args) {
			ctx.handleErrorf(deco.At, deco.End(),
				"decorator %s expects %d argument(s), got %d",
				nameIdent.Name, nConstArgs, len(deco.Args))
			cb.End(d)
			return
		}
		decorObjs[i] = obj
		forms[i] = form
	}

	// Build the nested decorator call chain starting from index 0 (outermost).
	buildDecoCall(ctx, cb, sig, decorators, decorObjs, forms, internalName, 0)
	cb.EndStmt()

	// Bare return (consumes named result variables set by the inner closures).
	if sig.Results().Len() > 0 {
		cb.Return(0)
	}

	cb.End(d)
}

// buildDecoCall emits the call to decorators[idx] (and recursively any inner
// decorators), pushing the result onto the CodeBuilder stack as an ExprStmt-ready
// call expression.  The caller is responsible for calling EndStmt() once for the
// outermost call.
func buildDecoCall(ctx *blockCtx, cb *gogen.CodeBuilder, sig *types.Signature,
	decorators []*ast.FuncDecorator, decorObjs []types.Object, forms []int,
	internalName string, idx int) {

	pkg := ctx.pkg
	deco := decorators[idx]
	decorObj := decorObjs[idx]
	form := forms[idx]
	decorSig := decorObj.Type().(*types.Signature)
	nConstArgs := decorSig.Params().Len() - 1

	// Push decorator function.
	cb.Val(decorObj)

	// Push constant arguments.
	for i := 0; i < nConstArgs; i++ {
		compileExpr(ctx, 1, deco.Args[i])
	}

	// Build the closure for this decorator level.
	var closureSig *types.Signature
	if form == 1 {
		closureSig = types.NewSignatureType(nil, nil, nil, nil, nil, false) // func()
	} else {
		errResult := types.NewParam(token.NoPos, pkg.Types, "", gogen.TyError)
		closureSig = types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(errResult), false) // func() error
	}

	innerFn := cb.NewClosureWith(closureSig)
	innerFn.BodyStart(pkg)

	if idx == len(decorators)-1 {
		// Innermost: call _xgodeco_<name> and assign results.
		buildInnerCallBody(ctx, cb, sig, internalName, form)
	} else {
		// Recurse for the next (inner) decorator.
		buildDecoCall(ctx, cb, sig, decorators, decorObjs, forms, internalName, idx+1)
		cb.EndStmt()
	}

	cb.End(deco) // end inner closure → pushes closure onto stack

	// Call decorator(constArgs..., closure).
	cb.Call(nConstArgs + 1)
}

// buildInnerCallBody emits the body of the innermost closure:
//
//	rets... = _xgodeco_Foo(args...)   [if nResults > 0]
//	_xgodeco_Foo(args...)             [if nResults == 0]
//	return err / nil                  [only for Form 2]
func buildInnerCallBody(ctx *blockCtx, cb *gogen.CodeBuilder, sig *types.Signature,
	internalName string, outerForm int) {

	results := sig.Results()
	nResults := results.Len()
	params := sig.Params()
	nParams := params.Len()
	variadic := sig.Variadic()
	recv := sig.Recv()

	if nResults > 0 {
		// LHS: push all named result variables.
		for i := 0; i < nResults; i++ {
			cb.VarRef(results.At(i))
		}
	}

	// Push function (or method via receiver).
	if recv != nil {
		cb.Val(recv).MemberVal(internalName, 0)
	} else {
		internalFn := ctx.pkg.Types.Scope().Lookup(internalName)
		cb.Val(internalFn)
	}

	// Push parameters.
	for i := 0; i < nParams; i++ {
		cb.Val(params.At(i))
	}

	// Call.
	cb.Call(nParams, variadic)

	if nResults > 0 {
		cb.Assign(nResults, 1)
	} else {
		cb.EndStmt()
	}

	// For Form 2 closures (func() error), emit a return statement.
	if outerForm == 2 {
		if errVar := findErrorResult(results); errVar != nil {
			cb.Val(errVar).Return(1)
		} else {
			cb.ZeroLit(gogen.TyError).Return(1)
		}
	}
}
