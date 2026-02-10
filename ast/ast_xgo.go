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

package ast

import (
	"github.com/goplus/xgo/token"
)

// -----------------------------------------------------------------------------

// OverloadFuncDecl node represents an overload function declaration:
//
// `func name = (overloadFuncs)`
// `func (T).nameOrOp = (overloadFuncs)`
//
// here overloadFunc represents
//
// `func(params) results {...}`
// `funcName`
// `(*T).methodName`
type OverloadFuncDecl struct {
	Doc      *CommentGroup // associated documentation; or nil
	Func     token.Pos     // position of "func" keyword
	Recv     *FieldList    // receiver (methods); or nil (functions)
	Name     *Ident        // function/method name
	Assign   token.Pos     // position of token "="
	Lparen   token.Pos     // position of "("
	Funcs    []Expr        // overload functions. here `Expr` can be *FuncLit, *Ident or *SelectorExpr
	Rparen   token.Pos     // position of ")"
	Operator bool          // is operator or not
	IsClass  bool          // recv set by class
}

// Pos - position of first character belonging to the node.
func (p *OverloadFuncDecl) Pos() token.Pos {
	return p.Func
}

// End - position of first character immediately after the node.
func (p *OverloadFuncDecl) End() token.Pos {
	return p.Rparen + 1
}

func (*OverloadFuncDecl) declNode() {}

// -----------------------------------------------------------------------------

// A CallExpr node represents an expression followed by an argument list.
// The argument list may include positional arguments (Args) and/or
// keyword arguments (Kwargs).
type CallExpr struct {
	Fun        Expr         // function expression
	Lparen     token.Pos    // position of "("
	Args       []Expr       // positional arguments; or nil
	Ellipsis   token.Pos    // position of "..." (token.NoPos if there is no "...")
	Kwargs     []*KwargExpr // keyword arguments; or nil
	Rparen     token.Pos    // position of ")"
	NoParenEnd token.Pos
}

// Pos returns position of first character belonging to the node.
func (x *CallExpr) Pos() token.Pos { return x.Fun.Pos() }

// End returns position of first character immediately after the node.
func (x *CallExpr) End() token.Pos {
	if x.NoParenEnd != token.NoPos {
		return x.NoParenEnd
	}
	return x.Rparen + 1
}

func (*CallExpr) exprNode() {}

// IsCommand returns if a CallExpr is a command style CallExpr or not.
func (x *CallExpr) IsCommand() bool {
	return x.NoParenEnd != token.NoPos
}

// A KwargExpr node represents a keyword argument expression.
type KwargExpr struct {
	Name  *Ident // argument name
	Value Expr   // argument value
}

func (p *KwargExpr) Pos() token.Pos {
	return p.Name.Pos()
}

func (p *KwargExpr) End() token.Pos {
	return p.Value.End()
}

func (p *KwargExpr) exprNode() {}

// -----------------------------------------------------------------------------

// A DomainTextLit node represents a domain-specific text literal.
// https://github.com/goplus/xgo/issues/2143
//
//	domainTag`...`
//	domainTag`> arg1, arg2, ...
//	  ...
//	`
type DomainTextLit struct {
	Domain   *Ident    // domain name
	ValuePos token.Pos // literal position
	Value    string    // literal string; e.g. `\m\n\o`
	Extra    any       // *DomainTextLitEx or *xgo/tpl/ast.File, optional
}

// DomainTextLitEx represents extra information for domain text literal.
type DomainTextLitEx struct {
	Args   []Expr    // domain text arguments; or nil
	RawPos token.Pos // position of the first character of the raw string
	Raw    string    // raw string without backquote
}

// Pos returns position of first character belonging to the node.
func (x *DomainTextLit) Pos() token.Pos { return x.Domain.NamePos }

// End returns position of first character immediately after the node.
func (x *DomainTextLit) End() token.Pos { return token.Pos(int(x.ValuePos) + len(x.Value)) }

func (*DomainTextLit) exprNode() {}

// -----------------------------------------------------------------------------

// A BasicLit node represents a literal of basic type.
type BasicLit struct {
	ValuePos token.Pos    // literal position
	Kind     token.Token  // token.INT, token.FLOAT, token.IMAG, token.RAT, token.CHAR, token.STRING, token.CSTRING
	Value    string       // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 3r, 'a', '\x7f', "foo" or `\m\n\o`
	Extra    *StringLitEx // optional (only available when Kind == token.STRING)
}

type StringLitEx struct {
	Parts []any // can be (val string) or (xval Expr)
}

// NextPartPos - position of first character of next part.
// pos - position of this part (not including quote character).
func NextPartPos(pos token.Pos, part any) (nextPos token.Pos) {
	switch v := part.(type) {
	case string: // normal string literal or end with "$$"
		return pos + token.Pos(len(v))
	case Expr:
		return v.End()
	}
	panic("NextPartPos: unexpected parameters")
}

// Pos returns position of first character belonging to the node.
func (x *BasicLit) Pos() token.Pos { return x.ValuePos }

// End returns position of first character immediately after the node.
func (x *BasicLit) End() token.Pos { return token.Pos(int(x.ValuePos) + len(x.Value)) }

func (*BasicLit) exprNode() {}

// -----------------------------------------------------------------------------

// A NumberUnitLit node represents a number with unit.
type NumberUnitLit struct {
	ValuePos token.Pos   // literal position
	Kind     token.Token // token.INT or token.FLOAT
	Value    string      // literal string of the number; e.g. 42, 0x7f, 3.14, 1e-9
	Unit     string      // unit string of the number; e.g. "px", "em", "rem"
}

func (*NumberUnitLit) exprNode() {}

func (x *NumberUnitLit) Pos() token.Pos {
	return x.ValuePos
}

func (x *NumberUnitLit) End() token.Pos {
	return token.Pos(int(x.ValuePos) + len(x.Value) + len(x.Unit))
}

// -----------------------------------------------------------------------------

// A EnvExpr node represents a ${name} expression.
type EnvExpr struct {
	TokPos token.Pos // position of "$"
	Lbrace token.Pos // position of "{"
	Name   *Ident    // name
	Rbrace token.Pos // position of "}"
}

// Pos - position of first character belonging to the node.
func (p *EnvExpr) Pos() token.Pos {
	return p.TokPos
}

// End - position of first character immediately after the node.
func (p *EnvExpr) End() token.Pos {
	if p.Rbrace != token.NoPos {
		return p.Rbrace
	}
	return p.Name.End()
}

// HasBrace checks is this EnvExpr ${name} or $name.
func (p *EnvExpr) HasBrace() bool {
	return p.Rbrace != token.NoPos
}

func (*EnvExpr) exprNode() {}

// -----------------------------------------------------------------------------

// AnySelectorExpr represents `X.**.Sel` expression, which selects any field
// named Sel in the nested object X.
// Sel may not be a simple identifier. For example:
//   - x.**.field
//   - x.**."field-name"
type AnySelectorExpr struct {
	X      Expr      // expression
	TokPos token.Pos // position of "**"
	Sel    *Ident    // field selector (it may not be a simple identifier)
}

// Pos - position of first character belonging to the node.
func (p *AnySelectorExpr) Pos() token.Pos {
	return p.X.Pos()
}

// End - position of first character immediately after the node.
func (p *AnySelectorExpr) End() token.Pos {
	return p.Sel.End()
}

func (*AnySelectorExpr) exprNode() {}

// -----------------------------------------------------------------------------

// A CondExpr node represents a conditional expression: `expr @ cond`.
type CondExpr struct {
	X     Expr      // expression
	OpPos token.Pos // position of "@"
	Cond  Expr      // condition expression
}

// Pos - position of first character belonging to the node.
func (p *CondExpr) Pos() token.Pos {
	return p.X.Pos()
}

// End - position of first character immediately after the node.
func (p *CondExpr) End() token.Pos {
	return p.Cond.End()
}

func (*CondExpr) exprNode() {}

// -----------------------------------------------------------------------------

// A SliceLit node represents a slice literal.
type SliceLit struct {
	Lbrack     token.Pos // position of "["
	Elts       []Expr    // list of slice elements; or nil
	Rbrack     token.Pos // position of "]"
	Incomplete bool      // true if (source) expressions are missing in the Elts list
}

// Pos - position of first character belonging to the node.
func (p *SliceLit) Pos() token.Pos {
	return p.Lbrack
}

// End - position of first character immediately after the node.
func (p *SliceLit) End() token.Pos {
	return p.Rbrack + 1
}

func (*SliceLit) exprNode() {}

// -----------------------------------------------------------------------------

// A TupleLit node represents a tuple literal.
type TupleLit struct {
	Lparen   token.Pos // position of "("
	Elts     []Expr    // list of tuple elements; or nil
	Ellipsis token.Pos // position of "..." (token.NoPos if there is no "...")
	Rparen   token.Pos // position of ")"
}

// Pos - position of first character belonging to the node.
func (p *TupleLit) Pos() token.Pos {
	return p.Lparen
}

// End - position of first character immediately after the node.
func (p *TupleLit) End() token.Pos {
	return p.Rparen + 1
}

func (*TupleLit) exprNode() {}

// -----------------------------------------------------------------------------

// A MatrixLit node represents a matrix literal.
type MatrixLit struct {
	Lbrack     token.Pos // position of "["
	Elts       [][]Expr  // list of matrix elements
	Rbrack     token.Pos // position of "]"
	Incomplete bool      // true if (source) expressions are missing in the Elts list
}

// Pos - position of first character belonging to the node.
func (p *MatrixLit) Pos() token.Pos {
	return p.Lbrack
}

// End - position of first character immediately after the node.
func (p *MatrixLit) End() token.Pos {
	return p.Rbrack + 1
}

func (*MatrixLit) exprNode() {}

// -----------------------------------------------------------------------------

// A ElemEllipsis node represents a matrix row elements.
type ElemEllipsis struct {
	Elt      Expr      // ellipsis element
	Ellipsis token.Pos // position of "..."
}

// Pos - position of first character belonging to the node.
func (p *ElemEllipsis) Pos() token.Pos {
	return p.Elt.Pos()
}

// End - position of first character immediately after the node.
func (p *ElemEllipsis) End() token.Pos {
	return p.Ellipsis + 3
}

func (*ElemEllipsis) exprNode() {}

// -----------------------------------------------------------------------------

// ErrWrapExpr represents `expr!`, `expr?` or `expr?: defaultValue`.
type ErrWrapExpr struct {
	X       Expr
	Tok     token.Token // ! or ?
	TokPos  token.Pos
	Default Expr // can be nil
}

// Pos - position of first character belonging to the node.
func (p *ErrWrapExpr) Pos() token.Pos {
	return p.X.Pos()
}

// End - position of first character immediately after the node.
func (p *ErrWrapExpr) End() token.Pos {
	if p.Default != nil {
		return p.Default.End()
	}
	return p.TokPos + 1
}

func (*ErrWrapExpr) exprNode() {}

// -----------------------------------------------------------------------------

// LambdaExpr represents one of the following expressions:
//
//	`(x, y, ...) => exprOrExprTuple`
//	`x => exprOrExprTuple`
//	`=> exprOrExprTuple`
//
// here exprOrExprTuple represents
//
//	`expr`
//	`(expr1, expr2, ...)`
type LambdaExpr struct {
	First       token.Pos
	Lhs         []*Ident
	Rarrow      token.Pos
	Rhs         []Expr
	Last        token.Pos
	LhsHasParen bool
	RhsHasParen bool
}

// LambdaExpr2 represents one of the following expressions:
//
//	`(x, y, ...) => { ... }`
//	`x => { ... }`
//	`=> { ... }`
type LambdaExpr2 struct {
	First       token.Pos
	Lhs         []*Ident
	Rarrow      token.Pos
	Body        *BlockStmt
	LhsHasParen bool
}

// Pos - position of first character belonging to the node.
func (p *LambdaExpr) Pos() token.Pos {
	return p.First
}

// End - position of first character immediately after the node.
func (p *LambdaExpr) End() token.Pos {
	return p.Last
}

// Pos - position of first character belonging to the node.
func (p *LambdaExpr2) Pos() token.Pos {
	return p.First
}

// End - position of first character immediately after the node.
func (p *LambdaExpr2) End() token.Pos {
	if p.Body == nil {
		return p.Rarrow + 2
	}
	return p.Body.End()
}

func (*LambdaExpr) exprNode()  {}
func (*LambdaExpr2) exprNode() {}

// -----------------------------------------------------------------------------

// A RangeExpr node represents a range expression.
type RangeExpr struct {
	First  Expr      // start of composite elements; or nil
	To     token.Pos // position of ":"
	Last   Expr      // end of composite elements
	Colon2 token.Pos // position of ":" or token.NoPos
	Expr3  Expr      // step (or max) of composite elements; or nil
}

// Pos - position of first character belonging to the node.
func (p *RangeExpr) Pos() token.Pos {
	if p.First != nil {
		return p.First.Pos()
	}
	return p.To
}

// End - position of first character immediately after the node.
func (p *RangeExpr) End() token.Pos {
	if p.Expr3 != nil {
		return p.Expr3.End()
	}
	if p.Colon2 != token.NoPos {
		return p.Colon2 + 1
	}
	if p.Last != nil {
		return p.Last.End()
	}
	return p.To + 1
}

func (*RangeExpr) exprNode() {}

// -----------------------------------------------------------------------------

// ForPhrase represents `for k, v in container if init; cond` phrase.
type ForPhrase struct {
	For        token.Pos // position of "for" keyword
	Key, Value *Ident    // Key may be nil
	TokPos     token.Pos // position of "in" operator
	X          Expr      // value to range over
	IfPos      token.Pos // position of if or comma; or NoPos
	Init       Stmt      // initialization statement; or nil
	Cond       Expr      // value filter, can be nil
}

// Pos returns position of first character belonging to the node.
func (p *ForPhrase) Pos() token.Pos { return p.For }

// End returns position of first character immediately after the node.
func (p *ForPhrase) End() token.Pos {
	if p.Cond != nil {
		return p.Cond.End()
	}
	return p.X.End()
}

func (p *ForPhrase) exprNode() {}

// ComprehensionExpr represents one of the following expressions:
//
//	`[vexpr for k1, v1 in container1, cond1 ...]` or
//	`{vexpr for k1, v1 in container1, cond1 ...}` or
//	`{kexpr: vexpr for k1, v1 in container1, cond1 ...}` or
//	`{for k1, v1 in container1, cond1 ...}`
type ComprehensionExpr struct {
	Lpos token.Pos   // position of "[" or "{"
	Tok  token.Token // token.LBRACK '[' or token.LBRACE '{'
	Elt  Expr        // *KeyValueExpr or Expr or nil
	Fors []*ForPhrase
	Rpos token.Pos // position of "]" or "}"
}

// Pos - position of first character belonging to the node.
func (p *ComprehensionExpr) Pos() token.Pos {
	return p.Lpos
}

// End - position of first character immediately after the node.
func (p *ComprehensionExpr) End() token.Pos {
	return p.Rpos + 1
}

func (*ComprehensionExpr) exprNode() {}

// -----------------------------------------------------------------------------

// A ForPhraseStmt represents a for statement with a for..in clause.
type ForPhraseStmt struct {
	*ForPhrase
	Body *BlockStmt
}

// Pos - position of first character belonging to the node.
func (p *ForPhraseStmt) Pos() token.Pos {
	return p.For
}

// End - position of first character immediately after the node.
func (p *ForPhraseStmt) End() token.Pos {
	return p.Body.End()
}

func (*ForPhraseStmt) stmtNode() {}

// -----------------------------------------------------------------------------

// A SendStmt node represents a send statement.
type SendStmt struct {
	Chan     Expr
	Arrow    token.Pos // position of "<-"
	Values   []Expr    // len(Values) must > 0
	Ellipsis token.Pos // position of "..."
}

// End returns position of first character immediately after the node.
func (s *SendStmt) End() token.Pos {
	if s.Ellipsis != token.NoPos {
		return s.Ellipsis + 3
	}
	vals := s.Values
	return vals[len(vals)-1].End()
}

// -----------------------------------------------------------------------------

// A File node represents an XGo source file.
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
//
// For correct printing of source code containing comments (using packages
// go/format and go/printer), special care must be taken to update comments
// when a File's syntax tree is modified: For printing, comments are interspersed
// between tokens based on their position. If syntax tree nodes are
// removed or moved, relevant comments in their vicinity must also be removed
// (from the File.Comments list) or moved accordingly (by updating their
// positions). A CommentMap may be used to facilitate some of these operations.
//
// Whether and how a comment is associated with a node depends on the
// interpretation of the syntax tree by the manipulating program: Except for Doc
// and Comment comments directly associated with nodes, the remaining comments
// are "free-floating" (see also issues #18593, #20744).
type File struct {
	Doc     *CommentGroup // associated documentation; or nil
	Package token.Pos     // position of "package" keyword; or NoPos
	Name    *Ident        // package name
	Decls   []Decl        // top-level declarations; or nil

	Imports     []*ImportSpec   // imports in this file
	Comments    []*CommentGroup // list of all comments in the source file
	Code        []byte
	ShadowEntry *FuncDecl // indicate the module entry point.
	NoPkgDecl   bool      // no `package xxx` declaration
	IsClass     bool      // is a classfile (including normal .gox file)
	IsProj      bool      // is a project classfile
	IsNormalGox bool      // is a normal .gox file
}

// There is no entrypoint func to indicate the module entry point.
func (f *File) HasShadowEntry() bool {
	return f.ShadowEntry != nil
}

// HasPkgDecl checks if `package xxx` exists or not.
func (f *File) HasPkgDecl() bool {
	return f.Package != token.NoPos
}

// ClassFieldsDecl returns the class fields declaration.
func (f *File) ClassFieldsDecl() *GenDecl {
	if f.IsClass {
		for _, decl := range f.Decls {
			if g, ok := decl.(*GenDecl); ok {
				if g.Tok == token.VAR {
					return g
				}
				continue
			}
			break
		}
	}
	return nil
}

// Pos returns position of first character belonging to the node.
func (f *File) Pos() token.Pos {
	if f.Package != token.NoPos {
		return f.Package
	}
	// if no package clause, name records the position of the first token in the file
	return f.Name.NamePos
}

// End returns position of first character immediately after the node.
func (f *File) End() token.Pos {
	if f.ShadowEntry != nil { // has shadow entry
		return f.ShadowEntry.End()
	}
	for n := len(f.Decls) - 1; n >= 0; n-- {
		d := f.Decls[n]
		if fn, ok := d.(*FuncDecl); ok && fn.Shadow {
			// skip shadow functions like Classfname (see cl.astFnClassfname)
			continue
		}
		return d.End()
	}
	if f.Package != token.NoPos { // has package clause
		return f.Name.End()
	}
	return f.Name.Pos()
}

// -----------------------------------------------------------------------------
