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

package fromgo

import (
	"go/ast"
	"go/token"
	"log"
	"reflect"

	gopast "github.com/goplus/xgo/ast"
	goptoken "github.com/goplus/xgo/token"

	"github.com/goplus/xgo/ast/fromgo/typeparams"
)

// ----------------------------------------------------------------------------

func gopExpr(val ast.Expr) gopast.Expr {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case *ast.Ident:
		return gopIdent(v)
	case *ast.SelectorExpr:
		return &gopast.SelectorExpr{
			X:   gopExpr(v.X),
			Sel: gopIdent(v.Sel),
		}
	case *ast.SliceExpr:
		return &gopast.SliceExpr{
			X:      gopExpr(v.X),
			Lbrack: v.Lbrack,
			Low:    gopExpr(v.Low),
			High:   gopExpr(v.High),
			Max:    gopExpr(v.Max),
			Slice3: v.Slice3,
			Rbrack: v.Rbrack,
		}
	case *ast.StarExpr:
		return &gopast.StarExpr{
			Star: v.Star,
			X:    gopExpr(v.X),
		}
	case *ast.MapType:
		return &gopast.MapType{
			Map:   v.Map,
			Key:   gopType(v.Key),
			Value: gopType(v.Value),
		}
	case *ast.StructType:
		return &gopast.StructType{
			Struct: v.Struct,
			Fields: gopFieldList(v.Fields),
		}
	case *ast.FuncType:
		return gopFuncType(v)
	case *ast.InterfaceType:
		return &gopast.InterfaceType{
			Interface: v.Interface,
			Methods:   gopFieldList(v.Methods),
		}
	case *ast.ArrayType:
		return &gopast.ArrayType{
			Lbrack: v.Lbrack,
			Len:    gopExpr(v.Len),
			Elt:    gopType(v.Elt),
		}
	case *ast.ChanType:
		return &gopast.ChanType{
			Begin: v.Begin,
			Arrow: v.Arrow,
			Dir:   gopast.ChanDir(v.Dir),
			Value: gopType(v.Value),
		}
	case *ast.BasicLit:
		return gopBasicLit(v)
	case *ast.BinaryExpr:
		return &gopast.BinaryExpr{
			X:     gopExpr(v.X),
			OpPos: v.OpPos,
			Op:    goptoken.Token(v.Op),
			Y:     gopExpr(v.Y),
		}
	case *ast.UnaryExpr:
		return &gopast.UnaryExpr{
			OpPos: v.OpPos,
			Op:    goptoken.Token(v.Op),
			X:     gopExpr(v.X),
		}
	case *ast.CallExpr:
		return &gopast.CallExpr{
			Fun:      gopExpr(v.Fun),
			Lparen:   v.Lparen,
			Args:     gopExprs(v.Args),
			Ellipsis: v.Ellipsis,
			Rparen:   v.Rparen,
		}
	case *ast.IndexExpr:
		return &gopast.IndexExpr{
			X:      gopExpr(v.X),
			Lbrack: v.Lbrack,
			Index:  gopExpr(v.Index),
			Rbrack: v.Rbrack,
		}
	case *typeparams.IndexListExpr:
		return &gopast.IndexListExpr{
			X:       gopExpr(v.X),
			Lbrack:  v.Lbrack,
			Indices: gopExprs(v.Indices),
			Rbrack:  v.Rbrack,
		}
	case *ast.ParenExpr:
		return &gopast.ParenExpr{
			Lparen: v.Lparen,
			X:      gopExpr(v.X),
			Rparen: v.Rparen,
		}
	case *ast.CompositeLit:
		return &gopast.CompositeLit{
			Type:   gopType(v.Type),
			Lbrace: v.Lbrace,
			Elts:   gopExprs(v.Elts),
			Rbrace: v.Rbrace,
		}
	case *ast.FuncLit:
		return &gopast.FuncLit{
			Type: gopFuncType(v.Type),
			Body: &gopast.BlockStmt{}, // skip closure body
		}
	case *ast.TypeAssertExpr:
		return &gopast.TypeAssertExpr{
			X:      gopExpr(v.X),
			Lparen: v.Lparen,
			Type:   gopType(v.Type),
			Rparen: v.Rparen,
		}
	case *ast.KeyValueExpr:
		return &gopast.KeyValueExpr{
			Key:   gopExpr(v.Key),
			Colon: v.Colon,
			Value: gopExpr(v.Value),
		}
	case *ast.Ellipsis:
		return &gopast.Ellipsis{
			Ellipsis: v.Ellipsis,
			Elt:      gopExpr(v.Elt),
		}
	}
	log.Panicln("gopExpr: unknown expr -", reflect.TypeOf(val))
	return nil
}

func gopExprs(vals []ast.Expr) []gopast.Expr {
	n := len(vals)
	if n == 0 {
		return nil
	}
	ret := make([]gopast.Expr, n)
	for i, v := range vals {
		ret[i] = gopExpr(v)
	}
	return ret
}

// ----------------------------------------------------------------------------

func gopFuncType(v *ast.FuncType) *gopast.FuncType {
	return &gopast.FuncType{
		Func:       v.Func,
		TypeParams: gopFieldList(typeparams.ForFuncType(v)),
		Params:     gopFieldList(v.Params),
		Results:    gopFieldList(v.Results),
	}
}

func gopType(v ast.Expr) gopast.Expr {
	return gopExpr(v)
}

func gopBasicLit(v *ast.BasicLit) *gopast.BasicLit {
	if v == nil {
		return nil
	}
	return &gopast.BasicLit{
		ValuePos: v.ValuePos,
		Kind:     goptoken.Token(v.Kind),
		Value:    v.Value,
	}
}

func gopIdent(v *ast.Ident) *gopast.Ident {
	if v == nil {
		return nil
	}
	return &gopast.Ident{
		NamePos: v.NamePos,
		Name:    v.Name,
		Obj:     &gopast.Object{Data: v},
	}
}

// CheckIdent checks if a XGo ast.Ident is converted from a Go ast.Ident or not.
// If it is, CheckIdent returns the original Go ast.Ident object.
func CheckIdent(v *gopast.Ident) (id *ast.Ident, ok bool) {
	if o := v.Obj; o != nil && o.Kind == 0 && o.Data != nil {
		id, ok = o.Data.(*ast.Ident)
	}
	return
}

func gopIdents(names []*ast.Ident) []*gopast.Ident {
	ret := make([]*gopast.Ident, len(names))
	for i, v := range names {
		ret[i] = gopIdent(v)
	}
	return ret
}

// ----------------------------------------------------------------------------

func gopField(v *ast.Field) *gopast.Field {
	return &gopast.Field{
		Names: gopIdents(v.Names),
		Type:  gopType(v.Type),
		Tag:   gopBasicLit(v.Tag),
	}
}

func gopFieldList(v *ast.FieldList) *gopast.FieldList {
	if v == nil {
		return nil
	}
	list := make([]*gopast.Field, len(v.List))
	for i, item := range v.List {
		list[i] = gopField(item)
	}
	return &gopast.FieldList{Opening: v.Opening, List: list, Closing: v.Closing}
}

func gopFuncDecl(v *ast.FuncDecl) *gopast.FuncDecl {
	return &gopast.FuncDecl{
		Doc:  v.Doc,
		Recv: gopFieldList(v.Recv),
		Name: gopIdent(v.Name),
		Type: gopFuncType(v.Type),
		Body: &gopast.BlockStmt{}, // ignore function body
	}
}

// ----------------------------------------------------------------------------

func gopImportSpec(spec *ast.ImportSpec) *gopast.ImportSpec {
	return &gopast.ImportSpec{
		Name:   gopIdent(spec.Name),
		Path:   gopBasicLit(spec.Path),
		EndPos: spec.EndPos,
	}
}

func gopTypeSpec(spec *ast.TypeSpec) *gopast.TypeSpec {
	return &gopast.TypeSpec{
		Name:       gopIdent(spec.Name),
		TypeParams: gopFieldList(typeparams.ForTypeSpec(spec)),
		Assign:     spec.Assign,
		Type:       gopType(spec.Type),
	}
}

func gopValueSpec(spec *ast.ValueSpec) *gopast.ValueSpec {
	return &gopast.ValueSpec{
		Names:  gopIdents(spec.Names),
		Type:   gopType(spec.Type),
		Values: gopExprs(spec.Values),
	}
}

func gopGenDecl(v *ast.GenDecl) *gopast.GenDecl {
	specs := make([]gopast.Spec, len(v.Specs))
	for i, spec := range v.Specs {
		switch v.Tok {
		case token.IMPORT:
			specs[i] = gopImportSpec(spec.(*ast.ImportSpec))
		case token.TYPE:
			specs[i] = gopTypeSpec(spec.(*ast.TypeSpec))
		case token.VAR, token.CONST:
			specs[i] = gopValueSpec(spec.(*ast.ValueSpec))
		default:
			log.Panicln("gopGenDecl: unknown spec -", v.Tok)
		}
	}
	return &gopast.GenDecl{
		Doc:    v.Doc,
		TokPos: v.TokPos,
		Tok:    goptoken.Token(v.Tok),
		Lparen: v.Lparen,
		Specs:  specs,
		Rparen: v.Rparen,
	}
}

// ----------------------------------------------------------------------------

func gopDecl(decl ast.Decl) gopast.Decl {
	switch v := decl.(type) {
	case *ast.GenDecl:
		return gopGenDecl(v)
	case *ast.FuncDecl:
		return gopFuncDecl(v)
	}
	log.Panicln("gopDecl: unknown decl -", reflect.TypeOf(decl))
	return nil
}

func gopDecls(decls []ast.Decl) []gopast.Decl {
	ret := make([]gopast.Decl, len(decls))
	for i, decl := range decls {
		ret[i] = gopDecl(decl)
	}
	return ret
}

// ----------------------------------------------------------------------------

const (
	KeepFuncBody = 1 << iota
	KeepCgo
)

// ASTFile converts a Go ast.File into a XGo ast.File object.
func ASTFile(f *ast.File, mode int) *gopast.File {
	if (mode & KeepFuncBody) != 0 {
		log.Panicln("ASTFile: doesn't support keeping func body now")
	}
	if (mode & KeepCgo) != 0 {
		log.Panicln("ASTFile: doesn't support keeping cgo now")
	}
	return &gopast.File{
		Doc:     f.Doc,
		Package: f.Package,
		Name:    gopIdent(f.Name),
		Decls:   gopDecls(f.Decls),
	}
}

// ----------------------------------------------------------------------------
