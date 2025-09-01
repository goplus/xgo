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
	"go/constant"
	"go/types"
	"log"
	"math/big"
	"strconv"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

// -----------------------------------------------------------------------------

func toRecv(ctx *blockCtx, recv *ast.FieldList) *types.Var {
	v := recv.List[0]
	var name string
	if len(v.Names) > 0 {
		name = v.Names[0].Name
	}
	typ, star, _ := getRecvType(v.Type)
	id, ok := typ.(*ast.Ident)
	if !ok {
		panic("TODO: getRecvType")
	}
	t := toIdentType(ctx, id)
	if star {
		t = types.NewPointer(t)
	}
	ret := ctx.pkg.NewParam(v.Pos(), name, t)
	if rec := ctx.recorder(); rec != nil {
		dRecv := recv.List[0]
		if names := dRecv.Names; len(names) == 1 {
			rec.Def(names[0], ret)
		}
	}
	return ret
}

func getRecvTypeName(ctx *pkgCtx, recv *ast.FieldList, handleErr bool) (string, bool) {
	typ, _, _ := getRecvType(recv.List[0].Type)
	if t, ok := typ.(*ast.Ident); ok {
		return t.Name, true
	}
	if handleErr {
		src := ctx.LoadExpr(typ)
		ctx.handleErrorf(typ.Pos(), typ.End(), "invalid receiver type %v (%v is not a defined type)", src, src)
	}
	return "", false
}

func toResults(ctx *blockCtx, in *ast.FieldList) *types.Tuple {
	if in == nil {
		return nil
	}
	flds := in.List
	n := len(flds)
	args := make([]*types.Var, 0, n)
	for _, fld := range flds {
		args = toParam(ctx, fld, args)
	}
	return types.NewTuple(args...)
}

func toParams(ctx *blockCtx, flds []*ast.Field) (typ *types.Tuple, variadic bool) {
	n := len(flds)
	if n == 0 {
		return nil, false
	}
	args := make([]*types.Var, 0, n)
	for _, fld := range flds {
		args = toParam(ctx, fld, args)
	}
	_, ok := flds[n-1].Type.(*ast.Ellipsis)
	return types.NewTuple(args...), ok
}

func toParam(ctx *blockCtx, fld *ast.Field, args []*gogen.Param) []*gogen.Param {
	typ := toType(ctx, fld.Type)
	pkg := ctx.pkg
	if len(fld.Names) == 0 {
		return append(args, pkg.NewParam(fld.Pos(), "", typ))
	}
	for _, name := range fld.Names {
		param := pkg.NewParam(name.Pos(), name.Name, typ)
		args = append(args, param)
		if rec := ctx.recorder(); rec != nil {
			rec.Def(name, param)
		}
	}
	return args
}

// -----------------------------------------------------------------------------

func toType(ctx *blockCtx, typ ast.Expr) (t types.Type) {
	if rec := ctx.recorder(); rec != nil {
		defer func() {
			rec.recordType(typ, t)
		}()
	}
	switch v := typ.(type) {
	case *ast.Ident:
		ctx.idents = append(ctx.idents, v)
		defer func() {
			ctx.idents = ctx.idents[:len(ctx.idents)-1]
		}()
		typ := toIdentType(ctx, v)
		if ctx.inInst == 0 {
			if t, ok := typ.(*types.Named); ok {
				if namedIsTypeParams(ctx, t) {
					pos := ctx.idents[0].Pos()
					end := ctx.idents[0].End()
					for _, i := range ctx.idents {
						if i.Name == v.Name {
							pos = i.Pos()
							end = i.End()
							break
						}
					}
					ctx.handleErrorf(pos, end, "cannot use generic type %v without instantiation", t.Obj().Type())
					return types.Typ[types.Invalid]
				}
			}
		}
		return typ
	case *ast.StarExpr:
		elem := toType(ctx, v.X)
		return types.NewPointer(elem)
	case *ast.ArrayType:
		return toArrayType(ctx, v)
	case *ast.InterfaceType:
		return toInterfaceType(ctx, v)
	case *ast.Ellipsis:
		elem := toType(ctx, v.Elt)
		return types.NewSlice(elem)
	case *ast.MapType:
		return toMapType(ctx, v)
	case *ast.StructType:
		return toStructType(ctx, v)
	case *ast.ChanType:
		return toChanType(ctx, v)
	case *ast.FuncType:
		return toFuncType(ctx, v, nil, nil)
	case *ast.SelectorExpr:
		typ := toExternalType(ctx, v)
		if ctx.inInst == 0 {
			if t, ok := typ.(*types.Named); ok {
				if namedIsTypeParams(ctx, t) {
					panic(ctx.newCodeErrorf(v.Pos(), v.End(), "cannot use generic type %v without instantiation", t.Obj().Type()))
				}
			}
		}
		return typ
	case *ast.ParenExpr:
		return toType(ctx, v.X)
	case *ast.BinaryExpr:
		return toBinaryExprType(ctx, v)
	case *ast.UnaryExpr:
		return toUnaryExprType(ctx, v)
	case *ast.IndexExpr:
		return toIndexType(ctx, v)
	case *ast.IndexListExpr:
		return toIndexListType(ctx, v)
	default:
		ctx.handleErrorf(v.Pos(), v.End(), "toType unexpected: %T", v)
		return types.Typ[types.Invalid]
	}
}

var (
	typesChanDirs = [...]types.ChanDir{
		ast.RECV:            types.RecvOnly,
		ast.SEND:            types.SendOnly,
		ast.SEND | ast.RECV: types.SendRecv,
	}
)

func toChanType(ctx *blockCtx, v *ast.ChanType) *types.Chan {
	return types.NewChan(typesChanDirs[v.Dir], toType(ctx, v.Value))
}

func toExternalType(ctx *blockCtx, v *ast.SelectorExpr) types.Type {
	id := v.X.(*ast.Ident)
	name := id.Name
	if pi, ok := ctx.findImport(name); ok {
		rec := ctx.recorder()
		if rec != nil {
			rec.Use(id, pi.pkgName)
		}
		o := pi.TryRef(v.Sel.Name)
		if t, ok := o.(*types.TypeName); ok {
			if rec != nil {
				rec.Use(v.Sel, t)
			}
			return t.Type()
		}
		ctx.handleErrorf(v.Pos(), v.End(), "%s.%s is not a type", name, v.Sel.Name)
	} else {
		ctx.handleErrorf(v.Pos(), v.End(), "undefined: %s", name)
	}
	return types.Typ[types.Invalid]
}

/*-----------------------------------------------------------------------------

Name context:
- type
- pkgRef.type
- spx.type

// ---------------------------------------------------------------------------*/

func toIdentType(ctx *blockCtx, ident *ast.Ident) (ret types.Type) {
	var obj types.Object
	if rec := ctx.recorder(); rec != nil {
		defer func() {
			if obj != nil {
				rec.recordIdent(ident, obj)
			}
		}()
	}
	if ctx.tlookup != nil {
		if typ := ctx.tlookup.Lookup(ident.Name); typ != nil {
			obj = typ.Obj()
			return typ
		}
	}
	v, builtin := lookupType(ctx, ident.Name)
	if isBuiltin(builtin) {
		ctx.handleErrorf(ident.Pos(), ident.End(), "use of builtin %s not in function call", ident.Name)
		return types.Typ[types.Invalid]
	}
	if t, ok := v.(*types.TypeName); ok {
		obj = t
		return t.Type()
	}
	if v, _ := lookupPkgRef(ctx, gogen.PkgRef{}, ident, objPkgRef); v != nil {
		if t, ok := v.(*types.TypeName); ok {
			obj = t
			return t.Type()
		}
	}
	ctx.handleErrorf(ident.Pos(), ident.End(), "%s is not a type", ident.Name)
	return types.Typ[types.Invalid]
}

// TODO: optimization
func lookupType(ctx *blockCtx, name string) (types.Object, types.Object) {
	at, o := ctx.cb.Scope().LookupParent(name, token.NoPos)
	if o != nil && at != types.Universe {
		if debugLookup {
			log.Println("==> LookupParent", name, "=>", o)
		}
		return o, nil
	}
	if ctx.loadSymbol(name) {
		if v := ctx.pkg.Types.Scope().Lookup(name); v != nil {
			if debugLookup {
				log.Println("==> Lookup (LoadSymbol)", name, "=>", v)
			}
			return v, nil
		}
	}
	if obj := ctx.pkg.Builtin().TryRef(name); obj != nil {
		return obj, o
	}
	return o, o
}

type FieldKind int

const (
	FieldKindUser FieldKind = iota
	FieldKindClass
)

type fieldElem struct {
	pos  token.Pos
	end  token.Pos
	kind FieldKind
}

type checkRedecl struct {
	names map[string]fieldElem
}

func newCheckRedecl() *checkRedecl {
	return &checkRedecl{names: make(map[string]fieldElem)}
}

func (p *checkRedecl) chkRedecl(ctx *blockCtx, name string, pos, end token.Pos, kind FieldKind) bool {
	if name == "_" {
		return false
	}

	if existing, ok := p.names[name]; ok {
		switch existing.kind {
		case FieldKindClass:
			ctx.handleErrorf(
				pos, end, "field %s conflicts with class name. rename the field to resolve the naming conflict.",
				name)
		case FieldKindUser:
			ctx.handleErrorf(
				pos, end, "field %s conflicts with field %s defined at %v. rename one of the fields to resolve the naming conflict.",
				name, name, ctx.Position(existing.pos))
		}
		return true
	}

	p.names[name] = fieldElem{
		pos:  pos,
		end:  end,
		kind: kind,
	}
	return false
}

func toStructType(ctx *blockCtx, v *ast.StructType) *types.Struct {
	pkg := ctx.pkg.Types
	fieldList := v.Fields.List
	fields := make([]*types.Var, 0, len(fieldList))
	tags := make([]string, 0, len(fieldList))
	chk := newCheckRedecl()
	rec := ctx.recorder()
	for _, field := range fieldList {
		typ := toType(ctx, field.Type)
		if len(field.Names) == 0 { // embedded
			name := getTypeName(typ)
			if chk.chkRedecl(ctx, name, field.Type.Pos(), field.Type.End(), FieldKindUser) {
				continue
			}
			if t, ok := typ.(*types.Named); ok { // #1196: embedded type should ensure loaded
				ctx.loadNamed(ctx.pkg, t)
			}
			ident := parseTypeEmbedName(field.Type)
			fld := types.NewField(ident.NamePos, pkg, name, typ, true)
			fields = append(fields, fld)
			tags = append(tags, toFieldTag(field.Tag))
			if rec != nil {
				rec.Def(ident, fld)
			}
			continue
		}
		for _, name := range field.Names {
			if chk.chkRedecl(ctx, name.Name, name.Pos(), name.End(), FieldKindUser) {
				continue
			}
			fld := types.NewField(name.NamePos, pkg, name.Name, typ, false)
			fields = append(fields, fld)
			tags = append(tags, toFieldTag(field.Tag))
			if rec != nil {
				rec.Def(name, fld)
			}
		}
	}
	return types.NewStruct(fields, tags)
}

func toFieldTag(v *ast.BasicLit) string {
	if v != nil {
		tag, err := strconv.Unquote(v.Value)
		if err != nil {
			log.Panicln("TODO: toFieldTag -", err)
		}
		return tag
	}
	return ""
}

func getTypeName(typ types.Type) string {
	if t, ok := typ.(*types.Pointer); ok {
		typ = t.Elem()
	}
	switch t := typ.(type) {
	case *types.Named:
		return t.Obj().Name()
	case *types.Basic:
		return t.Name()
	default:
		panic("TODO: getTypeName")
	}
}

func toMapType(ctx *blockCtx, v *ast.MapType) *types.Map {
	key := toType(ctx, v.Key)
	val := toType(ctx, v.Value)
	return types.NewMap(key, val)
}

func toArrayType(ctx *blockCtx, v *ast.ArrayType) types.Type {
	elem := toType(ctx, v.Elt)
	if v.Len == nil {
		return types.NewSlice(elem)
	}
	if _, ok := v.Len.(*ast.Ellipsis); ok {
		return types.NewArray(elem, -1) // A negative length indicates an unknown length
	}
	return types.NewArray(elem, toInt64(ctx, v.Len, "non-constant array bound %s"))
}

func toInt64(ctx *blockCtx, e ast.Expr, emsg string) int64 {
	cb := ctx.pkg.ConstStart()
	compileExpr(ctx, e)
	tv := cb.EndConst()
	if val := tv.CVal; val != nil {
		if val.Kind() == constant.Float {
			if v, ok := constant.Val(val).(*big.Rat); ok && v.IsInt() {
				return v.Num().Int64()
			}
		} else if v, ok := constant.Int64Val(val); ok {
			return v
		}
	}
	src := ctx.LoadExpr(e)
	panic(ctx.newCodeErrorf(e.Pos(), e.End(), emsg, src))
}

func toInterfaceType(ctx *blockCtx, v *ast.InterfaceType) types.Type {
	methodsList := v.Methods.List
	if methodsList == nil {
		return types.NewInterfaceType(nil, nil)
	}
	var rec = ctx.recorder()
	var pkg = ctx.pkg.Types
	var methods []*types.Func
	var embeddeds []types.Type
	for _, m := range methodsList {
		if len(m.Names) == 0 { // embedded
			typ := toType(ctx, m.Type)
			if t, ok := typ.(*types.Named); ok { // #1198: embedded type should ensure loaded
				ctx.loadNamed(ctx.pkg, t)
			}
			embeddeds = append(embeddeds, typ)
			continue
		}
		name := m.Names[0]
		sig := toFuncType(ctx, m.Type.(*ast.FuncType), nil, nil)
		mthd := types.NewFunc(name.NamePos, pkg, name.Name, sig)
		methods = append(methods, mthd)
		if rec != nil {
			rec.Def(name, mthd)
		}
	}
	intf := types.NewInterfaceType(methods, embeddeds).Complete()
	return intf
}

func instantiate(ctx *blockCtx, exprX ast.Expr, indices ...ast.Expr) types.Type {
	ctx.inInst++
	defer func() {
		ctx.inInst--
	}()

	x := toType(ctx, exprX)
	idx := make([]types.Type, len(indices))
	for i, index := range indices {
		idx[i] = toType(ctx, index)
	}
	typ := ctx.pkg.Instantiate(x, idx, exprX)
	if rec := ctx.recorder(); rec != nil {
		rec.instantiate(exprX, x, typ)
	}
	return typ
}

func toIndexType(ctx *blockCtx, v *ast.IndexExpr) types.Type {
	return instantiate(ctx, v.X, v.Index)
}

func toIndexListType(ctx *blockCtx, v *ast.IndexListExpr) types.Type {
	return instantiate(ctx, v.X, v.Indices...)
}

// -----------------------------------------------------------------------------

func toString(l *ast.BasicLit) string {
	if l.Kind == token.STRING {
		s, err := strconv.Unquote(l.Value)
		if err == nil {
			return s
		}
	}
	panic("TODO: toString - convert ast.BasicLit to string failed")
}

// -----------------------------------------------------------------------------
