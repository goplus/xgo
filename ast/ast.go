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

// Package ast declares the types used to represent syntax trees for XGo
// packages.
package ast

import (
	"go/ast"

	"github.com/goplus/xgo/token"
)

// ----------------------------------------------------------------------------
// Interfaces
//
// There are 3 main classes of nodes: Expressions and type nodes,
// statement nodes, and declaration nodes. The node names usually
// match the corresponding Go spec production names to which they
// correspond. The node fields correspond to the individual parts
// of the respective productions.
//
// All nodes contain position information marking the beginning of
// the corresponding source text segment; it is accessible via the
// Pos accessor method. Nodes may contain additional position info
// for language constructs where comments may be found between parts
// of the construct (typically any larger, parenthesized subpart).
// That position information is needed to properly position comments
// when printing the construct.

// Node interface: all node types implement the Node interface.
type Node = ast.Node

// Expr interface: all expression nodes implement the Expr interface.
type Expr interface {
	Node
	exprNode()
}

// Stmt interface: all statement nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

// Decl interface: all declaration nodes implement the Decl interface.
type Decl interface {
	Node
	declNode()
}

// ----------------------------------------------------------------------------
// Comments

// A Comment node represents a single //-style 、#-style or /*-style comment.
type Comment = ast.Comment

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup = ast.CommentGroup

// ----------------------------------------------------------------------------
// Expressions and types

// A Field represents a Field declaration list in a struct type,
// a method list in an interface type, or a parameter/result declaration
// in a signature.
// Field.Names is nil for unnamed parameters (parameter lists which only contain types)
// and embedded struct fields. In the latter case, the field name is the type name.
type Field struct {
	Doc      *CommentGroup // associated documentation; or nil
	Names    []*Ident      // field/method/parameter names; or nil
	Type     Expr          // field/method/parameter type
	Tag      *BasicLit     // field tag; or nil
	Comment  *CommentGroup // line comments; or nil
	Optional bool          // true if this parameter is optional (T?)
}

// Pos returns position of first character belonging to the node.
func (f *Field) Pos() token.Pos {
	if len(f.Names) > 0 {
		return f.Names[0].Pos()
	}
	return f.Type.Pos()
}

// End returns position of first character immediately after the node.
func (f *Field) End() token.Pos {
	if f.Tag != nil {
		return f.Tag.End()
	}
	return f.Type.End()
}

// A FieldList represents a list of Fields, enclosed by parentheses or braces.
type FieldList struct {
	Opening token.Pos // position of opening parenthesis/brace, if any
	List    []*Field  // field list; or nil
	Closing token.Pos // position of closing parenthesis/brace, if any
}

// Pos returns position of first character belonging to the node.
func (f *FieldList) Pos() token.Pos {
	if f.Opening.IsValid() {
		return f.Opening
	}
	// the list should not be empty in this case;
	// be conservative and guard against bad ASTs
	if len(f.List) > 0 {
		return f.List[0].Pos()
	}
	return token.NoPos
}

// End returns position of first character immediately after the node.
func (f *FieldList) End() token.Pos {
	if f.Closing.IsValid() {
		return f.Closing + 1
	}
	// the list should not be empty in this case;
	// be conservative and guard against bad ASTs
	if n := len(f.List); n > 0 {
		return f.List[n-1].End()
	}
	return token.NoPos
}

// NumFields returns the number of parameters or struct fields represented by a FieldList.
func (f *FieldList) NumFields() int {
	n := 0
	if f != nil {
		for _, g := range f.List {
			m := len(g.Names)
			if m == 0 {
				m = 1
			}
			n += m
		}
	}
	return n
}

// An expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
type (
	// A BadExpr node is a placeholder for expressions containing
	// syntax errors for which no correct expression nodes can be
	// created.
	BadExpr struct {
		From, To token.Pos // position range of bad expression
	}

	// An Ident node represents an identifier.
	Ident struct {
		NamePos token.Pos // identifier position
		Name    string    // identifier name
		Obj     *Object   // denoted object; or nil
	}

	// An Ellipsis node stands for the "..." type in a
	// parameter list or the "..." length in an array type.
	Ellipsis struct {
		Ellipsis token.Pos // position of "..."
		Elt      Expr      // ellipsis element type (parameter lists only); or nil
	}

	// A FuncLit node represents a function literal.
	FuncLit struct {
		Type *FuncType  // function type
		Body *BlockStmt // function body
	}

	// A CompositeLit node represents a composite literal.
	CompositeLit struct {
		Type       Expr      // literal type; or nil
		Lbrace     token.Pos // position of "{"
		Elts       []Expr    // list of composite elements; or nil
		Rbrace     token.Pos // position of "}"
		Incomplete bool      // true if (source) expressions are missing in the Elts list
	}

	// A ParenExpr node represents a parenthesized expression.
	ParenExpr struct {
		Lparen token.Pos // position of "("
		X      Expr      // parenthesized expression
		Rparen token.Pos // position of ")"
	}

	// A SelectorExpr node represents an expression followed by a selector.
	SelectorExpr struct {
		X   Expr   // expression
		Sel *Ident // field selector
	}

	// An IndexExpr node represents an expression followed by an index.
	IndexExpr struct {
		X      Expr      // expression
		Lbrack token.Pos // position of "["
		Index  Expr      // index expression
		Rbrack token.Pos // position of "]"
	}

	// An IndexListExpr node represents an expression followed by multiple
	// indices.
	IndexListExpr struct {
		X       Expr      // expression
		Lbrack  token.Pos // position of "["
		Indices []Expr    // index expressions
		Rbrack  token.Pos // position of "]"
	}

	// A SliceExpr node represents an expression followed by slice indices.
	SliceExpr struct {
		X      Expr      // expression
		Lbrack token.Pos // position of "["
		Low    Expr      // begin of slice range; or nil
		High   Expr      // end of slice range; or nil
		Max    Expr      // maximum capacity of slice; or nil
		Slice3 bool      // true if 3-index slice (2 colons present)
		Rbrack token.Pos // position of "]"
	}

	// A TypeAssertExpr node represents an expression followed by a
	// type assertion.
	TypeAssertExpr struct {
		X      Expr      // expression
		Lparen token.Pos // position of "("
		Type   Expr      // asserted type; nil means type switch X.(type)
		Rparen token.Pos // position of ")"
	}

	// A StarExpr node represents an expression of the form "*" Expression.
	// Semantically it could be a unary "*" expression, or a pointer type.
	StarExpr struct {
		Star token.Pos // position of "*"
		X    Expr      // operand
	}

	// A UnaryExpr node represents a unary expression.
	// Unary "*" expressions are represented via StarExpr nodes.
	UnaryExpr struct {
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		X     Expr        // operand
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X     Expr        // left operand
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		Y     Expr        // right operand
	}

	// A KeyValueExpr node represents (key : value) pairs
	// in composite literals.
	KeyValueExpr struct {
		Key   Expr
		Colon token.Pos // position of ":"
		Value Expr
	}
)

// ChanDir is the direction of a channel type is indicated by a bit
// mask including one or both of the following constants.
type ChanDir int

const (
	// SEND - ChanDir
	SEND ChanDir = 1 << iota
	// RECV - ChanDir
	RECV
)

// A type is represented by a tree consisting of one
// or more of the following type-specific expression
// nodes.
type (
	// An ArrayType node represents an array or slice type.
	ArrayType struct {
		Lbrack token.Pos // position of "["
		Len    Expr      // Ellipsis node for [...]T array types, nil for slice types
		Elt    Expr      // element type
	}

	// A StructType node represents a struct type.
	StructType struct {
		Struct     token.Pos  // position of "struct" keyword
		Fields     *FieldList // list of field declarations
		Incomplete bool       // true if (source) fields are missing in the Fields list
	}

	// Pointer types are represented via StarExpr nodes.

	// A FuncType node represents a function type.
	FuncType struct {
		Func       token.Pos  // position of "func" keyword (token.NoPos if there is no "func")
		TypeParams *FieldList // type parameters; or nil
		Params     *FieldList // (incoming) parameters; non-nil
		Results    *FieldList // (outgoing) results; or nil
	}

	// An InterfaceType node represents an interface type.
	InterfaceType struct {
		Interface  token.Pos  // position of "interface" keyword
		Methods    *FieldList // list of methods
		Incomplete bool       // true if (source) methods are missing in the Methods list
	}

	// A MapType node represents a map type.
	MapType struct {
		Map   token.Pos // position of "map" keyword
		Key   Expr
		Value Expr
	}

	// A ChanType node represents a channel type.
	ChanType struct {
		Begin token.Pos // position of "chan" keyword or "<-" (whichever comes first)
		Arrow token.Pos // position of "<-" (token.NoPos if there is no "<-")
		Dir   ChanDir   // channel direction
		Value Expr      // value type
	}
)

// Pos and End implementations for expression/type nodes.

// Pos returns position of first character belonging to the node.
func (x *BadExpr) Pos() token.Pos { return x.From }

// Pos returns position of first character belonging to the node.
func (x *Ident) Pos() token.Pos { return x.NamePos }

// Pos returns position of first character belonging to the node.
func (x *Ellipsis) Pos() token.Pos { return x.Ellipsis }

// Pos returns position of first character belonging to the node.
func (x *FuncLit) Pos() token.Pos { return x.Type.Pos() }

// Pos returns position of first character belonging to the node.
func (x *CompositeLit) Pos() token.Pos {
	if x.Type != nil {
		return x.Type.Pos()
	}
	return x.Lbrace
}

// Pos returns position of first character belonging to the node.
func (x *ParenExpr) Pos() token.Pos { return x.Lparen }

// Pos returns position of first character belonging to the node.
func (x *SelectorExpr) Pos() token.Pos { return x.X.Pos() }

// Pos returns position of first character belonging to the node.
func (x *IndexExpr) Pos() token.Pos { return x.X.Pos() }

// Pos returns position of first character belonging to the node.
func (x *IndexListExpr) Pos() token.Pos { return x.X.Pos() }

// Pos returns position of first character belonging to the node.
func (x *SliceExpr) Pos() token.Pos { return x.X.Pos() }

// Pos returns position of first character belonging to the node.
func (x *TypeAssertExpr) Pos() token.Pos { return x.X.Pos() }

// Pos returns position of first character belonging to the node.
func (x *StarExpr) Pos() token.Pos { return x.Star }

// Pos returns position of first character belonging to the node.
func (x *UnaryExpr) Pos() token.Pos { return x.OpPos }

// Pos returns position of first character belonging to the node.
func (x *BinaryExpr) Pos() token.Pos { return x.X.Pos() }

// Pos returns position of first character belonging to the node.
func (x *KeyValueExpr) Pos() token.Pos { return x.Key.Pos() }

// Pos returns position of first character belonging to the node.
func (x *ArrayType) Pos() token.Pos { return x.Lbrack }

// Pos returns position of first character belonging to the node.
func (x *StructType) Pos() token.Pos { return x.Struct }

// Pos returns position of first character belonging to the node.
func (x *FuncType) Pos() token.Pos {
	if x.Func.IsValid() || x.Params == nil { // see issue 3870
		return x.Func
	}
	return x.Params.Pos() // interface method declarations have no "func" keyword
}

// Pos returns position of first character belonging to the node.
func (x *InterfaceType) Pos() token.Pos { return x.Interface }

// Pos returns position of first character belonging to the node.
func (x *MapType) Pos() token.Pos { return x.Map }

// Pos returns position of first character belonging to the node.
func (x *ChanType) Pos() token.Pos { return x.Begin }

// End returns position of first character immediately after the node.
func (x *BadExpr) End() token.Pos { return x.To }

// End returns position of first character immediately after the node.
func (x *Ident) End() token.Pos {
	if x.Implicit() { // implicitly declared
		return x.NamePos
	}
	return x.NamePos + token.Pos(len(x.Name))
}

// Implicit reports whether the identifier was implicitly declared
func (x *Ident) Implicit() bool {
	o := x.Obj
	return o != nil && o.Kind >= implicitBase
}

// End returns position of first character immediately after the node.
func (x *Ellipsis) End() token.Pos {
	if x.Elt != nil {
		return x.Elt.End()
	}
	return x.Ellipsis + 3 // len("...")
}

// End returns position of first character immediately after the node.
func (x *FuncLit) End() token.Pos { return x.Body.End() }

// End returns position of first character immediately after the node.
func (x *CompositeLit) End() token.Pos { return x.Rbrace + 1 }

// End returns position of first character immediately after the node.
func (x *ParenExpr) End() token.Pos { return x.Rparen + 1 }

// End returns position of first character immediately after the node.
func (x *SelectorExpr) End() token.Pos { return x.Sel.End() }

// End returns position of first character immediately after the node.
func (x *IndexExpr) End() token.Pos { return x.Rbrack + 1 }

// End returns position of first character immediately after the node.
func (x *IndexListExpr) End() token.Pos { return x.Rbrack + 1 }

// End returns position of first character immediately after the node.
func (x *SliceExpr) End() token.Pos { return x.Rbrack + 1 }

// End returns position of first character immediately after the node.
func (x *TypeAssertExpr) End() token.Pos { return x.Rparen + 1 }

// End returns position of first character immediately after the node.
func (x *StarExpr) End() token.Pos { return x.X.End() }

// End returns position of first character immediately after the node.
func (x *UnaryExpr) End() token.Pos { return x.X.End() }

// End returns position of first character immediately after the node.
func (x *BinaryExpr) End() token.Pos { return x.Y.End() }

// End returns position of first character immediately after the node.
func (x *KeyValueExpr) End() token.Pos { return x.Value.End() }

// End returns position of first character immediately after the node.
func (x *ArrayType) End() token.Pos { return x.Elt.End() }

// End returns position of first character immediately after the node.
func (x *StructType) End() token.Pos { return x.Fields.End() }

// End returns position of first character immediately after the node.
func (x *FuncType) End() token.Pos {
	if x.Results != nil {
		return x.Results.End()
	}
	return x.Params.End()
}

// End returns position of first character immediately after the node.
func (x *InterfaceType) End() token.Pos { return x.Methods.End() }

// End returns position of first character immediately after the node.
func (x *MapType) End() token.Pos { return x.Value.End() }

// End returns position of first character immediately after the node.
func (x *ChanType) End() token.Pos { return x.Value.End() }

// exprNode() ensures that only expression/type nodes can be
// assigned to an Expr.
func (*BadExpr) exprNode()        {}
func (*Ident) exprNode()          {}
func (*Ellipsis) exprNode()       {}
func (*FuncLit) exprNode()        {}
func (*CompositeLit) exprNode()   {}
func (*ParenExpr) exprNode()      {}
func (*SelectorExpr) exprNode()   {}
func (*IndexExpr) exprNode()      {}
func (*IndexListExpr) exprNode()  {}
func (*SliceExpr) exprNode()      {}
func (*TypeAssertExpr) exprNode() {}
func (*StarExpr) exprNode()       {}
func (*UnaryExpr) exprNode()      {}
func (*BinaryExpr) exprNode()     {}
func (*KeyValueExpr) exprNode()   {}

func (*ArrayType) exprNode()     {}
func (*StructType) exprNode()    {}
func (*FuncType) exprNode()      {}
func (*InterfaceType) exprNode() {}
func (*MapType) exprNode()       {}
func (*ChanType) exprNode()      {}

// ----------------------------------------------------------------------------
// Convenience functions for Idents

// NewIdent creates a new Ident without position.
// Useful for ASTs generated by code other than the XGo parser.
func NewIdent(name string) *Ident { return &Ident{token.NoPos, name, nil} }

// NewIdentEx creates a new Ident with position and with the given kind.
func NewIdentEx(pos token.Pos, name string, kind ObjKind) *Ident {
	return &Ident{pos, name, NewObj(kind, name)}
}

// IsExported reports whether name starts with an upper-case letter.
func IsExported(name string) bool { return token.IsExported(name) }

// IsExported reports whether id starts with an upper-case letter.
func (x *Ident) IsExported() bool { return token.IsExported(x.Name) }

func (x *Ident) String() string {
	if x != nil {
		return x.Name
	}
	return "<nil>"
}

// ----------------------------------------------------------------------------
// Statements

// A statement is represented by a tree consisting of one
// or more of the following concrete statement nodes.
type (
	// A BadStmt node is a placeholder for statements containing
	// syntax errors for which no correct statement nodes can be
	// created.
	//
	BadStmt struct {
		From, To token.Pos // position range of bad statement
	}

	// A DeclStmt node represents a declaration in a statement list.
	DeclStmt struct {
		Decl Decl // *GenDecl with CONST, TYPE, or VAR token
	}

	// An EmptyStmt node represents an empty statement.
	// The "position" of the empty statement is the position
	// of the immediately following (explicit or implicit) semicolon.
	//
	EmptyStmt struct {
		Semicolon token.Pos // position of following ";"
		Implicit  bool      // if set, ";" was omitted in the source
	}

	// A LabeledStmt node represents a labeled statement.
	LabeledStmt struct {
		Label *Ident
		Colon token.Pos // position of ":"
		Stmt  Stmt
	}

	// An ExprStmt node represents a (stand-alone) expression
	// in a statement list.
	//
	ExprStmt struct {
		X Expr // expression
	}

	// An IncDecStmt node represents an increment or decrement statement.
	IncDecStmt struct {
		X      Expr
		TokPos token.Pos   // position of Tok
		Tok    token.Token // INC or DEC
	}

	// An AssignStmt node represents an assignment or
	// a short variable declaration.
	//
	AssignStmt struct {
		Lhs    []Expr      // left hand side expressions
		TokPos token.Pos   // position of Tok
		Tok    token.Token // assignment token, DEFINE
		Rhs    []Expr      // right hand side expressions
	}

	// A GoStmt node represents a go statement.
	GoStmt struct {
		Go   token.Pos // position of "go" keyword
		Call *CallExpr
	}

	// A DeferStmt node represents a defer statement.
	DeferStmt struct {
		Defer token.Pos // position of "defer" keyword
		Call  *CallExpr
	}

	// A ReturnStmt node represents a return statement.
	ReturnStmt struct {
		Return  token.Pos // position of "return" keyword
		Results []Expr    // result expressions; or nil
	}

	// A BranchStmt node represents a break, continue, goto,
	// or fallthrough statement.
	//
	BranchStmt struct {
		TokPos token.Pos   // position of Tok
		Tok    token.Token // keyword token (BREAK, CONTINUE, GOTO, FALLTHROUGH)
		Label  *Ident      // label name; or nil
	}

	// A BlockStmt node represents a braced statement list.
	BlockStmt struct {
		Lbrace token.Pos // position of "{"
		List   []Stmt
		Rbrace token.Pos // position of "}", if any (may be absent due to syntax error)
	}

	// An IfStmt node represents an if statement.
	IfStmt struct {
		If   token.Pos // position of "if" keyword
		Init Stmt      // initialization statement; or nil
		Cond Expr      // condition
		Body *BlockStmt
		Else Stmt // else branch; or nil
	}

	// A CaseClause represents a case of an expression or type switch statement.
	CaseClause struct {
		Case  token.Pos // position of "case" or "default" keyword
		List  []Expr    // list of expressions or types; nil means default case
		Colon token.Pos // position of ":"
		Body  []Stmt    // statement list; or nil
	}

	// A SwitchStmt node represents an expression switch statement.
	SwitchStmt struct {
		Switch token.Pos  // position of "switch" keyword
		Init   Stmt       // initialization statement; or nil
		Tag    Expr       // tag expression; or nil
		Body   *BlockStmt // CaseClauses only
	}

	// A TypeSwitchStmt node represents a type switch statement.
	TypeSwitchStmt struct {
		Switch token.Pos  // position of "switch" keyword
		Init   Stmt       // initialization statement; or nil
		Assign Stmt       // x := y.(type) or y.(type)
		Body   *BlockStmt // CaseClauses only
	}

	// A CommClause node represents a case of a select statement.
	CommClause struct {
		Case  token.Pos // position of "case" or "default" keyword
		Comm  Stmt      // send or receive statement; nil means default case
		Colon token.Pos // position of ":"
		Body  []Stmt    // statement list; or nil
	}

	// A SelectStmt node represents a select statement.
	SelectStmt struct {
		Select token.Pos  // position of "select" keyword
		Body   *BlockStmt // CommClauses only
	}

	// A ForStmt represents a `for init; cond; post { ... }` statement.
	ForStmt struct {
		For  token.Pos // position of "for" keyword
		Init Stmt      // initialization statement; or nil
		Cond Expr      // condition; or nil
		Post Stmt      // post iteration statement; or nil
		Body *BlockStmt
	}

	// A RangeStmt represents a for statement with a range clause.
	RangeStmt struct {
		For        token.Pos   // position of "for" keyword
		Key, Value Expr        // Key, Value may be nil
		TokPos     token.Pos   // position of Tok; invalid if Key == nil
		Tok        token.Token // ILLEGAL if Key == nil, ASSIGN, DEFINE
		X          Expr        // value to range over
		Body       *BlockStmt
		NoRangeOp  bool
	}
)

// Pos and End implementations for statement nodes.

// Pos returns position of first character belonging to the node.
func (s *BadStmt) Pos() token.Pos { return s.From }

// Pos returns position of first character belonging to the node.
func (s *DeclStmt) Pos() token.Pos { return s.Decl.Pos() }

// Pos returns position of first character belonging to the node.
func (s *EmptyStmt) Pos() token.Pos { return s.Semicolon }

// Pos returns position of first character belonging to the node.
func (s *LabeledStmt) Pos() token.Pos { return s.Label.Pos() }

// Pos returns position of first character belonging to the node.
func (s *ExprStmt) Pos() token.Pos { return s.X.Pos() }

// Pos returns position of first character belonging to the node.
func (s *SendStmt) Pos() token.Pos { return s.Chan.Pos() }

// Pos returns position of first character belonging to the node.
func (s *IncDecStmt) Pos() token.Pos { return s.X.Pos() }

// Pos returns position of first character belonging to the node.
func (s *AssignStmt) Pos() token.Pos { return s.Lhs[0].Pos() }

// Pos returns position of first character belonging to the node.
func (s *GoStmt) Pos() token.Pos { return s.Go }

// Pos returns position of first character belonging to the node.
func (s *DeferStmt) Pos() token.Pos { return s.Defer }

// Pos returns position of first character belonging to the node.
func (s *ReturnStmt) Pos() token.Pos { return s.Return }

// Pos returns position of first character belonging to the node.
func (s *BranchStmt) Pos() token.Pos { return s.TokPos }

// Pos returns position of first character belonging to the node.
func (s *BlockStmt) Pos() token.Pos { return s.Lbrace }

// Pos returns position of first character belonging to the node.
func (s *IfStmt) Pos() token.Pos { return s.If }

// Pos returns position of first character belonging to the node.
func (s *CaseClause) Pos() token.Pos { return s.Case }

// Pos returns position of first character belonging to the node.
func (s *SwitchStmt) Pos() token.Pos { return s.Switch }

// Pos returns position of first character belonging to the node.
func (s *TypeSwitchStmt) Pos() token.Pos { return s.Switch }

// Pos returns position of first character belonging to the node.
func (s *CommClause) Pos() token.Pos { return s.Case }

// Pos returns position of first character belonging to the node.
func (s *SelectStmt) Pos() token.Pos { return s.Select }

// Pos returns position of first character belonging to the node.
func (s *ForStmt) Pos() token.Pos { return s.For }

// Pos returns position of first character belonging to the node.
func (s *RangeStmt) Pos() token.Pos { return s.For }

// End returns position of first character immediately after the node.
func (s *BadStmt) End() token.Pos { return s.To }

// End returns position of first character immediately after the node.
func (s *DeclStmt) End() token.Pos { return s.Decl.End() }

// End returns position of first character immediately after the node.
func (s *EmptyStmt) End() token.Pos {
	if s.Implicit {
		return s.Semicolon
	}
	return s.Semicolon + 1 /* len(";") */
}

// End returns position of first character immediately after the node.
func (s *LabeledStmt) End() token.Pos { return s.Stmt.End() }

// End returns position of first character immediately after the node.
func (s *ExprStmt) End() token.Pos { return s.X.End() }

// End returns position of first character immediately after the node.
func (s *IncDecStmt) End() token.Pos {
	return s.TokPos + 2 /* len("++") */
}

// End returns position of first character immediately after the node.
func (s *AssignStmt) End() token.Pos { return s.Rhs[len(s.Rhs)-1].End() }

// End returns position of first character immediately after the node.
func (s *GoStmt) End() token.Pos { return s.Call.End() }

// End returns position of first character immediately after the node.
func (s *DeferStmt) End() token.Pos { return s.Call.End() }

// End returns position of first character immediately after the node.
func (s *ReturnStmt) End() token.Pos {
	if n := len(s.Results); n > 0 {
		return s.Results[n-1].End()
	}
	return s.Return + 6 // len("return")
}

// End returns position of first character immediately after the node.
func (s *BranchStmt) End() token.Pos {
	if s.Label != nil {
		return s.Label.End()
	}
	return token.Pos(int(s.TokPos) + len(s.Tok.String()))
}

// End returns position of first character immediately after the node.
func (s *BlockStmt) End() token.Pos {
	if s.Rbrace.IsValid() {
		return s.Rbrace + 1
	}
	if n := len(s.List); n > 0 {
		return s.List[n-1].End()
	}
	return s.Lbrace + 1
}

// End returns position of first character immediately after the node.
func (s *IfStmt) End() token.Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	return s.Body.End()
}

// End returns position of first character immediately after the node.
func (s *CaseClause) End() token.Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return s.Colon + 1
}

// End returns position of first character immediately after the node.
func (s *SwitchStmt) End() token.Pos { return s.Body.End() }

// End returns position of first character immediately after the node.
func (s *TypeSwitchStmt) End() token.Pos { return s.Body.End() }

// End returns position of first character immediately after the node.
func (s *CommClause) End() token.Pos {
	if n := len(s.Body); n > 0 {
		return s.Body[n-1].End()
	}
	return s.Colon + 1
}

// End returns position of first character immediately after the node.
func (s *SelectStmt) End() token.Pos { return s.Body.End() }

// End returns position of first character immediately after the node.
func (s *ForStmt) End() token.Pos { return s.Body.End() }

// End returns position of first character immediately after the node.
func (s *RangeStmt) End() token.Pos { return s.Body.End() }

// stmtNode() ensures that only statement nodes can be
// assigned to a Stmt.
func (*BadStmt) stmtNode()        {}
func (*DeclStmt) stmtNode()       {}
func (*EmptyStmt) stmtNode()      {}
func (*LabeledStmt) stmtNode()    {}
func (*ExprStmt) stmtNode()       {}
func (*SendStmt) stmtNode()       {}
func (*IncDecStmt) stmtNode()     {}
func (*AssignStmt) stmtNode()     {}
func (*GoStmt) stmtNode()         {}
func (*DeferStmt) stmtNode()      {}
func (*ReturnStmt) stmtNode()     {}
func (*BranchStmt) stmtNode()     {}
func (*BlockStmt) stmtNode()      {}
func (*IfStmt) stmtNode()         {}
func (*CaseClause) stmtNode()     {}
func (*SwitchStmt) stmtNode()     {}
func (*TypeSwitchStmt) stmtNode() {}
func (*CommClause) stmtNode()     {}
func (*SelectStmt) stmtNode()     {}
func (*ForStmt) stmtNode()        {}
func (*RangeStmt) stmtNode()      {}

// ----------------------------------------------------------------------------
// Declarations

// A Spec node represents a single (non-parenthesized) import,
// constant, type, or variable declaration.
type (
	// The Spec type stands for any of *ImportSpec, *ValueSpec, and *TypeSpec.
	Spec interface {
		Node
		specNode()
	}

	// An ImportSpec node represents a single package import.
	ImportSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Name    *Ident        // local package name (including "."); or nil
		Path    *BasicLit     // import path
		Comment *CommentGroup // line comments; or nil
		EndPos  token.Pos     // end of spec (overrides Path.Pos if nonzero)
	}

	// A ValueSpec node represents a constant or variable declaration
	// (ConstSpec or VarSpec production).
	//
	ValueSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Names   []*Ident      // value names (len(Names) > 0)
		Type    Expr          // value type; or nil
		Tag     *BasicLit     // classfile field tag; or nil
		Values  []Expr        // initial values; or nil
		Comment *CommentGroup // line comments; or nil
	}

	// A TypeSpec node represents a type declaration (TypeSpec production).
	TypeSpec struct {
		Doc        *CommentGroup // associated documentation; or nil
		Name       *Ident        // type name
		TypeParams *FieldList    // type parameters; or nil
		Assign     token.Pos     // position of '=', if any
		Type       Expr          // *Ident, *ParenExpr, *SelectorExpr, *StarExpr, or any of the *XxxTypes
		Comment    *CommentGroup // line comments; or nil
	}
)

// Pos and End implementations for spec nodes.

// Pos returns position of first character belonging to the node.
func (s *ImportSpec) Pos() token.Pos {
	if s.Name != nil {
		return s.Name.Pos()
	}
	return s.Path.Pos()
}

// Pos returns position of first character belonging to the node.
func (s *ValueSpec) Pos() token.Pos {
	if len(s.Names) == 0 {
		return s.Type.Pos()
	}
	return s.Names[0].Pos()
}

// Pos returns position of first character belonging to the node.
func (s *TypeSpec) Pos() token.Pos { return s.Name.Pos() }

// End returns position of first character immediately after the node.
func (s *ImportSpec) End() token.Pos {
	if s.EndPos != 0 {
		return s.EndPos
	}
	return s.Path.End()
}

// End returns position of first character immediately after the node.
func (s *ValueSpec) End() token.Pos {
	if n := len(s.Values); n > 0 {
		return s.Values[n-1].End()
	}
	if s.Type != nil {
		return s.Type.End()
	}
	return s.Names[len(s.Names)-1].End()
}

// End returns position of first character immediately after the node.
func (s *TypeSpec) End() token.Pos { return s.Type.End() }

// specNode() ensures that only spec nodes can be
// assigned to a Spec.
func (*ImportSpec) specNode() {}
func (*ValueSpec) specNode()  {}
func (*TypeSpec) specNode()   {}

// A declaration is represented by one of the following declaration nodes.
type (
	// A BadDecl node is a placeholder for declarations containing
	// syntax errors for which no correct declaration nodes can be
	// created.
	//
	BadDecl struct {
		From, To token.Pos // position range of bad declaration
	}

	// A GenDecl node (generic declaration node) represents an import,
	// constant, type or variable declaration. A valid Lparen position
	// (Lparen.IsValid()) indicates a parenthesized declaration.
	//
	// Relationship between Tok value and Specs element type:
	//
	//	token.IMPORT  *ImportSpec
	//	token.CONST   *ValueSpec
	//	token.TYPE    *TypeSpec
	//	token.VAR     *ValueSpec
	//
	GenDecl struct {
		Doc    *CommentGroup // associated documentation; or nil
		TokPos token.Pos     // position of Tok
		Tok    token.Token   // IMPORT, CONST, TYPE, VAR
		Lparen token.Pos     // position of '(', if any
		Specs  []Spec
		Rparen token.Pos // position of ')', if any
	}

	// A FuncDecl node represents a function declaration.
	FuncDecl struct {
		Doc      *CommentGroup // associated documentation; or nil
		Recv     *FieldList    // receiver (methods); or nil (functions)
		Name     *Ident        // function/method name
		Type     *FuncType     // function signature: parameters, results, and position of "func" keyword
		Body     *BlockStmt    // function body; or nil for external (non-Go) function
		Operator bool          // is operator or not
		Shadow   bool          // is a shadow entry
		IsClass  bool          // recv set by class
		Static   bool          // recv is static (class method)
	}
)

// Pos and End implementations for declaration nodes.

// Pos returns position of first character belonging to the node.
func (d *BadDecl) Pos() token.Pos { return d.From }

// Pos returns position of first character belonging to the node.
func (d *GenDecl) Pos() token.Pos { return d.TokPos }

// Pos returns position of first character belonging to the node.
func (d *FuncDecl) Pos() token.Pos { return d.Type.Pos() }

// End returns position of first character immediately after the node.
func (d *BadDecl) End() token.Pos { return d.To }

// End returns position of first character immediately after the node.
func (d *GenDecl) End() token.Pos {
	if d.Rparen.IsValid() {
		return d.Rparen + 1
	}
	return d.Specs[0].End()
}

// End returns position of first character immediately after the node.
func (d *FuncDecl) End() token.Pos {
	if d.Body != nil {
		return d.Body.End()
	}
	return d.Type.End()
}

// declNode() ensures that only declaration nodes can be
// assigned to a Decl.
func (*BadDecl) declNode()  {}
func (*GenDecl) declNode()  {}
func (*FuncDecl) declNode() {}

// ----------------------------------------------------------------------------

// A Package node represents a set of source files
// collectively building an XGo package.
type Package struct {
	Name    string               // package name
	Imports map[string]*Object   // map of package id -> package object
	Files   map[string]*File     // XGo source files by filename
	GoFiles map[string]*ast.File // Go source files by filename
}

// Pos returns position of first character belonging to the node.
func (p *Package) Pos() token.Pos { return token.NoPos }

// End returns position of first character immediately after the node.
func (p *Package) End() token.Pos { return token.NoPos }

// ----------------------------------------------------------------------------
