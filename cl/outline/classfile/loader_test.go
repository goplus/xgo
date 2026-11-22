/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package classfile

import (
	"go/types"
	"testing"

	"github.com/goplus/xgo/token"
)

func TestResourceSchemaLoaderAddRawAPIScopeBindings(t *testing.T) {
	pkg := types.NewPackage("example.com/spx", "spx")
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	fn := types.NewFunc(0, pkg, "SetCostume", sig)
	l := resourceSchemaLoader{schema: &ResourceSchema{}, fset: token.NewFileSet()}

	l.addRawAPIScopeBindings(fn, []apiScopeBindingDirective{
		{target: 0, source: ResourceAPIScopeSource{Receiver: true}},
		{target: 0, source: ResourceAPIScopeSource{Param: 1}},
	})

	if got := l.rawAPIScopeBindings[fn]; len(got) != 0 {
		t.Fatalf("unexpected raw bindings: %#v", got)
	}
	if len(l.errs) != 2 {
		t.Fatalf("unexpected error count: %d", len(l.errs))
	}
}

func TestResourceSchemaLoaderValidateKinds(t *testing.T) {
	fset := token.NewFileSet()
	file := fset.AddFile("spx.go", fset.Base(), 64)
	kindPos := file.Pos(12)
	l := resourceSchemaLoader{
		fset: fset,
		schema: &ResourceSchema{
			Kinds: []*ResourceKind{{Name: "sprite.costume", pos: kindPos}},
			byKind: map[string]*ResourceKind{
				"sprite.costume": {Name: "sprite.costume", pos: kindPos},
			},
		},
	}

	l.validateKinds()

	if len(l.errs) != 1 {
		t.Fatalf("unexpected error count: %d", len(l.errs))
	}
	if l.errs[0].Pos.Filename != "spx.go" || l.errs[0].Pos.Offset != 12 {
		t.Fatalf("unexpected error position: %#v", l.errs[0].Pos)
	}
}

func TestResourceSchemaLoaderKindOfTarget(t *testing.T) {
	l, _, costumeKind, spriteName, costumeName, _ := testResourceSchemaLoaderKinds()

	t.Run("ValidScopedCanonical", func(t *testing.T) {
		sig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(types.NewVar(0, nil, "costume", costumeName)),
			types.NewTuple(),
			false,
		)

		got, ok := l.kindOfTarget(sig, rawAPIScopeBinding{target: 0})
		if !ok || got != costumeKind {
			t.Fatalf("unexpected target kind: %#v, %v", got, ok)
		}
	})

	t.Run("TopLevelCanonical", func(t *testing.T) {
		sig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(types.NewVar(0, nil, "sprite", spriteName)),
			types.NewTuple(),
			false,
		)

		got, ok := l.kindOfTarget(sig, rawAPIScopeBinding{target: 0})
		if ok || got != nil {
			t.Fatalf("unexpected target kind: %#v, %v", got, ok)
		}
	})

	t.Run("VariadicLastParam", func(t *testing.T) {
		sig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(
				types.NewVar(0, nil, "costume", costumeName),
				types.NewVar(0, nil, "rest", types.NewSlice(types.Typ[types.String])),
			),
			types.NewTuple(),
			true,
		)

		got, ok := l.kindOfTarget(sig, rawAPIScopeBinding{target: 1})
		if ok || got != nil {
			t.Fatalf("unexpected variadic target kind: %#v, %v", got, ok)
		}
	})
}

func TestResourceSchemaLoaderValidSource(t *testing.T) {
	l, _, costumeKind, spriteName, costumeName, spriteImpl := testResourceSchemaLoaderKinds()

	t.Run("ReceiverHandleBearing", func(t *testing.T) {
		sig := types.NewSignatureType(
			types.NewVar(0, nil, "recv", types.NewPointer(spriteImpl)),
			nil,
			nil,
			types.NewTuple(types.NewVar(0, nil, "costume", costumeName)),
			types.NewTuple(),
			false,
		)
		fn := types.NewFunc(0, nil, "SetCostume", sig)

		if !l.validSource(fn, sig, costumeKind, rawAPIScopeBinding{target: 0, source: ResourceAPIScopeSource{Receiver: true}}) {
			t.Fatal("expected valid receiver source")
		}
	})

	t.Run("CanonicalParam", func(t *testing.T) {
		sig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(
				types.NewVar(0, nil, "sprite", spriteName),
				types.NewVar(0, nil, "costume", costumeName),
			),
			types.NewTuple(),
			false,
		)
		fn := types.NewFunc(0, nil, "SetCostume", sig)

		if !l.validSource(fn, sig, costumeKind, rawAPIScopeBinding{target: 1, source: ResourceAPIScopeSource{Param: 0}}) {
			t.Fatal("expected valid canonical param source")
		}
	})

	t.Run("WrongKind", func(t *testing.T) {
		sig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(
				types.NewVar(0, nil, "costume", costumeName),
				types.NewVar(0, nil, "sprite", spriteName),
			),
			types.NewTuple(),
			false,
		)
		fn := types.NewFunc(0, nil, "SetCostume", sig)

		if l.validSource(fn, sig, costumeKind, rawAPIScopeBinding{target: 1, source: ResourceAPIScopeSource{Param: 0}}) {
			t.Fatal("unexpected valid source")
		}
	})

	t.Run("InterfaceReceiverHandleBearing", func(t *testing.T) {
		pkg := types.NewPackage("example.com/spx", "spx")
		sig := types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(types.NewVar(0, nil, "costume", costumeName)),
			types.NewTuple(),
			false,
		)
		fn := types.NewFunc(0, pkg, "SetCostume", sig)
		iface := types.NewInterfaceType([]*types.Func{fn}, nil)
		iface.Complete()
		spriteObj := types.NewTypeName(0, pkg, "Sprite", nil)
		types.NewNamed(spriteObj, iface, nil)
		interfaceMethod := iface.ExplicitMethod(0)
		l.schema.byHandle[spriteObj] = l.schema.byKind["sprite"]
		l.callableHandles = map[*types.Func]*types.TypeName{interfaceMethod: spriteObj}

		if !l.validSource(interfaceMethod, interfaceMethod.Type().(*types.Signature), costumeKind, rawAPIScopeBinding{target: 0, source: ResourceAPIScopeSource{Receiver: true}}) {
			t.Fatal("expected valid interface receiver source")
		}
	})
}

func TestIsStringBased(t *testing.T) {
	t.Run("StringAlias", func(t *testing.T) {
		obj := types.NewTypeName(0, nil, "SpriteName", types.Typ[types.String])
		if !isStringBased(obj) {
			t.Fatal("expected exported string-based type")
		}
	})

	t.Run("Unexported", func(t *testing.T) {
		obj := types.NewTypeName(0, nil, "spriteName", types.Typ[types.String])
		if isStringBased(obj) {
			t.Fatal("unexpected unexported string-based type")
		}
	})

	t.Run("NonString", func(t *testing.T) {
		obj := types.NewTypeName(0, nil, "SpriteID", types.Typ[types.Int])
		if isStringBased(obj) {
			t.Fatal("unexpected non-string type")
		}
	})
}

func TestIsHandleBearing(t *testing.T) {
	t.Run("Struct", func(t *testing.T) {
		obj := types.NewTypeName(0, nil, "SpriteImpl", nil)
		named := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
		if !isHandleBearing(obj) {
			t.Fatal("expected exported struct handle-bearing type")
		}
		if named.Obj() != obj {
			t.Fatal("expected named type object identity")
		}
	})

	t.Run("Interface", func(t *testing.T) {
		obj := types.NewTypeName(0, nil, "Sprite", nil)
		empty := types.NewInterfaceType(nil, nil)
		empty.Complete()
		types.NewNamed(obj, empty, nil)
		if !isHandleBearing(obj) {
			t.Fatal("expected exported interface handle-bearing type")
		}
	})

	t.Run("Alias", func(t *testing.T) {
		obj := types.NewTypeName(0, nil, "SpriteAlias", nil)
		alias := types.NewAlias(obj, types.Typ[types.String])
		if isHandleBearing(alias.Obj()) {
			t.Fatal("unexpected alias handle-bearing type")
		}
	})
}

func TestHasAPIScopeBindingCycle(t *testing.T) {
	t.Run("Cyclic", func(t *testing.T) {
		if !hasAPIScopeBindingCycle([]rawAPIScopeBinding{
			{target: 0, source: ResourceAPIScopeSource{Param: 1}},
			{target: 1, source: ResourceAPIScopeSource{Param: 0}},
		}) {
			t.Fatal("expected cycle")
		}
	})

	t.Run("Acyclic", func(t *testing.T) {
		if hasAPIScopeBindingCycle([]rawAPIScopeBinding{
			{target: 0, source: ResourceAPIScopeSource{Receiver: true}},
			{target: 1, source: ResourceAPIScopeSource{Param: 0}},
		}) {
			t.Fatal("unexpected cycle")
		}
	})
}

func testResourceSchemaLoaderKinds() (resourceSchemaLoader, *ResourceKind, *ResourceKind, *types.Alias, *types.Alias, *types.Named) {
	pkg := types.NewPackage("example.com/spx", "spx")

	spriteNameObj := types.NewTypeName(0, pkg, "SpriteName", nil)
	spriteName := types.NewAlias(spriteNameObj, types.Typ[types.String])

	costumeNameObj := types.NewTypeName(0, pkg, "SpriteCostumeName", nil)
	costumeName := types.NewAlias(costumeNameObj, types.Typ[types.String])

	spriteImplObj := types.NewTypeName(0, pkg, "SpriteImpl", nil)
	spriteImpl := types.NewNamed(spriteImplObj, types.NewStruct(nil, nil), nil)

	spriteKind := &ResourceKind{
		Name:          "sprite",
		CanonicalType: spriteNameObj,
		HandleTypes:   []*types.TypeName{spriteImplObj},
	}
	costumeKind := &ResourceKind{
		Name:          "sprite.costume",
		CanonicalType: costumeNameObj,
	}

	schema := &ResourceSchema{
		Package: pkg,
		Kinds:   []*ResourceKind{spriteKind, costumeKind},
		byKind: map[string]*ResourceKind{
			"sprite":         spriteKind,
			"sprite.costume": costumeKind,
		},
		byCanonical: map[*types.TypeName]*ResourceKind{
			spriteNameObj:  spriteKind,
			costumeNameObj: costumeKind,
		},
		byHandle: map[*types.TypeName]*ResourceKind{
			spriteImplObj: spriteKind,
		},
	}

	return resourceSchemaLoader{fset: token.NewFileSet(), schema: schema}, spriteKind, costumeKind, spriteName, costumeName, spriteImpl
}
