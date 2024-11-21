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

package format

import (
	"bytes"
	"go/types"
	"path"
	"strconv"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/format"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

// -----------------------------------------------------------------------------

func XGoStyleSource(src []byte, class bool, filename ...string) (ret []byte, err error) {
	var fname string
	if filename != nil {
		fname = filename[0]
	}
	fset := token.NewFileSet()
	mode := parser.ParseComments
	if class {
		mode |= parser.ParseXGoClass
	}
	var f *ast.File
	if f, err = parser.ParseFile(fset, fname, src, mode); err == nil {
		XGoStyle(f)
		var buf bytes.Buffer
		if err = format.Node(&buf, fset, f); err == nil {
			ret = buf.Bytes()
		}
	}
	return
}

// Deprecated: Use XGoStyleSource instead.
func GopstyleSource(src []byte, filename ...string) (ret []byte, err error) {
	return XGoStyleSource(src, false, filename...)
}

// -----------------------------------------------------------------------------

func XGoStyle(file *ast.File) {
	if identEqual(file.Name, "main") {
		file.NoPkgDecl = true
	}
	if idx, fn := findFuncDecl(file.Decls, "main"); idx >= 0 {
		last := len(file.Decls) - 1
		if idx == last {
			file.ShadowEntry = fn
			// TODO: idx != last: swap main func to last
			// TODO: should also swap file.Comments
			/*
				fn := file.Decls[idx]
				copy(file.Decls[idx:], file.Decls[idx+1:])
				file.Decls[last] = fn
			*/
		}
	}
	formatFile(file)
}

// Deprecated: Use XGoStyle instead.
func Gopstyle(file *ast.File) {
	XGoStyle(file)
}

func XGoClassSource(src []byte, pkg string, class string, entry string, filename ...string) (ret []byte, err error) {
	var fname string
	if filename != nil {
		fname = filename[0]
	}
	fset := token.NewFileSet()
	var f *ast.File
	if f, err = parser.ParseFile(fset, fname, src, parser.ParseComments); err == nil {
		XGoClass(f, pkg, class, entry)
		var buf bytes.Buffer
		if err = format.Node(&buf, fset, f); err == nil {
			ret = buf.Bytes()
		}
	}
	return
}

// GopClass format ast.File to Go+ class
func XGoClass(file *ast.File, pkg string, class string, entry string) {
	formatClass(file, pkg, class, entry)
}

func findFuncDecl(decls []ast.Decl, name string) (int, *ast.FuncDecl) {
	for i, decl := range decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if identEqual(fn.Name, name) {
				return i, fn
			}
		}
	}
	return -1, nil
}

func findDecl(decls []ast.Decl, v ast.Decl) int {
	for i, decl := range decls {
		if decl == v {
			return i
		}
	}
	return -1
}

func findSpec(specs []ast.Spec, v ast.Spec) int {
	for i, spec := range specs {
		if spec == v {
			return i
		}
	}
	return -1
}

func deleteDecl(decls []ast.Decl, v ast.Decl) []ast.Decl {
	if idx := findDecl(decls, v); idx >= 0 {
		decls = append(decls[:idx], decls[idx+1:]...)
	}
	return decls
}

func deleteSpec(specs []ast.Spec, v ast.Spec) []ast.Spec {
	if idx := findSpec(specs, v); idx >= 0 {
		specs = append(specs[:idx], specs[idx+1:]...)
	}
	return specs
}

func startWithLowerCase(v *ast.Ident) {
	if c := v.Name[0]; c >= 'A' && c <= 'Z' {
		v.Name = string(c+('a'-'A')) + v.Name[1:]
	}
}

func identEqual(v *ast.Ident, name string) bool {
	return v != nil && v.Name == name
}

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

type importCtx struct {
	pkgPath string
	decl    *ast.GenDecl
	spec    *ast.ImportSpec
	isUsed  bool
}

type formatCtx struct {
	imports   map[string]*importCtx
	scope     *types.Scope
	classMode bool   //class mode
	classPkg  string //class pkg name
	className string //this class
	funcRecv  string //this class func recv
}

func (ctx *formatCtx) insert(name string) {
	o := types.NewParam(token.NoPos, nil, name, types.Typ[types.UntypedNil])
	ctx.scope.Insert(o)
}

func (ctx *formatCtx) enterBlock() *types.Scope {
	old := ctx.scope
	ctx.scope = types.NewScope(old, token.NoPos, token.NoPos, "")
	return old
}

func (ctx *formatCtx) leaveBlock(old *types.Scope) {
	ctx.scope = old
}

func formatFile(file *ast.File) {
	var funcs []*ast.FuncDecl
	ctx := &formatCtx{
		imports: make(map[string]*importCtx),
		scope:   types.NewScope(nil, token.NoPos, token.NoPos, ""),
	}
	for _, decl := range file.Decls {
		switch v := decl.(type) {
		case *ast.FuncDecl:
			// delay the process, because package level vars need to be processed first.
			funcs = append(funcs, v)
		case *ast.GenDecl:
			switch v.Tok {
			case token.IMPORT:
				for _, item := range v.Specs {
					var spec = item.(*ast.ImportSpec)
					var pkgPath = toString(spec.Path)
					var name string
					if spec.Name == nil {
						name = path.Base(pkgPath) // TODO: open pkgPath to get pkgName
					} else {
						name = spec.Name.Name
						if name == "." || name == "_" {
							continue
						}
					}
					ctx.imports[name] = &importCtx{pkgPath: pkgPath, decl: v, spec: spec}
				}
			default:
				formatGenDecl(ctx, v)
			}
		}
	}

	for _, fn := range funcs {
		formatFuncDecl(ctx, fn)
	}
	for _, imp := range ctx.imports {
		if imp.pkgPath == "fmt" && !imp.isUsed {
			if len(imp.decl.Specs) == 1 {
				file.Decls = deleteDecl(file.Decls, imp.decl)
			} else {
				imp.decl.Specs = deleteSpec(imp.decl.Specs, imp.spec)
			}
		}
	}
}

func formatClass(file *ast.File, pkg string, class string, entry string) {
	var funcs []*ast.FuncDecl
	ctx := &formatCtx{
		imports:   make(map[string]*importCtx),
		scope:     types.NewScope(nil, token.NoPos, token.NoPos, ""),
		classMode: true,
		classPkg:  path.Base(pkg),
		className: class,
	}
	if file.Name.Name == "main" {
		file.NoPkgDecl = true
	}
	var fnEntry *ast.FuncDecl
	var decls []ast.Decl
	var varSpecs []ast.Spec
	for _, decl := range file.Decls {
		switch v := decl.(type) {
		case *ast.FuncDecl:
			if isClassFunc(v, class) {
				v.IsClass = true
				switch v.Name.Name {
				case entry:
					fnEntry = v
					file.ShadowEntry = v
					continue
				case "Classfname":
					v.Shadow = true
				}
			}
		case *ast.GenDecl:
			if v.Tok == token.TYPE {
				if spec, ok := v.Specs[0].(*ast.TypeSpec); ok && spec.Name.Name == class {
					if st, ok := spec.Type.(*ast.StructType); ok {
						for _, fs := range st.Fields.List {
							if len(fs.Names) == 0 {
								continue
							}
							varSpecs = append(varSpecs, &ast.ValueSpec{Names: fs.Names, Type: fs.Type})
						}
						continue
					}
				}
			}
		}
		decls = append(decls, decl)
	}
	if len(varSpecs) != 0 {
		decls = append([]ast.Decl{&ast.GenDecl{Tok: token.VAR, Specs: varSpecs}}, decls...)
	}
	if fnEntry != nil {
		decls = append(decls, fnEntry)
	}
	file.Decls = decls

	for _, decl := range file.Decls {
		switch v := decl.(type) {
		case *ast.FuncDecl:
			// delay the process, because package level vars need to be processed first.
			funcs = append(funcs, v)
		case *ast.GenDecl:
			switch v.Tok {
			case token.IMPORT:
				for _, item := range v.Specs {
					var spec = item.(*ast.ImportSpec)
					var pkgPath = toString(spec.Path)
					var name string
					if spec.Name == nil {
						name = path.Base(pkgPath) // TODO: open pkgPath to get pkgName
					} else {
						name = spec.Name.Name
						if name == "." || name == "_" {
							continue
						}
					}
					ctx.imports[name] = &importCtx{pkgPath: pkgPath, decl: v, spec: spec}
				}
			default:
				formatGenDecl(ctx, v)
			}
		}
	}

	for _, fn := range funcs {
		formatFuncDecl(ctx, fn)
	}
	for _, imp := range ctx.imports {
		if imp.pkgPath == "fmt" && !imp.isUsed {
			if len(imp.decl.Specs) == 1 {
				file.Decls = deleteDecl(file.Decls, imp.decl)
			} else {
				imp.decl.Specs = deleteSpec(imp.decl.Specs, imp.spec)
			}
		}
	}
}

func formatGenDecl(ctx *formatCtx, v *ast.GenDecl) {
	switch v.Tok {
	case token.VAR, token.CONST:
		for _, item := range v.Specs {
			spec := item.(*ast.ValueSpec)
			formatType(ctx, spec.Type, &spec.Type)
			formatExprs(ctx, spec.Values)
			for _, name := range spec.Names {
				ctx.insert(name.Name)
			}
		}
	case token.TYPE:
		for _, item := range v.Specs {
			spec := item.(*ast.TypeSpec)
			formatType(ctx, spec.Type, &spec.Type)
		}
	}
}

func isClassFunc(v *ast.FuncDecl, className string) bool {
	if v.Recv != nil && len(v.Recv.List) == 1 {
		typ := v.Recv.List[0].Type
		if star, ok := typ.(*ast.StarExpr); ok {
			typ = star.X
		}
		if ident, ok := typ.(*ast.Ident); ok && ident.Name == className {
			return true
		}
	}
	return false
}

func funcRecv(v *ast.FuncDecl) *ast.Ident {
	if v.Recv != nil && len(v.Recv.List) == 1 && len(v.Recv.List[0].Names) == 1 {
		return v.Recv.List[0].Names[0]
	}
	return nil
}

func formatFuncDecl(ctx *formatCtx, v *ast.FuncDecl) {
	if ctx.classMode && isClassFunc(v, ctx.className) {
		v.IsClass = true
		if recv := funcRecv(v); recv != nil {
			ctx.funcRecv = recv.Name
			defer func() {
				ctx.funcRecv = ""
			}()
		}
	}
	formatFuncType(ctx, v.Type)
	formatBlockStmt(ctx, v.Body)
}

/*
func fillVarCtx(ctx *formatCtx, spec *ast.ValueSpec) {
	for _, name := range spec.Names {
		ctx.scp.addVar(name.Name)
	}
}

type scope struct {
	vars []map[string]bool
}

func newScope() *scope {
	return &scope{
		vars: []map[string]bool{make(map[string]bool)},
	}
}

func (s *scope) addVar(name string) {
	s.vars[len(s.vars)-1][name] = true
}

func (s *scope) containsVar(name string) bool {
	for _, m := range s.vars {
		if m[name] {
			return true
		}
	}

	return false
}

func (s *scope) enterScope() {
	s.vars = append(s.vars, make(map[string]bool))
}

func (s *scope) exitScope() {
	s.vars = s.vars[:len(s.vars)-1]
}
*/
// -----------------------------------------------------------------------------
