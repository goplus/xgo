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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/mod/modfile"
	xast "github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl/outline"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/scanner"
	"github.com/goplus/xgo/token"
)

func TestLoadResourceSchema(t *testing.T) {
	t.Run("ValidGoPackage", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

// SpriteName identifies a sprite by name.
//
//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

type SpriteAlias = SpriteName

// SpriteCostumeName identifies a sprite costume by name.
//
//xgo:class:resource sprite.costume
//xgo:class:resource-discovery costumes.*
type SpriteCostumeName = string

// SpriteCostumeFrameName identifies one sprite costume frame by name.
//
//xgo:class:resource sprite.costume.frame
//xgo:class:resource-discovery frames.*
type SpriteCostumeFrameName = string

// WidgetName identifies a widget by name.
//
//xgo:class:resource widget
//xgo:class:resource-discovery widgets.*
//xgo:class:resource-name-discovery id
type WidgetName = string

// SpriteImpl is a handle-bearing type of the sprite resource kind.
//
//xgo:class:resource sprite
type SpriteImpl struct{}

// SetCostume__0 sets the current costume.
//
//xgo:class:resource-api-scope-binding param.0 receiver
func (s *SpriteImpl) SetCostume__0(costume SpriteCostumeName) {}

// SetCostumeAndFrame sets the current costume and frame.
//
//xgo:class:resource-api-scope-binding param.0 receiver
//xgo:class:resource-api-scope-binding param.1 param.0
func (s *SpriteImpl) SetCostumeAndFrame(costume SpriteCostumeName, frame SpriteCostumeFrameName) {}
`,
		})

		schema, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err != nil {
			t.Fatalf("LoadResourceSchema failed: %v", err)
		}

		sprite, ok := schema.Kind("sprite")
		if !ok {
			t.Fatal("sprite kind not found")
		}
		if sprite.CanonicalType == nil || sprite.CanonicalType.Name() != "SpriteName" {
			t.Fatalf("unexpected sprite canonical type: %#v", sprite.CanonicalType)
		}
		if sprite.DiscoveryQuery != "sprites.*" {
			t.Fatalf("unexpected sprite discovery query: %q", sprite.DiscoveryQuery)
		}
		if len(sprite.HandleTypes) != 1 || sprite.HandleTypes[0].Name() != "SpriteImpl" {
			t.Fatalf("unexpected sprite handle types: %#v", sprite.HandleTypes)
		}

		widget, ok := schema.Kind("widget")
		if !ok {
			t.Fatal("widget kind not found")
		}
		if widget.NameDiscoveryQuery != "id" {
			t.Fatalf("unexpected widget name-discovery query: %q", widget.NameDiscoveryQuery)
		}

		if got, ok := schema.KindOfCanonical(sprite.CanonicalType); !ok || got != sprite {
			t.Fatal("KindOfCanonical did not resolve sprite canonical type")
		}
		spriteAlias, ok := schema.Package.Scope().Lookup("SpriteAlias").(*types.TypeName)
		if !ok {
			t.Fatal("SpriteAlias type not found")
		}
		if got, ok := schema.CanonicalKindOfType(spriteAlias.Type()); !ok || got != sprite {
			t.Fatal("CanonicalKindOfType did not follow alias chain to sprite")
		}
		if got, ok := schema.KindOfHandle(sprite.HandleTypes[0]); !ok || got != sprite {
			t.Fatal("KindOfHandle did not resolve sprite handle type")
		}

		handleNamed, ok := types.Unalias(sprite.HandleTypes[0].Type()).(*types.Named)
		if !ok {
			t.Fatalf("unexpected handle type: %T", sprite.HandleTypes[0].Type())
		}
		if got, ok := schema.HandleKindOfType(types.NewPointer(handleNamed)); !ok || got != sprite {
			t.Fatal("HandleKindOfType did not resolve sprite handle type")
		}

		setCostume := lookupMethod(t, handleNamed, "SetCostume__0")
		apiScopeBindings := schema.APIScopeBindings(setCostume)
		if len(apiScopeBindings) != 1 || apiScopeBindings[0].TargetParam != 0 || !apiScopeBindings[0].Source.Receiver {
			t.Fatalf("unexpected bindings for SetCostume__0: %#v", apiScopeBindings)
		}

		setCostumeAndFrame := lookupMethod(t, handleNamed, "SetCostumeAndFrame")
		apiScopeBindings = schema.APIScopeBindings(setCostumeAndFrame)
		if len(apiScopeBindings) != 2 {
			t.Fatalf("unexpected binding count for SetCostumeAndFrame: %#v", apiScopeBindings)
		}
		if apiScopeBindings[0].TargetParam != 0 || !apiScopeBindings[0].Source.Receiver {
			t.Fatalf("unexpected first binding for SetCostumeAndFrame: %#v", apiScopeBindings[0])
		}
		if apiScopeBindings[1].TargetParam != 1 || apiScopeBindings[1].Source.Receiver || apiScopeBindings[1].Source.Param != 0 {
			t.Fatalf("unexpected second binding for SetCostumeAndFrame: %#v", apiScopeBindings[1])
		}
	})

	t.Run("ValidGoInterfaceMethodSpec", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

// SpriteName identifies a sprite by name.
//
//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

// SpriteCostumeName identifies a sprite costume by name.
//
//xgo:class:resource sprite.costume
//xgo:class:resource-discovery costumes.*
type SpriteCostumeName = string

// Sprite is a handle-bearing type of the sprite resource kind.
//
//xgo:class:resource sprite
type Sprite interface {
	//xgo:class:resource-api-scope-binding param.0 receiver
	SetCostume__0(costume SpriteCostumeName)
}
`,
		})

		schema, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err != nil {
			t.Fatalf("LoadResourceSchema failed: %v", err)
		}

		sprite, ok := schema.Kind("sprite")
		if !ok {
			t.Fatal("sprite kind not found")
		}
		if len(sprite.HandleTypes) != 1 || sprite.HandleTypes[0].Name() != "Sprite" {
			t.Fatalf("unexpected sprite handle types: %#v", sprite.HandleTypes)
		}

		handleNamed, ok := types.Unalias(sprite.HandleTypes[0].Type()).(*types.Named)
		if !ok {
			t.Fatalf("unexpected handle type: %T", sprite.HandleTypes[0].Type())
		}
		iface, ok := handleNamed.Underlying().(*types.Interface)
		if !ok {
			t.Fatalf("unexpected handle underlying type: %T", handleNamed.Underlying())
		}
		if iface.NumExplicitMethods() != 1 {
			t.Fatalf("unexpected explicit method count: %d", iface.NumExplicitMethods())
		}
		apiScopeBindings := schema.APIScopeBindings(iface.ExplicitMethod(0))
		if len(apiScopeBindings) != 1 || apiScopeBindings[0].TargetParam != 0 || !apiScopeBindings[0].Source.Receiver {
			t.Fatalf("unexpected bindings for Sprite.SetCostume__0: %#v", apiScopeBindings)
		}
	})

	t.Run("ValidXGoPackage", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.xgo": `package spx

// SpriteName identifies a sprite by name.
//
//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

// SpriteCostumeName identifies a sprite costume by name.
//
//xgo:class:resource sprite.costume
//xgo:class:resource-discovery costumes.*
type SpriteCostumeName = string

// SpriteImpl is a handle-bearing type of the sprite resource kind.
//
//xgo:class:resource sprite
type SpriteImpl struct{}

// SetCostume__0 sets the current costume.
//
//xgo:class:resource-api-scope-binding param.0 receiver
func (s *SpriteImpl) SetCostume__0(costume SpriteCostumeName) {}
`,
		})

		schema, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err != nil {
			t.Fatalf("LoadResourceSchema failed: %v", err)
		}

		sprite, ok := schema.Kind("sprite")
		if !ok {
			t.Fatal("sprite kind not found")
		}
		if sprite.CanonicalType == nil || sprite.CanonicalType.Name() != "SpriteName" {
			t.Fatalf("unexpected sprite canonical type: %#v", sprite.CanonicalType)
		}
		if sprite.DiscoveryQuery != "sprites.*" {
			t.Fatalf("unexpected sprite discovery query: %q", sprite.DiscoveryQuery)
		}
		if len(sprite.HandleTypes) != 1 || sprite.HandleTypes[0].Name() != "SpriteImpl" {
			t.Fatalf("unexpected sprite handle types: %#v", sprite.HandleTypes)
		}

		handleNamed, ok := types.Unalias(sprite.HandleTypes[0].Type()).(*types.Named)
		if !ok {
			t.Fatalf("unexpected handle type: %T", sprite.HandleTypes[0].Type())
		}
		setCostume := lookupMethod(t, handleNamed, "SetCostume__0")
		apiScopeBindings := schema.APIScopeBindings(setCostume)
		if len(apiScopeBindings) != 1 || apiScopeBindings[0].TargetParam != 0 || !apiScopeBindings[0].Source.Receiver {
			t.Fatalf("unexpected bindings for SetCostume__0: %#v", apiScopeBindings)
		}
	})

	t.Run("GroupedTypeSpec", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

type (
	// SpriteName identifies a sprite by name.
	//
	//xgo:class:resource sprite
	//xgo:class:resource-discovery sprites.*
	SpriteName = string

	// SoundName identifies a sound by name.
	//
	//xgo:class:resource sound
	//xgo:class:resource-discovery sounds.*
	SoundName = string
)
`,
		})

		schema, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err != nil {
			t.Fatalf("LoadResourceSchema failed: %v", err)
		}

		sprite, ok := schema.Kind("sprite")
		if !ok {
			t.Fatal("sprite kind not found")
		}
		if sprite.CanonicalType == nil || sprite.CanonicalType.Name() != "SpriteName" {
			t.Fatalf("unexpected sprite canonical type: %#v", sprite.CanonicalType)
		}
		if sprite.DiscoveryQuery != "sprites.*" {
			t.Fatalf("unexpected sprite discovery query: %q", sprite.DiscoveryQuery)
		}

		sound, ok := schema.Kind("sound")
		if !ok {
			t.Fatal("sound kind not found")
		}
		if sound.CanonicalType == nil || sound.CanonicalType.Name() != "SoundName" {
			t.Fatalf("unexpected sound canonical type: %#v", sound.CanonicalType)
		}
		if sound.DiscoveryQuery != "sounds.*" {
			t.Fatalf("unexpected sound discovery query: %q", sound.DiscoveryQuery)
		}
	})

	t.Run("GroupedDeclarationDocIsIgnored", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type (
	SpriteName = string
	SoundName  = string
)
`,
		})

		schema, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err != nil {
			t.Fatalf("LoadResourceSchema failed: %v", err)
		}
		if len(schema.Kinds) != 0 {
			t.Fatalf("unexpected kinds from grouped declaration doc: %#v", schema.Kinds)
		}
	})

	t.Run("DuplicateCanonicalType", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName2 = string
`,
		})

		_, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err == nil || !strings.Contains(err.Error(), `resource kind "sprite" has more than one canonical type`) {
			t.Fatalf("expected duplicate canonical type error, got %v", err)
		}
		if !strings.Contains(err.Error(), "previous at") {
			t.Fatalf("expected duplicate canonical type error to include previous position, got %v", err)
		}
	})

	t.Run("ScopedHandleBearingType", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource sprite.costume
type SpriteCostumeImpl struct{}
`,
		})

		_, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err == nil || !strings.Contains(err.Error(), "must declare one top-level resource kind") {
			t.Fatalf("expected scoped handle-bearing type error, got %v", err)
		}
	})

	t.Run("InvalidBindingsDoNotTriggerCycle", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

//xgo:class:resource sprite.costume
//xgo:class:resource-discovery costumes.*
type SpriteCostumeName = string

//xgo:class:resource sprite
type SpriteImpl struct{}

//xgo:class:resource-api-scope-binding param.0 param.1
//xgo:class:resource-api-scope-binding param.1 param.0
func (s *SpriteImpl) SetCostume(costume SpriteCostumeName, unrelated string) {}
`,
		})

		_, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err == nil {
			t.Fatal("expected invalid binding error")
		}
		if strings.Contains(err.Error(), "induces a cycle") {
			t.Fatalf("unexpected cycle error for invalid bindings: %v", err)
		}
		if !strings.Contains(err.Error(), "invalid resource-api-scope-binding source param.1 for target param.0") {
			t.Fatalf("expected invalid binding source error, got %v", err)
		}
	})

	t.Run("DuplicateBindingTargetIsRejectedAsWhole", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource sprite
//xgo:class:resource-discovery sprites.*
type SpriteName = string

//xgo:class:resource sprite.costume
//xgo:class:resource-discovery costumes.*
type SpriteCostumeName = string

//xgo:class:resource sprite
type SpriteImpl struct{}

//xgo:class:resource-api-scope-binding param.0 receiver
//xgo:class:resource-api-scope-binding param.0 receiver
func (s *SpriteImpl) SetCostume(costume SpriteCostumeName) {}
`,
		})

		_, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err == nil {
			t.Fatal("expected duplicate binding target error")
		}
		if !strings.Contains(err.Error(), "duplicate resource-api-scope-binding target param.0") {
			t.Fatalf("expected duplicate target error, got %v", err)
		}
	})

	t.Run("InvalidDirectiveSyntax", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource sprite
//xgo:class:resource-discovery
type SpriteName = string

//xgo:class:resource sprite
type SpriteImpl struct{}

//xgo:class:resource-api-scope-binding param.x receiver
func (s *SpriteImpl) SetCostume__0(costume SpriteName) {}
`,
		})

		_, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err == nil {
			t.Fatal("expected invalid directive syntax error")
		}
		errs, ok := err.(scanner.ErrorList)
		if !ok {
			t.Fatalf("expected scanner.ErrorList, got %T", err)
		}
		var foundDiscovery, foundBinding bool
		for _, scanErr := range errs {
			switch {
			case strings.Contains(scanErr.Msg, "invalid resource-discovery directive syntax"):
				foundDiscovery = true
			case strings.Contains(scanErr.Msg, "invalid resource-api-scope-binding directive syntax"):
				foundBinding = true
			}
		}
		if !foundDiscovery {
			t.Fatalf("expected invalid resource-discovery syntax error, got %v", err)
		}
		if !foundBinding {
			t.Fatalf("expected invalid scope-binding syntax error, got %v", err)
		}
	})

	t.Run("NameDiscoveryWithoutDiscoveryIsIgnored", func(t *testing.T) {
		pkgPath := "example.com/spx"
		pkg, conf := loadTestPackage(t, map[string]string{
			"spx.go": `package spx

//xgo:class:resource widget
//xgo:class:resource-name-discovery id
//xgo:class:resource-name-discovery name
type WidgetName = string
`,
		})

		schema, err := LoadResourceSchema(pkg, &modfile.Project{PkgPaths: []string{pkgPath}}, conf)
		if err != nil {
			t.Fatalf("expected name-discovery directives without discovery to be ignored, got %v", err)
		}
		widget, ok := schema.Kind("widget")
		if !ok {
			t.Fatal("widget kind not found")
		}
		if widget.NameDiscoveryQuery != "" {
			t.Fatalf("unexpected widget name-discovery query: %q", widget.NameDiscoveryQuery)
		}
	})
}

// loadTestPackage parses one temporary package and returns its XGo package
// representation together with one outline config.
func loadTestPackage(t *testing.T, files map[string]string) (*xast.Package, *outline.Config) {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", path, err)
		}
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDirEx(fset, dir, parser.Config{
		Mode: parser.ParseComments,
	})
	if err != nil {
		t.Fatalf("ParseDirEx(%q) failed: %v", dir, err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected exactly one package, got %d", len(pkgs))
	}
	for _, pkg := range pkgs {
		return pkg, &outline.Config{Fset: fset}
	}
	t.Fatal("no package returned")
	return nil, nil
}

// lookupMethod looks up one method of the named handle-bearing type.
func lookupMethod(t *testing.T, named *types.Named, name string) *types.Func {
	t.Helper()

	methods := types.NewMethodSet(types.NewPointer(named))
	sel := methods.Lookup(named.Obj().Pkg(), name)
	if sel == nil {
		t.Fatalf("method %s not found on %s", name, named.Obj().Name())
	}
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		t.Fatalf("selection %s is not one method", name)
	}
	return fn
}
