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

package togo

import (
	"go/ast"
	"go/token"
	"log"
	"reflect"

	gopast "github.com/goplus/xgo/ast"
	goptoken "github.com/goplus/xgo/token"
)

// ----------------------------------------------------------------------------

func goExpr(val gopast.Expr) ast.Expr {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case *gopast.Ident:
		return goIdent(v)
	case *gopast.SelectorExpr:
		return &ast.SelectorExpr{
			X:   goExpr(v.X),
			Sel: goIdent(v.Sel),
		}
	case *gopast.SliceExpr:
		return &ast.SliceExpr{
			X:      goExpr(v.X),
			Lbrack: v.Lbrack,
			Low:    goExpr(v.Low),
			High:   goExpr(v.High),
			Max:    goExpr(v.Max),
			Slice3: v.Slice3,
			Rbrack: v.Rbrack,
		}
	case *gopast.StarExpr:
		return &ast.StarExpr{
			Star: v.Star,
			X:    goExpr(v.X),
		}
	case *gopast.MapType:
		return &ast.MapType{
			Map:   v.Map,
			Key:   goType(v.Key),
			Value: goType(v.Value),
		}
	case *gopast.StructType:
		return &ast.StructType{
			Struct: v.Struct,
			Fields: goFieldList(v.Fields),
		}
	case *gopast.FuncType:
		return goFuncType(v)
	case *gopast.InterfaceType:
		return &ast.InterfaceType{
			Interface: v.Interface,
			Methods:   goFieldList(v.Methods),
		}
	case *gopast.ArrayType:
		return &ast.ArrayType{
			Lbrack: v.Lbrack,
			Len:    goExpr(v.Len),
			Elt:    goType(v.Elt),
		}
	case *gopast.ChanType:
		return &ast.ChanType{
			Begin: v.Begin,
			Arrow: v.Arrow,
			Dir:   ast.ChanDir(v.Dir),
			Value: goType(v.Value),
		}
	case *gopast.BasicLit:
		return goBasicLit(v)
	case *gopast.BinaryExpr:
		return &ast.BinaryExpr{
			X:     goExpr(v.X),
			OpPos: v.OpPos,
			Op:    token.Token(v.Op),
			Y:     goExpr(v.Y),
		}
	case *gopast.UnaryExpr:
		return &ast.UnaryExpr{
			OpPos: v.OpPos,
			Op:    token.Token(v.Op),
			X:     goExpr(v.X),
		}
	case *gopast.CallExpr:
		return &ast.CallExpr{
			Fun:      goExpr(v.Fun),
			Lparen:   v.Lparen,
			Args:     goExprs(v.Args),
			Ellipsis: v.Ellipsis,
			Rparen:   v.Rparen,
		}
	case *gopast.IndexExpr:
		return &ast.IndexExpr{
			X:      goExpr(v.X),
			Lbrack: v.Lbrack,
			Index:  goExpr(v.Index),
			Rbrack: v.Rbrack,
		}
	case *gopast.ParenExpr:
		return &ast.ParenExpr{
			Lparen: v.Lparen,
			X:      goExpr(v.X),
			Rparen: v.Rparen,
		}
	case *gopast.CompositeLit:
		return &ast.CompositeLit{
			Type:   goType(v.Type),
			Lbrace: v.Lbrace,
			Elts:   goExprs(v.Elts),
			Rbrace: v.Rbrace,
		}
	case *gopast.FuncLit:
		return &ast.FuncLit{
			Type: goFuncType(v.Type),
			Body: &ast.BlockStmt{}, // skip closure body
		}
	case *gopast.TypeAssertExpr:
		return &ast.TypeAssertExpr{
			X:      goExpr(v.X),
			Lparen: v.Lparen,
			Type:   goType(v.Type),
			Rparen: v.Rparen,
		}
	case *gopast.KeyValueExpr:
		return &ast.KeyValueExpr{
			Key:   goExpr(v.Key),
			Colon: v.Colon,
			Value: goExpr(v.Value),
		}
	case *gopast.Ellipsis:
		return &ast.Ellipsis{
			Ellipsis: v.Ellipsis,
			Elt:      goExpr(v.Elt),
		}
	}
	log.Panicln("goExpr: unknown expr -", reflect.TypeOf(val))
	return nil
}

func goExprs(vals []gopast.Expr) []ast.Expr {
	n := len(vals)
	if n == 0 {
		return nil
	}
	ret := make([]ast.Expr, n)
	for i, v := range vals {
		ret[i] = goExpr(v)
	}
	return ret
}

// ----------------------------------------------------------------------------

func goFuncType(v *gopast.FuncType) *ast.FuncType {
	return &ast.FuncType{
		Func:    v.Func,
		Params:  goFieldList(v.Params),
		Results: goFieldList(v.Results),
	}
}

func goType(v gopast.Expr) ast.Expr {
	return goExpr(v)
}

func goBasicLit(v *gopast.BasicLit) *ast.BasicLit {
	if v == nil {
		return nil
	}
	return &ast.BasicLit{
		ValuePos: v.ValuePos,
		Kind:     token.Token(v.Kind),
		Value:    v.Value,
	}
}

func goIdent(v *gopast.Ident) *ast.Ident {
	if v == nil {
		return nil
	}
	return &ast.Ident{
		NamePos: v.NamePos,
		Name:    v.Name,
	}
}

func goIdents(names []*gopast.Ident) []*ast.Ident {
	ret := make([]*ast.Ident, len(names))
	for i, v := range names {
		ret[i] = goIdent(v)
	}
	return ret
}

// ----------------------------------------------------------------------------

func goField(v *gopast.Field) *ast.Field {
	return &ast.Field{
		Names: goIdents(v.Names),
		Type:  goType(v.Type),
		Tag:   goBasicLit(v.Tag),
	}
}

func goFieldList(v *gopast.FieldList) *ast.FieldList {
	if v == nil {
		return nil
	}
	list := make([]*ast.Field, len(v.List))
	for i, item := range v.List {
		list[i] = goField(item)
	}
	return &ast.FieldList{Opening: v.Opening, List: list, Closing: v.Closing}
}

func goFuncDecl(v *gopast.FuncDecl) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: goFieldList(v.Recv),
		Name: goIdent(v.Name),
		Type: goFuncType(v.Type),
		Body: &ast.BlockStmt{}, // ignore function body
	}
}

// ----------------------------------------------------------------------------

func goImportSpec(spec *gopast.ImportSpec) *ast.ImportSpec {
	return &ast.ImportSpec{
		Name:   goIdent(spec.Name),
		Path:   goBasicLit(spec.Path),
		EndPos: spec.EndPos,
	}
}

func goTypeSpec(spec *gopast.TypeSpec) *ast.TypeSpec {
	return &ast.TypeSpec{
		Name:   goIdent(spec.Name),
		Assign: spec.Assign,
		Type:   goType(spec.Type),
	}
}

func goValueSpec(spec *gopast.ValueSpec) *ast.ValueSpec {
	return &ast.ValueSpec{
		Names:  goIdents(spec.Names),
		Type:   goType(spec.Type),
		Values: goExprs(spec.Values),
	}
}

func goGenDecl(v *gopast.GenDecl) *ast.GenDecl {
	specs := make([]ast.Spec, len(v.Specs))
	for i, spec := range v.Specs {
		switch v.Tok {
		case goptoken.IMPORT:
			specs[i] = goImportSpec(spec.(*gopast.ImportSpec))
		case goptoken.TYPE:
			specs[i] = goTypeSpec(spec.(*gopast.TypeSpec))
		case goptoken.VAR, goptoken.CONST:
			specs[i] = goValueSpec(spec.(*gopast.ValueSpec))
		default:
			log.Panicln("goGenDecl: unknown spec -", v.Tok)
		}
	}
	return &ast.GenDecl{
		TokPos: v.TokPos,
		Tok:    token.Token(v.Tok),
		Lparen: v.Lparen,
		Specs:  specs,
		Rparen: v.Rparen,
	}
}

// ----------------------------------------------------------------------------

func goDecl(decl gopast.Decl) ast.Decl {
	switch v := decl.(type) {
	case *gopast.GenDecl:
		return goGenDecl(v)
	case *gopast.FuncDecl:
		return goFuncDecl(v)
	}
	log.Panicln("goDecl: unknown decl -", reflect.TypeOf(decl))
	return nil
}

func goDecls(decls []gopast.Decl) []ast.Decl {
	ret := make([]ast.Decl, len(decls))
	for i, decl := range decls {
		ret[i] = goDecl(decl)
	}
	return ret
}

// ----------------------------------------------------------------------------

const (
	KeepFuncBody = 1 << iota
)

func ASTFile(f *gopast.File, mode int) *ast.File {
	if (mode & KeepFuncBody) != 0 {
		log.Panicln("ASTFile: doesn't support keeping func body now")
	}
	return &ast.File{
		Package: f.Package,
		Name:    goIdent(f.Name),
		Decls:   goDecls(f.Decls),
	}
}

// ----------------------------------------------------------------------------
