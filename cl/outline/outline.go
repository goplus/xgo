/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package outline

import (
	"go/types"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/gogen/typeutil"
	"github.com/goplus/mod/modfile"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/token"
)

// -----------------------------------------------------------------------------

type Project = modfile.Project

type Config struct {
	// Fset provides source position information for syntax trees and types.
	// If Fset is nil, Load will use a new fileset, but preserve Fset's value.
	Fset *token.FileSet

	// LookupClass lookups a class by specified file extension.
	LookupClass func(ext string) (c *Project, ok bool)

	// An Importer resolves import paths to Packages.
	Importer types.Importer
}

type Package struct {
	pkg  *types.Package
	docs gogen.ObjectDocs
}

// NewPackage creates a Go/XGo outline package.
func NewPackage(pkgPath string, pkg *ast.Package, conf *Config) (_ Package, err error) {
	ret, err := cl.NewPackage(pkgPath, pkg, &cl.Config{
		Fset:           conf.Fset,
		LookupClass:    conf.LookupClass,
		Importer:       conf.Importer,
		NoFileLine:     true,
		NoAutoGenMain:  true,
		NoSkipConstant: true,
		Outline:        true,
	})
	if err != nil {
		return
	}
	return Package{ret.Types, ret.Docs}, nil
}

func (p Package) Pkg() *types.Package {
	return p.pkg
}

func (p Package) Valid() bool {
	return p.pkg != nil
}

// -----------------------------------------------------------------------------

type All struct {
	Consts []Const
	Vars   []Var
	Funcs  []Func
	Types  []*TypeName

	Package
	named map[*types.TypeName]*TypeName
}

func setAlias(aliasr *typeutil.Map, t types.Type, named *TypeName) {
	real := indirect(t)
	if aliasr.Set(real, named) != nil { // conflict: has old value
		aliasr.Set(real, nil)
	}
}

// aliasr typeutil.Map // types.Type => *TypeName
func (p *All) checkAlias(aliasr *typeutil.Map, t types.Type, withBasic bool) *TypeName {
	if _, ok := t.(*types.Basic); !ok || withBasic {
		if v := aliasr.At(t); v != nil {
			named := v.(*TypeName)
			p.markUsed(named)
			return named
		}
	}
	return nil
}

func (p *All) markUsed(named *TypeName) {
	if !named.isUsed {
		named.isUsed = true
		o := named.TypeName
		typ := o.Type()
		p.checkUsed(typ.Underlying())
		if !o.IsAlias() {
			if t, ok := typ.(*types.Named); ok {
				for i, n := 0, t.NumMethods(); i < n; i++ {
					p.checkUsedMethod(t.Method(i))
				}
			}
		}
	}
}

func (p *All) checkUsed(typ types.Type) {
	switch t := typ.(type) {
	case *types.Basic:
	case *types.Pointer:
		p.checkUsed(t.Elem())
	case *types.Signature:
		p.checkUsedSig(t)
	case *types.Slice:
		p.checkUsed(t.Elem())
	case *types.Map:
		p.checkUsed(t.Key())
		p.checkUsed(t.Elem())
	case *types.Struct:
		for i, n := 0, t.NumFields(); i < n; i++ {
			fld := t.Field(i)
			if fld.Exported() {
				p.checkUsed(fld.Type())
			}
		}
	case *types.Named:
		o := t.Obj()
		if p.pkg == o.Pkg() {
			p.markUsed(p.getNamed(o))
		}
	case *types.Interface:
		for i, n := 0, t.NumExplicitMethods(); i < n; i++ {
			p.checkUsedMethod(t.ExplicitMethod(i))
		}
		for i, n := 0, t.NumEmbeddeds(); i < n; i++ {
			p.checkUsed(t.EmbeddedType(i))
		}
	case *types.Chan:
		p.checkUsed(t.Elem())
	case *types.Array:
		p.checkUsed(t.Elem())
	default:
		panic("checkUsed: unknown type - " + typ.String())
	}
}

func (p *All) checkUsedMethod(fn *types.Func) {
	if fn.Exported() {
		p.checkUsedSig(fn.Type().(*types.Signature))
	}
}

func (p *All) checkUsedSig(sig *types.Signature) {
	p.checkUsedTuple(sig.Params())
	p.checkUsedTuple(sig.Results())
}

func (p *All) checkUsedTuple(t *types.Tuple) {
	for i, n := 0, t.Len(); i < n; i++ {
		p.checkUsed(t.At(i).Type())
	}
}

func (p *All) getNamed(t *types.TypeName) *TypeName {
	if named, ok := p.named[t]; ok {
		return named
	}
	panic("getNamed: type not found - " + t.Name())
}

func (p *All) initNamed(aliasr *typeutil.Map, objs []types.Object) {
	for _, o := range objs {
		if t, ok := o.(*types.TypeName); ok {
			named := &TypeName{TypeName: t}
			p.named[t] = named
			p.Types = append(p.Types, named)
			if t.IsAlias() {
				setAlias(aliasr, t.Type(), named)
			}
		}
	}
}

func (p *All) lookupNamed(pkg *types.Package, name string) (_ *TypeName, ok bool) {
	o := pkg.Scope().Lookup(name)
	if o == nil {
		return
	}
	t, ok := o.(*types.TypeName)
	if !ok {
		return
	}
	return p.getNamed(t), true
}

func (p Package) Outline(withUnexported ...bool) (ret *All) {
	pkg := p.Pkg()
	ret = &All{
		Package: p,
		named:   make(map[*types.TypeName]*TypeName),
	}
	all := (withUnexported != nil && withUnexported[0])
	aliasr := &typeutil.Map{}
	scope := pkg.Scope()
	names := scope.Names()
	objs := make([]types.Object, len(names))
	for i, name := range names {
		objs[i] = scope.Lookup(name)
	}
	ret.initNamed(aliasr, objs)
	for _, o := range objs {
		if !(all || o.Exported()) {
			continue
		}
		if obj, ok := o.(*types.TypeName); ok {
			if !all {
				ret.markUsed(ret.getNamed(obj))
			}
			continue
		}
		switch v := o.(type) {
		case *types.Func:
			sig := v.Type().(*types.Signature)
			if !all {
				ret.checkUsedSig(sig)
			}
			if name, ok := checkGoptFunc(o.Name()); ok {
				if named, ok := ret.lookupNamed(pkg, name); ok {
					named.GoptFuncs = append(named.GoptFuncs, Func{v, p.docs})
					continue
				}
			}
			kind, named := ret.sigKind(aliasr, sig)
			switch kind {
			case sigNormal:
				ret.Funcs = append(ret.Funcs, Func{v, p.docs})
			case sigCreator:
				named.Creators = append(named.Creators, Func{v, p.docs})
			case sigHelper:
				named.Helpers = append(named.Helpers, Func{v, p.docs})
			}
		case *types.Const:
			if name := v.Name(); strings.HasPrefix(name, "Gop") || strings.HasPrefix(name, "XGo") {
				if name == "GopPackage" || name == "XGoPackage" || name == "Gop_sched" {
					continue
				}
			}
			typ := v.Type()
			if !all {
				ret.checkUsed(typ)
			}
			if named := ret.checkLocal(aliasr, typ, true); named != nil {
				named.Consts = append(named.Consts, Const{v})
			} else {
				ret.Consts = append(ret.Consts, Const{v})
			}
		case *types.Var:
			if !all {
				ret.checkUsed(v.Type())
			}
			ret.Vars = append(ret.Vars, Var{v})
		}
	}
	return
}

// -----------------------------------------------------------------------------

type sigKindType int

const (
	sigNormal sigKindType = iota
	sigCreator
	sigHelper
)

func (p *All) sigKind(aliasr *typeutil.Map, sig *types.Signature) (sigKindType, *TypeName) {
	rets := sig.Results()
	if rets.Len() > 0 {
		if t := p.checkLocal(aliasr, rets.At(0).Type(), false); t != nil {
			return sigCreator, t
		}
	}
	params := sig.Params()
	if params.Len() > 0 {
		if t := p.checkLocal(aliasr, params.At(0).Type(), false); t != nil {
			return sigHelper, t
		}
	}
	return sigNormal, nil
}

func (p *All) checkLocal(aliasr *typeutil.Map, first types.Type, withBasic bool) *TypeName {
	first = indirect(first)
	if t, ok := first.(*types.Named); ok {
		o := t.Obj()
		if o.Pkg() == p.pkg {
			return p.getNamed(o)
		}
	}
	return p.checkAlias(aliasr, first, withBasic)
}

func indirect(typ types.Type) types.Type {
	if t, ok := typ.(*types.Pointer); ok {
		return t.Elem()
	}
	return typ
}

// -----------------------------------------------------------------------------

type Const struct {
	*types.Const
}

func (p Const) Obj() types.Object {
	return p.Const
}

func (p Const) Doc() string {
	return ""
}

type Var struct {
	*types.Var
}

func (p Var) Obj() types.Object {
	return p.Var
}

func (p Var) Doc() string {
	return ""
}

type Func struct {
	*types.Func
	docs gogen.ObjectDocs
}

func (p Func) Obj() types.Object {
	return p.Func
}

func (p Func) Doc() string {
	return p.docs[p.Func].Text()
}

func CheckOverload(obj types.Object) (name string, fn *types.Func, ok bool) {
	if fn, ok = obj.(*types.Func); ok {
		name, ok = checkOverloadFunc(fn.Name())
	}
	return
}

const (
	goptPrefix = "Gopt_"
)

func isGoptFunc(name string) bool {
	return strings.HasPrefix(name, goptPrefix)
}

func isOverloadFunc(name string) bool {
	n := len(name)
	return n > 3 && name[n-3:n-1] == "__"
}

func checkGoptFunc(name string) (string, bool) {
	if isGoptFunc(name) {
		name = name[len(goptPrefix):]
		if pos := strings.IndexByte(name, '_'); pos > 0 {
			return name[:pos], true
		}
	}
	return "", false
}

func checkOverloadFunc(name string) (string, bool) {
	if isOverloadFunc(name) {
		return name[:len(name)-3], true
	}
	return "", false
}

// -----------------------------------------------------------------------------

type TypeName struct {
	*types.TypeName
	Consts    []Const
	Creators  []Func
	GoptFuncs []Func
	Helpers   []Func
	isUsed    bool
}

func (p *TypeName) IsUsed() bool {
	return p.isUsed
}

func (p *TypeName) ObjWith(all bool) *types.TypeName {
	o := p.TypeName
	if all {
		return o
	}
	return hideUnexported(o)
}

func (p *TypeName) Obj() types.Object {
	return p.TypeName
}

func (p *TypeName) Doc() string {
	return ""
}

func (p *TypeName) Type() Type {
	return Type{p.TypeName.Type()}
}

func hideUnexported(o *types.TypeName) *types.TypeName {
	if o.IsAlias() {
		if t, ok := typeHideUnexported(o.Type()); ok {
			return types.NewTypeName(o.Pos(), o.Pkg(), o.Name(), t)
		}
	} else if named, ok := o.Type().(*types.Named); ok {
		if t, ok := typeHideUnexported(named.Underlying()); ok {
			name := types.NewTypeName(o.Pos(), o.Pkg(), o.Name(), nil)
			n := named.NumMethods()
			var fns []*types.Func
			if n > 0 {
				fns = make([]*types.Func, n)
				for i := 0; i < n; i++ {
					fns[i] = named.Method(i)
				}
			}
			types.NewNamed(name, t, fns)
			return name
		}
	}
	return o
}

func typeHideUnexported(typ types.Type) (ret types.Type, ok bool) {
	switch t := typ.(type) {
	case *types.Struct:
		n := t.NumFields()
		for i := 0; i < n; i++ {
			fld := t.Field(i)
			if !fld.Exported() || t.Tag(i) != "" {
				ok = true
				break
			}
		}
		if ok {
			flds := make([]*types.Var, 0, n)
			for i := 0; i < n; i++ {
				fld := t.Field(i)
				if fld.Exported() {
					flds = append(flds, fld)
				}
			}
			if len(flds) < n {
				flds = append(flds, types.NewField(token.NoPos, nil, "", tyUnexp, true))
			}
			ret = types.NewStruct(flds, nil)
		}
	}
	return
}

type tyUnexpImp struct{}

func (p tyUnexpImp) String() string         { return "..." }
func (p tyUnexpImp) Underlying() types.Type { return p }

var (
	tyUnexp types.Type = tyUnexpImp{}
)

// -----------------------------------------------------------------------------

type Type struct {
	types.Type
}

func (p Type) CheckNamed(pkg Package) (_ Named, ok bool) {
	ret, ok := p.Type.(*types.Named)
	if ok && ret.Obj().Pkg() == pkg.pkg {
		return Named{ret, pkg.docs}, true
	}
	return
}

// -----------------------------------------------------------------------------

type Named struct {
	*types.Named
	docs gogen.ObjectDocs
}

func (p Named) Methods() []Func {
	n := p.NumMethods()
	ret := make([]Func, n)
	for i := 0; i < n; i++ {
		fn := p.Method(i)
		ret[i] = Func{fn, p.docs}
	}
	return ret
}

// -----------------------------------------------------------------------------
