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
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"go/types"
	"maps"
	"slices"
	"strings"

	xast "github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/scanner"
	"github.com/goplus/xgo/token"
)

// resourceSchemaLoader builds one [ResourceSchema] from one typed framework package.
type resourceSchemaLoader struct {
	fset   *token.FileSet
	pkg    *xast.Package
	types  *types.Package
	schema *ResourceSchema
	errs   scanner.ErrorList

	rawAPIScopeBindings map[*types.Func][]rawAPIScopeBinding
	callableHandles     map[*types.Func]*types.TypeName
}

// rawAPIScopeBinding is one unresolved API-position scope binding directive.
type rawAPIScopeBinding struct {
	pos    token.Pos
	target int
	source ResourceAPIScopeSource
}

// scanPackage scans all Go and XGo files of the framework package.
func (l *resourceSchemaLoader) scanPackage() {
	l.rawAPIScopeBindings = make(map[*types.Func][]rawAPIScopeBinding)
	l.callableHandles = make(map[*types.Func]*types.TypeName)
	for _, name := range slices.Sorted(maps.Keys(l.pkg.Files)) {
		l.scanXGoFile(l.pkg.Files[name])
	}

	for _, name := range slices.Sorted(maps.Keys(l.pkg.GoFiles)) {
		l.scanGoFile(l.pkg.GoFiles[name])
	}
}

// scanXGoFile scans one parsed XGo file.
func (l *resourceSchemaLoader) scanXGoFile(file *xast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *xast.GenDecl:
			if d.Tok == token.TYPE {
				l.scanXGoTypeDecl(d)
			}
		case *xast.FuncDecl:
			l.scanXGoFuncDecl(d)
		}
	}
}

// scanGoFile scans one parsed Go file.
func (l *resourceSchemaLoader) scanGoFile(file *goast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *goast.GenDecl:
			if d.Tok == gotoken.TYPE {
				l.scanGoTypeDecl(d)
			}
		case *goast.FuncDecl:
			l.scanGoFuncDecl(d)
		}
	}
}

// scanXGoTypeDecl scans one XGo type declaration for classfile resource directives.
func (l *resourceSchemaLoader) scanXGoTypeDecl(decl *xast.GenDecl) {
	for _, rawSpec := range decl.Specs {
		spec, ok := rawSpec.(*xast.TypeSpec)
		if !ok || spec.TypeParams != nil || !spec.Name.IsExported() {
			continue
		}
		l.scanXGoTypeSpec(spec.Name.Name, spec, parseDirectives(xgoDeclDoc(decl, spec)))
	}
}

// scanGoTypeDecl scans one Go type declaration for classfile resource directives.
func (l *resourceSchemaLoader) scanGoTypeDecl(decl *goast.GenDecl) {
	for _, rawSpec := range decl.Specs {
		spec, ok := rawSpec.(*goast.TypeSpec)
		if !ok || spec.TypeParams != nil || !spec.Name.IsExported() {
			continue
		}
		l.scanGoTypeSpec(spec.Name.Name, spec, parseDirectives(goDeclDoc(decl, spec)))
	}
}

// scanXGoTypeSpec loads classfile resource directives attached to one exported
// top-level XGo type spec.
func (l *resourceSchemaLoader) scanXGoTypeSpec(name string, spec *xast.TypeSpec, dirs directives) {
	obj, kind, ok := l.scanTypeDirectives(name, dirs)
	if !ok {
		return
	}
	if iface, ok := types.Unalias(obj.Type()).Underlying().(*types.Interface); ok {
		l.scanXGoInterfaceMethodSpecs(obj, kind, iface, spec)
	}
}

// scanGoTypeSpec loads classfile resource directives attached to one exported
// top-level Go type spec.
func (l *resourceSchemaLoader) scanGoTypeSpec(name string, spec *goast.TypeSpec, dirs directives) {
	obj, kind, ok := l.scanTypeDirectives(name, dirs)
	if !ok {
		return
	}
	if iface, ok := types.Unalias(obj.Type()).Underlying().(*types.Interface); ok {
		l.scanGoInterfaceMethodSpecs(obj, kind, iface, spec)
	}
}

// scanTypeDirectives loads classfile resource directives attached to one
// exported top-level type declaration.
func (l *resourceSchemaLoader) scanTypeDirectives(name string, dirs directives) (*types.TypeName, *ResourceKind, bool) {
	l.addTypeDirectiveSyntaxErrors(dirs)
	if len(dirs.resource) == 0 {
		return nil, nil, false
	}
	obj := l.lookupType(name)
	if obj == nil {
		return nil, nil, false
	}
	if len(dirs.resource) > 1 {
		l.addError(dirs.resource[1].pos, "duplicate resource directive on %s", name)
		return nil, nil, false
	}
	kindName, ok := parseResourceKind(dirs.resource[0].arg)
	if !ok {
		l.addError(dirs.resource[0].pos, "invalid resource kind %q", dirs.resource[0].arg)
		return nil, nil, false
	}

	if isStringBased(obj) {
		hasDiscovery := false
		if len(dirs.discovery) > 1 {
			l.addError(dirs.discovery[1].pos, "duplicate %s directive on %s", directiveResourceDiscovery, name)
		} else if len(dirs.discovery) == 1 &&
			validDQLQuery(dirs.discovery[0].pos, dirs.discovery[0].arg, directiveResourceDiscovery) {
			hasDiscovery = true
		}
		if hasDiscovery && len(dirs.nameDiscovery) > 1 {
			l.addError(dirs.nameDiscovery[1].pos, "duplicate %s directive on %s", directiveResourceNameDiscovery, name)
		}
		kind := l.ensureKind(kindName, dirs.resource[0].pos)
		if kind.CanonicalType != nil && kind.CanonicalType != obj {
			l.addError(
				dirs.resource[0].pos,
				"resource kind %q has more than one canonical type (previous at %v)",
				kindName,
				l.fset.Position(kind.CanonicalType.Pos()),
			)
			return nil, nil, false
		}
		kind.CanonicalType = obj
		l.schema.byCanonical[obj] = kind
		if hasDiscovery {
			kind.DiscoveryQuery = dirs.discovery[0].arg
		}
		if hasDiscovery && len(dirs.nameDiscovery) == 1 &&
			validDQLQuery(dirs.nameDiscovery[0].pos, dirs.nameDiscovery[0].arg, directiveResourceNameDiscovery) {
			kind.NameDiscoveryQuery = dirs.nameDiscovery[0].arg
		}
		return obj, kind, true
	}

	if isHandleBearing(obj) {
		if strings.Contains(kindName, ".") {
			l.addError(dirs.resource[0].pos, "handle-bearing type %s must declare one top-level resource kind", name)
			return nil, nil, false
		}
		kind := l.ensureKind(kindName, dirs.resource[0].pos)
		kind.HandleTypes = append(kind.HandleTypes, obj)
		l.schema.byHandle[obj] = kind
		return obj, kind, true
	}

	return nil, nil, false
}

// scanXGoInterfaceMethodSpecs loads classfile scope-binding directives
// attached to method specs declared by one handle-bearing XGo interface type.
func (l *resourceSchemaLoader) scanXGoInterfaceMethodSpecs(obj *types.TypeName, kind *ResourceKind, iface *types.Interface, spec *xast.TypeSpec) {
	xiface, ok := spec.Type.(*xast.InterfaceType)
	if !ok || kind == nil || kind != l.schema.byHandle[obj] || xiface.Methods == nil {
		return
	}
	for _, field := range xiface.Methods.List {
		if len(field.Names) != 1 {
			continue
		}
		dirs := parseDirectives(field.Doc)
		l.addScopeBindingSyntaxErrors(dirs)
		if len(dirs.apiScopeBindings) == 0 {
			continue
		}
		fn := l.lookupInterfaceMethod(iface, field.Names[0].Name)
		if fn == nil {
			continue
		}
		l.callableHandles[fn] = obj
		l.addRawAPIScopeBindings(fn, dirs.apiScopeBindings)
	}
}

// scanGoInterfaceMethodSpecs loads classfile scope-binding directives
// attached to method specs declared by one handle-bearing Go interface type.
func (l *resourceSchemaLoader) scanGoInterfaceMethodSpecs(obj *types.TypeName, kind *ResourceKind, iface *types.Interface, spec *goast.TypeSpec) {
	goiface, ok := spec.Type.(*goast.InterfaceType)
	if !ok || kind == nil || kind != l.schema.byHandle[obj] || goiface.Methods == nil {
		return
	}
	for _, field := range goiface.Methods.List {
		if len(field.Names) != 1 {
			continue
		}
		dirs := parseDirectives((*xast.CommentGroup)(field.Doc))
		l.addScopeBindingSyntaxErrors(dirs)
		if len(dirs.apiScopeBindings) == 0 {
			continue
		}
		fn := l.lookupInterfaceMethod(iface, field.Names[0].Name)
		if fn == nil {
			continue
		}
		l.callableHandles[fn] = obj
		l.addRawAPIScopeBindings(fn, dirs.apiScopeBindings)
	}
}

// scanXGoFuncDecl scans one XGo function or method declaration for classfile
// scope-binding directives.
func (l *resourceSchemaLoader) scanXGoFuncDecl(decl *xast.FuncDecl) {
	dirs := parseDirectives(decl.Doc)
	l.addScopeBindingSyntaxErrors(dirs)
	if len(dirs.apiScopeBindings) == 0 {
		return
	}
	fn := l.lookupXGoFunc(decl)
	if fn == nil {
		return
	}
	l.addRawAPIScopeBindings(fn, dirs.apiScopeBindings)
}

// scanGoFuncDecl scans one Go function or method declaration for classfile
// scope-binding directives.
func (l *resourceSchemaLoader) scanGoFuncDecl(decl *goast.FuncDecl) {
	dirs := parseDirectives(decl.Doc)
	l.addScopeBindingSyntaxErrors(dirs)
	if len(dirs.apiScopeBindings) == 0 {
		return
	}
	fn := l.lookupGoFunc(decl)
	if fn == nil {
		return
	}
	l.addRawAPIScopeBindings(fn, dirs.apiScopeBindings)
}

// addRawAPIScopeBindings records parsed scope bindings before type-driven validation.
func (l *resourceSchemaLoader) addRawAPIScopeBindings(fn *types.Func, dirs []apiScopeBindingDirective) {
	seen := make(map[int]rawAPIScopeBinding)
	invalid := make(map[int]bool)
	for _, dir := range dirs {
		if prev, ok := seen[dir.target]; ok {
			l.addError(dir.pos, "duplicate %s target param.%d", directiveResourceAPIScopeBinding, dir.target)
			if !invalid[dir.target] {
				l.addError(prev.pos, "previous binding for target param.%d", dir.target)
			}
			invalid[dir.target] = true
			continue
		}
		seen[dir.target] = rawAPIScopeBinding(dir)
	}
	for _, dir := range dirs {
		if invalid[dir.target] {
			continue
		}
		raw, ok := seen[dir.target]
		if !ok {
			continue
		}
		l.rawAPIScopeBindings[fn] = append(l.rawAPIScopeBindings[fn], raw)
		delete(seen, dir.target)
	}
}

// validateKinds validates inter-kind constraints after scanning.
func (l *resourceSchemaLoader) validateKinds() {
	for _, kind := range l.schema.Kinds {
		if strings.Contains(kind.Name, ".") {
			parent := kind.Name[:strings.LastIndexByte(kind.Name, '.')]
			if _, ok := l.schema.Kind(parent); !ok {
				l.addError(kind.pos, "resource kind %q declares undeclared direct parent kind %q", kind.Name, parent)
			}
		}
	}
}

// validateAPIScopeBindings validates raw API-position bindings and commits the ones
// with standardized meaning.
func (l *resourceSchemaLoader) validateAPIScopeBindings() {
	for fn, raws := range l.rawAPIScopeBindings {
		if len(raws) == 0 {
			continue
		}
		sig := fn.Type().(*types.Signature)
		var out []ResourceAPIScopeBinding
		var valid []rawAPIScopeBinding
		for _, raw := range raws {
			targetKind, ok := l.kindOfTarget(sig, raw)
			if !ok {
				l.addError(raw.pos, "invalid resource-api-scope-binding target param.%d", raw.target)
				continue
			}
			if !l.validSource(fn, sig, targetKind, raw) {
				if raw.source.Receiver {
					l.addError(raw.pos, "invalid resource-api-scope-binding source receiver for target param.%d", raw.target)
				} else {
					l.addError(raw.pos, "invalid resource-api-scope-binding source param.%d for target param.%d", raw.source.Param, raw.target)
				}
				continue
			}
			valid = append(valid, raw)
			out = append(out, ResourceAPIScopeBinding{TargetParam: raw.target, Source: raw.source})
		}
		if hasAPIScopeBindingCycle(valid) {
			l.addError(fn.Pos(), "resource-api-scope-binding on %s induces a cycle", fn.FullName())
			continue
		}
		if len(out) != 0 {
			l.schema.apiScopeBindings[fn] = out
		}
	}
}

// kindOfTarget reports the scoped resource kind bound at the target parameter.
func (l *resourceSchemaLoader) kindOfTarget(sig *types.Signature, raw rawAPIScopeBinding) (*ResourceKind, bool) {
	params := sig.Params()
	if raw.target < 0 || raw.target >= params.Len() {
		return nil, false
	}
	if sig.Variadic() && raw.target == params.Len()-1 {
		return nil, false
	}
	kind, ok := l.schema.CanonicalKindOfType(params.At(raw.target).Type())
	if !ok || !strings.Contains(kind.Name, ".") {
		return nil, false
	}
	return kind, true
}

// validSource reports whether one scope-binding source is valid for the target
// scoped kind.
func (l *resourceSchemaLoader) validSource(fn *types.Func, sig *types.Signature, targetKind *ResourceKind, raw rawAPIScopeBinding) bool {
	parentKind, ok := l.schema.Kind(targetKind.Name[:strings.LastIndexByte(targetKind.Name, '.')])
	if !ok {
		return false
	}
	if raw.source.Receiver {
		if recv := sig.Recv(); recv != nil && l.hasParentKind(recv.Type(), parentKind) {
			return true
		}
		handle := l.callableHandles[fn]
		return handle != nil && l.hasParentKind(handle.Type(), parentKind)
	}

	params := sig.Params()
	if raw.source.Param < 0 || raw.source.Param >= params.Len() {
		return false
	}
	if sig.Variadic() && raw.source.Param == params.Len()-1 {
		return false
	}
	return l.hasParentKind(params.At(raw.source.Param).Type(), parentKind)
}

// lookupInterfaceMethod looks up one explicit method of one interface type.
func (l *resourceSchemaLoader) lookupInterfaceMethod(iface *types.Interface, methodName string) *types.Func {
	for i := range iface.NumExplicitMethods() {
		fn := iface.ExplicitMethod(i)
		if fn.Name() == methodName {
			return fn
		}
	}
	return nil
}

// hasParentKind reports whether typ determines parentKind as either a canonical
// resource type or a handle-bearing type.
func (l *resourceSchemaLoader) hasParentKind(typ types.Type, parentKind *ResourceKind) bool {
	if kind, ok := l.schema.CanonicalKindOfType(typ); ok && kind == parentKind {
		return true
	}
	if kind, ok := l.schema.HandleKindOfType(typ); ok && kind == parentKind {
		return true
	}
	return false
}

// lookupType looks up one top-level type name in the typed package scope.
func (l *resourceSchemaLoader) lookupType(name string) *types.TypeName {
	obj, ok := l.types.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		return nil
	}
	return obj
}

// lookupXGoFunc looks up one typed XGo function or method declaration.
func (l *resourceSchemaLoader) lookupXGoFunc(decl *xast.FuncDecl) *types.Func {
	if decl.Recv == nil {
		obj, ok := l.types.Scope().Lookup(decl.Name.Name).(*types.Func)
		if !ok {
			return nil
		}
		return obj
	}
	name := xgoRecvBaseName(decl.Recv)
	if name == "" {
		return nil
	}
	return l.lookupMethod(name, decl.Name.Name)
}

// lookupGoFunc looks up one typed Go function or method declaration.
func (l *resourceSchemaLoader) lookupGoFunc(decl *goast.FuncDecl) *types.Func {
	if decl.Recv == nil {
		obj, ok := l.types.Scope().Lookup(decl.Name.Name).(*types.Func)
		if !ok {
			return nil
		}
		return obj
	}
	name := goRecvBaseName(decl.Recv)
	if name == "" {
		return nil
	}
	return l.lookupMethod(name, decl.Name.Name)
}

// lookupMethod looks up one method declared on the named receiver type.
func (l *resourceSchemaLoader) lookupMethod(recvName, methodName string) *types.Func {
	obj, ok := l.types.Scope().Lookup(recvName).(*types.TypeName)
	if !ok {
		return nil
	}
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok {
		return nil
	}
	for fn := range named.Methods() {
		if fn.Name() == methodName {
			return fn
		}
	}
	return nil
}

// ensureKind returns the resource kind record for name, creating it if needed.
func (l *resourceSchemaLoader) ensureKind(name string, pos token.Pos) *ResourceKind {
	if ret, ok := l.schema.byKind[name]; ok {
		return ret
	}
	ret := &ResourceKind{Name: name, pos: pos}
	l.schema.byKind[name] = ret
	l.schema.Kinds = append(l.schema.Kinds, ret)
	return ret
}

// addError records one schema-loading error at pos.
func (l *resourceSchemaLoader) addError(pos token.Pos, format string, args ...any) {
	l.errs.Add(l.fset.Position(pos), fmt.Sprintf(format, args...))
}

// addTypeDirectiveSyntaxErrors reports malformed type-level classfile directives.
func (l *resourceSchemaLoader) addTypeDirectiveSyntaxErrors(dirs directives) {
	for _, dir := range dirs.invalid {
		switch dir.arg {
		case directiveResource, directiveResourceDiscovery, directiveResourceNameDiscovery:
			l.addError(dir.pos, "invalid %s directive syntax", dir.arg)
		}
	}
}

// addScopeBindingSyntaxErrors reports malformed scope-binding directives.
func (l *resourceSchemaLoader) addScopeBindingSyntaxErrors(dirs directives) {
	for _, dir := range dirs.invalid {
		if dir.arg == directiveResourceAPIScopeBinding {
			l.addError(dir.pos, "invalid %s directive syntax", directiveResourceAPIScopeBinding)
		}
	}
}

// validDQLQuery reports whether arg may be accepted as one standard DQL query
// by this resource schema loader.
func validDQLQuery(pos token.Pos, arg, kind string) bool {
	// TODO: Replace this placeholder with shared DQL validation once the
	// standardized runtime DQL capability is available to classfile tooling.
	return true
}

// isStringBased reports whether obj is one exported string-based type.
func isStringBased(obj *types.TypeName) bool {
	if !obj.Exported() {
		return false
	}
	basic, ok := types.Unalias(obj.Type()).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}

// isHandleBearing reports whether obj is one exported defined interface or
// struct type.
func isHandleBearing(obj *types.TypeName) bool {
	if !obj.Exported() || obj.IsAlias() {
		return false
	}
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok {
		return false
	}
	switch named.Underlying().(type) {
	case *types.Interface, *types.Struct:
		return true
	default:
		return false
	}
}

// hasAPIScopeBindingCycle reports whether parameter-to-parameter scope
// bindings induce a cycle.
func hasAPIScopeBindingCycle(apiScopeBindings []rawAPIScopeBinding) bool {
	next := make(map[int]int)
	for _, binding := range apiScopeBindings {
		if !binding.source.Receiver {
			next[binding.target] = binding.source.Param
		}
	}
	seen := make(map[int]uint8)
	var visit func(int) bool
	visit = func(v int) bool {
		switch seen[v] {
		case 1:
			return true
		case 2:
			return false
		}
		seen[v] = 1
		if to, ok := next[v]; ok && visit(to) {
			return true
		}
		seen[v] = 2
		return false
	}
	for v := range next {
		if visit(v) {
			return true
		}
	}
	return false
}
