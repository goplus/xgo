/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package ast

import (
	"github.com/goplus/xgo/tpl/token"
)

// -----------------------------------------------------------------------------

// Node: File, Decl, Expr
type Node interface {
	Pos() token.Pos
	End() token.Pos
}

// Decl: Rule
type Decl interface {
	Node
	declNode()
}

// Expr: Ident, BasicLit, Choice, Sequence, UnaryExpr, BinaryExpr
type Expr interface {
	Node
	exprNode()
}

// -----------------------------------------------------------------------------

// File: *Decl
type File struct {
	Decls     []Decl
	FileStart token.Pos
}

func (p *File) Pos() token.Pos {
	if n := len(p.Decls); n > 0 {
		return p.Decls[0].Pos()
	}
	return p.FileStart
}

func (p *File) End() token.Pos {
	if n := len(p.Decls); n > 0 {
		return p.Decls[n-1].End()
	}
	return p.FileStart
}

// -----------------------------------------------------------------------------

// Rule:
//
//	IDENT '=' Expr
//	IDENT '=' Expr => { ... }
type Rule struct {
	Name    *Ident
	TokPos  token.Pos // position of '='
	Expr    Expr
	RetProc Node // => { ... } (see xgo/ast.LambdaExpr2) or nil
}

// IsList reports whether the rule is a list rule.
func (p *Rule) IsList() bool {
	switch e := p.Expr.(type) {
	case *Sequence:
		return true
	case *UnaryExpr:
		switch e.Op {
		case token.MUL, token.ADD: // *R, +R
			return true
		}
	case *BinaryExpr:
		switch e.Op {
		case token.REM, token.INC: // R1 % R2, R1 ++ R2
			return true
		}
	}
	return false
}

func (p *Rule) Pos() token.Pos { return p.Name.Pos() }
func (p *Rule) End() token.Pos {
	if p.RetProc != nil {
		return p.RetProc.End()
	}
	return p.Expr.End()
}

func (p *Rule) declNode() {}

// -----------------------------------------------------------------------------

// Ident: IDENT
type Ident struct {
	NamePos token.Pos // identifier position
	Name    string    // identifier name
}

func (p *Ident) Pos() token.Pos { return p.NamePos }
func (p *Ident) End() token.Pos { return p.NamePos + token.Pos(len(p.Name)) }
func (p *Ident) exprNode()      {}

// -----------------------------------------------------------------------------

// BasicLit: STRING | CHAR
type BasicLit struct {
	ValuePos token.Pos   // literal position
	Kind     token.Token // token.STRING or token.CHAR
	Value    string
}

func (p *BasicLit) Pos() token.Pos { return p.ValuePos }
func (p *BasicLit) End() token.Pos { return p.ValuePos + token.Pos(len(p.Value)) }
func (p *BasicLit) exprNode()      {}

// -----------------------------------------------------------------------------

// Choice: R1 | R2 | ... | Rn
type Choice struct {
	Options []Expr // multiple options
}

func (p *Choice) Pos() token.Pos { return p.Options[0].Pos() }
func (p *Choice) End() token.Pos { return p.Options[len(p.Options)-1].End() }
func (p *Choice) exprNode()      {}

// -----------------------------------------------------------------------------

// Sequence: R1 R2 ... Rn
type Sequence struct {
	Items []Expr // multiple items
}

func (p *Sequence) Pos() token.Pos { return p.Items[0].Pos() }
func (p *Sequence) End() token.Pos { return p.Items[len(p.Items)-1].End() }
func (p *Sequence) exprNode()      {}

// -----------------------------------------------------------------------------

// UnaryExpr: *R, +R or ?R
type UnaryExpr struct {
	OpPos token.Pos   // operator position
	Op    token.Token // operator: token.MUL, token.ADD or token.QUESTION
	X     Expr        // operand
}

func (p *UnaryExpr) Pos() token.Pos { return p.OpPos }
func (p *UnaryExpr) End() token.Pos { return p.X.End() }
func (p *UnaryExpr) exprNode()      {}

// -----------------------------------------------------------------------------

// BinaryExpr: R1 % R2, R1 ++ R2
type BinaryExpr struct {
	X     Expr        // left operand
	OpPos token.Pos   // operator position
	Op    token.Token // operator: token.REM (list operator), token.INC (adjoin operator)
	Y     Expr        // right operand
}

func (p *BinaryExpr) Pos() token.Pos { return p.X.Pos() }
func (p *BinaryExpr) End() token.Pos { return p.Y.End() }
func (p *BinaryExpr) exprNode()      {}

// -----------------------------------------------------------------------------
