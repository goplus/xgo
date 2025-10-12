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

package cl

import (
	"go/types"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl/internal/typesalias"
	"github.com/goplus/xgo/token"
)

func toTermList(ctx *blockCtx, expr ast.Expr) []*types.Term {
retry:
	switch v := expr.(type) {
	case *ast.UnaryExpr:
		if v.Op != token.TILDE {
			panic(ctx.newCodeErrorf(v.Pos(), v.End(), "invalid op %v must ~", v.Op))
		}
		return []*types.Term{types.NewTerm(true, toType(ctx, v.X))}
	case *ast.BinaryExpr:
		if v.Op != token.OR {
			panic(ctx.newCodeErrorf(v.Pos(), v.End(), "invalid op %v must |", v.Op))
		}
		return append(toTermList(ctx, v.X), toTermList(ctx, v.Y)...)
	case *ast.ParenExpr:
		expr = v.X
		goto retry
	}
	return []*types.Term{types.NewTerm(false, toType(ctx, expr))}
}

func toBinaryExprType(ctx *blockCtx, v *ast.BinaryExpr) types.Type {
	return types.NewInterfaceType(nil, []types.Type{types.NewUnion(toTermList(ctx, v))})
}

func toUnaryExprType(ctx *blockCtx, v *ast.UnaryExpr) types.Type {
	return types.NewInterfaceType(nil, []types.Type{types.NewUnion(toTermList(ctx, v))})
}

func toTypeParams(ctx *blockCtx, params *ast.FieldList) []*types.TypeParam {
	if params == nil {
		return nil
	}
	return collectTypeParams(ctx, params)
}

func recvTypeParams(ctx *blockCtx, typ ast.Expr, named *types.Named) (tparams []*types.TypeParam) {
	orgTypeParams := named.TypeParams()
	if orgTypeParams == nil {
		return nil
	}
L:
	for {
		switch t := typ.(type) {
		case *ast.ParenExpr:
			typ = t.X
		case *ast.StarExpr:
			typ = t.X
		default:
			break L
		}
	}
	switch t := typ.(type) {
	case *ast.IndexExpr:
		if orgTypeParams.Len() != 1 {
			panic(ctx.newCodeErrorf(typ.Pos(), typ.End(), "got 1 type parameter, but receiver base type declares %v", orgTypeParams.Len()))
		}
		v := t.Index.(*ast.Ident)
		tp := orgTypeParams.At(0)
		obj := types.NewTypeName(v.Pos(), tp.Obj().Pkg(), v.Name, nil)
		tparams = []*types.TypeParam{types.NewTypeParam(obj, tp.Constraint())}
	case *ast.IndexListExpr:
		n := len(t.Indices)
		if n != orgTypeParams.Len() {
			panic(ctx.newCodeErrorf(typ.Pos(), typ.End(), "got %v arguments but %v type parameters", n, orgTypeParams.Len()))
		}
		tparams = make([]*types.TypeParam, n)
		for i := 0; i < n; i++ {
			v := t.Indices[i].(*ast.Ident)
			tp := orgTypeParams.At(i)
			obj := types.NewTypeName(v.Pos(), tp.Obj().Pkg(), v.Name, nil)
			tparams[i] = types.NewTypeParam(obj, tp.Constraint())
		}
	default:
		panic(ctx.newCodeErrorf(typ.Pos(), typ.End(), "cannot use generic type %v without instantiation", named))
	}
	return
}

func toFuncType(ctx *blockCtx, typ *ast.FuncType, recv *types.Var, d *ast.FuncDecl) *types.Signature {
	var typeParams []*types.TypeParam
	if recv != nil && d.Recv != nil {
		typ := recv.Type()
		if pt, ok := typ.(*types.Pointer); ok {
			typ = pt.Elem()
		}
		typeParams = recvTypeParams(ctx, d.Recv.List[0].Type, typ.(*types.Named))
	} else {
		typeParams = toTypeParams(ctx, typ.TypeParams)
	}
	if len(typeParams) > 0 {
		ctx.tlookup = &typeParamLookup{typeParams}
		defer func() {
			ctx.tlookup = nil
		}()
	}
	params, variadic := toParams(ctx, typ.Params.List)
	results := toResults(ctx, typ.Results)
	if recv != nil {
		return types.NewSignatureType(recv, typeParams, nil, params, results, variadic)
	}
	return types.NewSignatureType(recv, nil, typeParams, params, results, variadic)
}

type typeParamLookup struct {
	typeParams []*types.TypeParam
}

func (p *typeParamLookup) Lookup(name string) *types.TypeParam {
	for _, t := range p.typeParams {
		tname := t.Obj().Name()
		if tname != "_" && name == tname {
			return t
		}
	}
	return nil
}

func initType(ctx *blockCtx, named *types.Named, spec *ast.TypeSpec) {
	typeParams := toTypeParams(ctx, spec.TypeParams)
	if len(typeParams) > 0 {
		named.SetTypeParams(typeParams)
		ctx.tlookup = &typeParamLookup{typeParams}
		defer func() {
			ctx.tlookup = nil
		}()
	}
	org := ctx.inInst
	ctx.inInst = 0
	defer func() {
		ctx.inInst = org
	}()
	typ := toType(ctx, spec.Type)
retry:
	switch t := typ.(type) {
	case *types.Named:
		typ = getUnderlying(ctx, t)
	case *typesalias.Alias:
		typ = typesalias.Unalias(t)
		goto retry
	}
	named.SetUnderlying(typ)
}

func getRecvType(expr ast.Expr) (typ ast.Expr, ptr bool, ok bool) {
	typ = expr
L:
	for {
		switch t := typ.(type) {
		case *ast.ParenExpr:
			typ = t.X
		case *ast.StarExpr:
			if ptr {
				ok = false
				return
			}
			ptr = true
			typ = t.X
		default:
			break L
		}
	}
	switch t := typ.(type) {
	case *ast.IndexExpr:
		typ = t.X
	case *ast.IndexListExpr:
		typ = t.X
	}
	ok = true
	return
}

func collectTypeParams(ctx *blockCtx, list *ast.FieldList) []*types.TypeParam {
	var tparams []*types.TypeParam
	// Declare type parameters up-front, with empty interface as type bound.
	// The scope of type parameters starts at the beginning of the type parameter
	// list (so we can have mutually recursive parameterized interfaces).
	for _, f := range list.List {
		tparams = declareTypeParams(ctx, tparams, f.Names)
	}

	ctx.tlookup = &typeParamLookup{tparams}
	defer func() {
		ctx.tlookup = nil
	}()

	index := 0
	for _, f := range list.List {
		var bound types.Type
		// NOTE: we may be able to assert that f.Type != nil here, but this is not
		// an invariant of the AST, so we are cautious.
		if f.Type != nil {
			bound = boundTypeParam(ctx, f.Type)
			if isTypeParam(bound) {
				// We may be able to allow this since it is now well-defined what
				// the underlying type and thus type set of a type parameter is.
				// But we may need some additional form of cycle detection within
				// type parameter lists.
				//check.error(f.Type, MisplacedTypeParam, "cannot use a type parameter as constraint")
				bound = types.Typ[types.Invalid]
			} else if t, ok := bound.(*types.Named); ok {
				if t.Underlying() == nil { // check named underlying is nil
					ctx.loadNamed(ctx.pkg, t)
				}
			}
		} else {
			bound = types.Typ[types.Invalid]
		}
		for i := range f.Names {
			tparams[index+i].SetConstraint(bound)
		}
		index += len(f.Names)
	}
	return tparams
}

func declareTypeParams(ctx *blockCtx, tparams []*types.TypeParam, names []*ast.Ident) []*types.TypeParam {
	// Use Typ[Invalid] for the type constraint to ensure that a type
	// is present even if the actual constraint has not been assigned
	// yet.
	// TODO(gri) Need to systematically review all uses of type parameter
	//           constraints to make sure we don't rely on them if they
	//           are not properly set yet.
	for _, name := range names {
		tname := types.NewTypeName(name.Pos(), ctx.pkg.Types, name.Name, nil)
		tpar := types.NewTypeParam(tname, types.Typ[types.Invalid]) // assigns type to tpar as a side-effect
		// check.declare(check.scope, name, tname, check.scope.pos)    // TODO(gri) check scope position
		tparams = append(tparams, tpar)
	}

	return tparams
}

func isTypeParam(t types.Type) bool {
	_, ok := t.(*types.TypeParam)
	return ok
}

func isSpecificSliceType(ctx *blockCtx, typ types.Type) bool {
	if typ == nil {
		return false
	}
	var t *types.Slice
	switch tt := typ.(type) {
	case *types.Named:
		t = getUnderlying(ctx, tt).(*types.Slice)
	case *types.Slice:
		t = tt
	default:
		return false
	}
	_, ok := t.Elem().(*types.TypeParam)
	return !ok
}

func boundTypeParam(ctx *blockCtx, x ast.Expr) types.Type {
	// A type set literal of the form ~T and A|B may only appear as constraint;
	// embed it in an implicit interface so that only interface type-checking
	// needs to take care of such type expressions.
	wrap := false
	switch op := x.(type) {
	case *ast.UnaryExpr:
		wrap = op.Op == token.TILDE
	case *ast.BinaryExpr:
		wrap = op.Op == token.OR
	}
	if wrap {
		x = &ast.InterfaceType{Methods: &ast.FieldList{List: []*ast.Field{{Type: x}}}}
		t := toType(ctx, x)
		// mark t as implicit interface if all went well
		if t, _ := t.(*types.Interface); t != nil {
			t.MarkImplicit()
		}
		return t
	}
	return toType(ctx, x)
}

func namedIsTypeParams(ctx *blockCtx, t *types.Named) bool {
	o := t.Obj()
	if o.Pkg() == ctx.pkg.Types {
		if _, ok := ctx.generics[o.Name()]; !ok {
			return false
		}
		ctx.loadType(o.Name())
	}
	return t.Obj() != nil && t.TypeArgs() == nil && t.TypeParams() != nil
}
