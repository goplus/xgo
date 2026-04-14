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

// Package classfile loads classfile resource schemas declared by framework packages.
package classfile

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/goplus/mod/modfile"
	xast "github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/cl/outline"
	"github.com/goplus/xgo/token"
)

// ResourceSchema is the classfile resource schema loaded from one framework package.
type ResourceSchema struct {
	// Package is the typed framework package from which the schema was loaded.
	Package *types.Package
	// Kinds contains all resource kinds declared by Package.
	Kinds []*ResourceKind

	byKind           map[string]*ResourceKind
	byCanonical      map[*types.TypeName]*ResourceKind
	byHandle         map[*types.TypeName]*ResourceKind
	apiScopeBindings map[*types.Func][]ResourceAPIScopeBinding
}

// LoadResourceSchema loads the classfile resource schema declared by pkg, using
// the first package path in proj as its framework package path.
func LoadResourceSchema(pkg *xast.Package, proj *modfile.Project, conf *outline.Config) (*ResourceSchema, error) {
	if len(proj.PkgPaths) == 0 {
		return nil, fmt.Errorf("project has no framework package path")
	}
	pkgPath := proj.PkgPaths[0]

	outlinePkg, err := outline.NewPackage(pkgPath, pkg, conf)
	if err != nil {
		return nil, err
	}
	typesPkg := outlinePkg.Pkg()

	loader := resourceSchemaLoader{
		fset:      conf.Fset,
		syntaxPkg: pkg,
		typesPkg:  typesPkg,
		schema: &ResourceSchema{
			Package:          typesPkg,
			byKind:           make(map[string]*ResourceKind),
			byCanonical:      make(map[*types.TypeName]*ResourceKind),
			byHandle:         make(map[*types.TypeName]*ResourceKind),
			apiScopeBindings: make(map[*types.Func][]ResourceAPIScopeBinding),
		},
	}
	loader.scanPackage()
	loader.validateKinds()
	loader.validateAPIScopeBindings()
	if len(loader.errs) != 0 {
		loader.errs.Sort()
		return nil, loader.errs
	}
	return loader.schema, nil
}

// Kind reports the resource kind by its canonical spelling.
func (s *ResourceSchema) Kind(name string) (*ResourceKind, bool) {
	ret, ok := s.byKind[name]
	return ret, ok
}

// KindOfCanonical reports the resource kind declared by one canonical resource
// reference type declaration.
func (s *ResourceSchema) KindOfCanonical(obj *types.TypeName) (*ResourceKind, bool) {
	ret, ok := s.byCanonical[obj]
	return ret, ok
}

// CanonicalKindOfType reports the canonical resource kind determined by typ by
// following alias declarations only.
func (s *ResourceSchema) CanonicalKindOfType(typ types.Type) (*ResourceKind, bool) {
	for typ != nil {
		switch t := typ.(type) {
		case *types.Named:
			return s.KindOfCanonical(t.Obj())
		case *types.Alias:
			if kind, ok := s.KindOfCanonical(t.Obj()); ok {
				return kind, true
			}
			typ = t.Rhs()
		default:
			return nil, false
		}
	}
	return nil, false
}

// KindOfHandle reports the resource kind declared by one handle-bearing type declaration.
func (s *ResourceSchema) KindOfHandle(obj *types.TypeName) (*ResourceKind, bool) {
	ret, ok := s.byHandle[obj]
	return ret, ok
}

// HandleKindOfType reports the handle-bearing resource kind determined by typ.
func (s *ResourceSchema) HandleKindOfType(typ types.Type) (*ResourceKind, bool) {
	for {
		typ = types.Unalias(typ)
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = ptr.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return nil, false
	}
	return s.KindOfHandle(named.Obj())
}

// APIScopeBindings reports the standardized API-position scope bindings
// declared on fn.
func (s *ResourceSchema) APIScopeBindings(fn *types.Func) []ResourceAPIScopeBinding {
	ret := s.apiScopeBindings[fn]
	if len(ret) == 0 {
		return nil
	}
	return slices.Clone(ret)
}

// ResourceKind is one resource kind declared in framework source.
type ResourceKind struct {
	// Name is the canonical resource kind spelling.
	Name string
	// CanonicalType is the canonical resource reference type declaration, if any.
	CanonicalType *types.TypeName
	// HandleTypes contains the handle-bearing type declarations of this kind.
	HandleTypes []*types.TypeName
	// DiscoveryQuery is the resource-discovery query fragment, if any.
	DiscoveryQuery string
	// NameDiscoveryQuery is the resource-name-discovery query fragment, if any.
	NameDiscoveryQuery string

	pos token.Pos
}

// ResourceAPIScopeBinding is one standardized resource-api-scope-binding.
type ResourceAPIScopeBinding struct {
	// TargetParam is the zero-based target parameter index.
	TargetParam int
	// Source is the direct scope source of the target parameter.
	Source ResourceAPIScopeSource
}

// ResourceAPIScopeSource is one direct scope source of one API-position binding.
type ResourceAPIScopeSource struct {
	// Receiver reports whether the source is the callable receiver.
	Receiver bool
	// Param is the zero-based source parameter index when Receiver is false.
	Param int
}
