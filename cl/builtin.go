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
	"go/token"
	"go/types"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/cl/internal/typesalias"
)

// -----------------------------------------------------------------------------

func initMathBig(_ *gogen.Package, conf *gogen.Config, big gogen.PkgRef) {
	conf.UntypedBigInt = big.Ref("UntypedBigint").Type().(*types.Named)
	conf.UntypedBigRat = big.Ref("UntypedBigrat").Type().(*types.Named)
	conf.UntypedBigFloat = big.Ref("UntypedBigfloat").Type().(*types.Named)
}

type Builtin struct {
	types.Object
	name string
	pkg  string
	sym  string
	fn   bool
}

func (t *Builtin) Parent() *types.Scope {
	return Universe
}
func (t *Builtin) Pos() token.Pos {
	return token.NoPos
}
func (t *Builtin) Pkg() *types.Package {
	return nil
}
func (t *Builtin) Name() string {
	return t.name
}
func (t *Builtin) Type() types.Type {
	return types.Typ[types.Invalid]
}
func (t *Builtin) Exported() bool {
	return false
}
func (t *Builtin) Id() string {
	return "_." + t.name
}
func (t *Builtin) String() string {
	return "builtin " + t.name
}
func (t *Builtin) Sym() string {
	return t.pkg + "." + t.sym
}
func (t *Builtin) IsFunc() bool {
	return t.fn
}

var (
	Universe *types.Scope
)

var builtinTypes = [...]struct {
	name string
	pkg  string
	sym  string
}{
	{"bigint", "github.com/qiniu/x/xgo/ng", ""},
	{"bigrat", "github.com/qiniu/x/xgo/ng", ""},
	{"bigfloat", "github.com/qiniu/x/xgo/ng", ""},
	{"int128", "github.com/qiniu/x/xgo/ng", ""},
	{"uint128", "github.com/qiniu/x/xgo/ng", ""},
}

var builtinFuncs = [...]struct {
	name string
	pkg  string
	sym  string
}{
	{"errorln", "github.com/qiniu/x/osx", ""},
	{"fatal", "github.com/qiniu/x/osx", ""},
	{"lines", "github.com/qiniu/x/osx", ""},
	{"blines", "github.com/qiniu/x/osx", "BLines"},
	{"newRange", "github.com/qiniu/x/xgo", "NewRange__0"},
	{"echo", "fmt", "Println"},
	{"print", "fmt", ""},
	{"println", "fmt", ""},
	{"printf", "fmt", ""},
	{"errorf", "fmt", ""},
	{"fprint", "fmt", ""},
	{"fprintln", "fmt", ""},
	{"sprint", "fmt", ""},
	{"sprintln", "fmt", ""},
	{"sprintf", "fmt", ""},
	{"open", "os", ""},
	{"create", "os", ""},
	{"type", "reflect", "TypeOf"},
}

type defSym struct {
	name string
	sym  string
	fn   bool
}

var (
	builtinSym map[string][]defSym
)

func insertBuiltin(name, pkg, sym string, fn bool) {
	if sym == "" {
		sym = string(name[0]-('a'-'A')) + name[1:]
	}
	builtinSym[pkg] = append(builtinSym[pkg], defSym{name: name, sym: sym, fn: fn})
	obj := &Builtin{name: name, pkg: pkg, sym: sym, fn: fn}
	Universe.Insert(obj)
}

func init() {
	Universe = types.NewScope(nil, 0, 0, "universe")
	builtinSym = make(map[string][]defSym)
	for _, def := range builtinTypes {
		insertBuiltin(def.name, def.pkg, def.sym, false)
	}
	for _, def := range builtinFuncs {
		insertBuiltin(def.name, def.pkg, def.sym, true)
	}
}

func initBuiltin(pkg *gogen.Package, builtin *types.Package, conf *gogen.Config) {
	scope := builtin.Scope()
	for im, defs := range builtinSym {
		if p := pkg.TryImport(im); p.Types != nil {
			for _, def := range defs {
				obj := p.Ref(def.sym)
				if def.fn {
					scope.Insert(gogen.NewOverloadFunc(token.NoPos, builtin, def.name, obj))
				} else {
					scope.Insert(types.NewTypeName(token.NoPos, builtin, def.name, obj.Type()))
				}
				if rec, ok := conf.Recorder.(*goxRecorder); ok {
					rec.Builtin(def.name, obj)
				}
			}
		}
	}
	scope.Insert(types.NewTypeName(token.NoPos, builtin, "any", gogen.TyEmptyInterface))
}

const (
	osxPkgPath = "github.com/qiniu/x/osx"
)

func getXGo_ninteger(ng gogen.PkgRef) types.Object {
	scope := ng.Types.Scope()
	obj := scope.Lookup("XGo_ninteger")
	if obj == nil {
		obj = scope.Lookup("Gop_integer") // for backward compatibility
	}
	return obj
}

func (ctx *pkgCtx) newBuiltinDefault(pkg *gogen.Package, conf *gogen.Config) *types.Package {
	builtin := types.NewPackage("", "")
	ng := pkg.TryImport("github.com/qiniu/x/xgo/ng")
	strx := pkg.TryImport("github.com/qiniu/x/stringutil")
	stringslice := pkg.TryImport("github.com/qiniu/x/stringslice")
	pkg.TryImport("strconv")
	pkg.TryImport("strings")
	if ng.Types != nil {
		initMathBig(pkg, conf, ng)
		if typesalias.Support {
			if obj := getXGo_ninteger(ng); obj != nil {
				if _, ok := obj.Type().(*types.Basic); !ok {
					conf.EnableTypesalias = true
					ctx.featTypesAlias = true
				}
			}
		}
	}
	initBuiltin(pkg, builtin, conf)
	gogen.InitBuiltin(pkg, builtin, conf)
	if strx.Types != nil {
		ti := pkg.BuiltinTI(types.Typ[types.String])
		ti.AddMethods(
			&gogen.BuiltinMethod{Name: "Capitalize", Fn: strx.Ref("Capitalize")},
		)
	}
	if stringslice.Types != nil {
		ti := pkg.BuiltinTI(types.NewSlice(types.Typ[types.String]))
		ti.AddMethods(
			&gogen.BuiltinMethod{Name: "Capitalize", Fn: stringslice.Ref("Capitalize")},
			&gogen.BuiltinMethod{Name: "ToTitle", Fn: stringslice.Ref("ToTitle")},
			&gogen.BuiltinMethod{Name: "ToUpper", Fn: stringslice.Ref("ToUpper")},
			&gogen.BuiltinMethod{Name: "ToLower", Fn: stringslice.Ref("ToLower")},
			&gogen.BuiltinMethod{Name: "Repeat", Fn: stringslice.Ref("Repeat")},
			&gogen.BuiltinMethod{Name: "Replace", Fn: stringslice.Ref("Replace")},
			&gogen.BuiltinMethod{Name: "ReplaceAll", Fn: stringslice.Ref("ReplaceAll")},
			&gogen.BuiltinMethod{Name: "Trim", Fn: stringslice.Ref("Trim")},
			&gogen.BuiltinMethod{Name: "TrimSpace", Fn: stringslice.Ref("TrimSpace")},
			&gogen.BuiltinMethod{Name: "TrimLeft", Fn: stringslice.Ref("TrimLeft")},
			&gogen.BuiltinMethod{Name: "TrimRight", Fn: stringslice.Ref("TrimRight")},
			&gogen.BuiltinMethod{Name: "TrimPrefix", Fn: stringslice.Ref("TrimPrefix")},
			&gogen.BuiltinMethod{Name: "TrimSuffix", Fn: stringslice.Ref("TrimSuffix")},
		)
	}
	return builtin
}

// -----------------------------------------------------------------------------
