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

package tpl

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/goplus/xgo/tpl/ast"
	"github.com/goplus/xgo/tpl/cl"
	"github.com/goplus/xgo/tpl/matcher"
	"github.com/goplus/xgo/tpl/parser"
	"github.com/goplus/xgo/tpl/scanner"
	"github.com/goplus/xgo/tpl/token"
	"github.com/goplus/xgo/tpl/types"
	"github.com/qiniu/x/errors"
	"github.com/qiniu/x/stream"
)

// -----------------------------------------------------------------------------

var showConflict = true

// ShowConflict sets the flag to show or hide conflicts.
func ShowConflict(f bool) int {
	showConflict = f
	return 0
}

func onConflictHidden(fset *token.FileSet, c *ast.Choice, firsts [][]any, i, at int) {
}

// -----------------------------------------------------------------------------

func relocatePos(ePos *token.Position, filename string, line, col int) {
	ePos.Filename = filename
	if ePos.Line == line {
		ePos.Column += col - 1
	} else {
		ePos.Line += line - 1
	}
}

// Relocate relocates the error positions.
func Relocate(err error, filename string, line, col int) error {
	switch e := err.(type) {
	case *matcher.Error:
		pos := e.Fset.Position(e.Pos)
		relocatePos(&pos, filename, line, col)
		return &scanner.Error{Pos: pos, Msg: e.Msg}
	case errors.List:
		for i, ie := range e {
			e[i] = Relocate(ie, filename, line, col)
		}
	case scanner.ErrorList:
		for _, ie := range e {
			relocatePos(&ie.Pos, filename, line, col)
		}
	case *scanner.Error:
		relocatePos(&e.Pos, filename, line, col)
	default:
		panic("todo: " + reflect.TypeOf(err).String())
	}
	return err
}

// -----------------------------------------------------------------------------

// Compiler represents a TPL compiler.
type Compiler struct {
	cl.Result
}

// New creates a new TPL compiler.
// params: ruleName1, retProc1, ..., ruleNameN, retProcN
func New(src any, params ...any) (ret Compiler, err error) {
	conf := &cl.Config{
		RetProcs: retProcs(params),
	}
	if !showConflict {
		conf.OnConflict = onConflictHidden
	}
	return FromFile(nil, "", src, conf)
}

// NewEx creates a new TPL compiler.
// params: ruleName1, retProc1, ..., ruleNameN, retProcN
func NewEx(src any, filename string, line, col int, params ...any) (ret Compiler, err error) {
	conf := &cl.Config{
		RetProcs: retProcs(params),
	}
	if showConflict {
		conf.OnConflict = func(fset *token.FileSet, c *ast.Choice, firsts [][]any, i, at int) {
			if showConflict {
				pos := fset.Position(c.Options[i].Pos())
				relocatePos(&pos, filename, line, col)
				cl.LogConflict(pos, firsts, i, at)
			}
		}
	} else {
		conf.OnConflict = onConflictHidden
	}
	ret, err = FromFile(nil, "", src, conf)
	if err != nil {
		err = Relocate(err, filename, line, col)
	}
	return
}

// FromFile creates a new TPL compiler from a file.
// fset can be nil.
func FromFile(fset *token.FileSet, filename string, src any, conf *cl.Config) (ret Compiler, err error) {
	if fset == nil {
		fset = token.NewFileSet()
	}
	f, err := parser.ParseFile(fset, filename, src, nil)
	if err != nil {
		return
	}
	ret.Result, err = cl.NewEx(conf, fset, f)
	return
}

func retProcs(params []any) map[string]any {
	n := len(params)
	if n == 0 {
		return nil
	}
	if n&1 != 0 {
		panic("tpl.New: invalid params. should be in form `ruleName1, retProc1, ..., ruleNameN, retProcN`")
	}
	ret := make(map[string]any, n>>1)
	for i := 0; i < n; i += 2 {
		ret[params[i].(string)] = params[i+1]
	}
	return ret
}

// -----------------------------------------------------------------------------

// Error represents a matching error.
type Error = matcher.Error

// A Token is a lexical unit returned by Scan.
type Token = types.Token

// Scanner represents a TPL scanner.
type Scanner interface {
	Scan() Token
	Init(file *token.File, src []byte, err scanner.ErrorHandler, mode scanner.Mode)
}

// Config represents a parsing configuration of [Compiler.Parse].
type Config struct {
	Scanner          Scanner
	ScanErrorHandler scanner.ErrorHandler
	ScanMode         scanner.Mode
	Fset             *token.FileSet
}

// ParseExpr parses an expression.
func (p *Compiler) ParseExpr(x string, conf *Config) (result any, err error) {
	return p.ParseExprFrom("", x, conf)
}

// ParseExprFrom parses an expression from a file.
func (p *Compiler) ParseExprFrom(filename string, src any, conf *Config) (result any, err error) {
	ms, result, err := p.Match(filename, src, conf)
	if err != nil {
		return
	}
	if len(ms.Toks) == ms.N || isEOL(ms.Toks[ms.N].Tok) {
		return
	}
	t := ms.Next()
	err = ms.Ctx.NewErrorf(t.Pos, "unexpected token: %v", t)
	return
}

// Parse parses a source file.
func (p *Compiler) Parse(filename string, src any, conf *Config) (result any, err error) {
	ms, result, err := p.Match(filename, src, conf)
	if err != nil {
		return
	}
	if len(ms.Toks) > ms.N {
		t := ms.Next()
		err = ms.Ctx.NewErrorf(t.Pos, "unexpected token: %v", t)
	}
	return
}

// MatchState represents a matching state.
type MatchState struct {
	Toks []*Token
	Ctx  *matcher.Context
	N    int
}

// Next returns the next token.
func (p *MatchState) Next() *Token {
	n := p.Ctx.Left
	if n > 0 {
		return p.Toks[len(p.Toks)-n]
	}
	return &Token{Tok: token.EOF, Pos: p.Ctx.FileEnd}
}

// Match matches a source file.
func (p *Compiler) Match(filename string, src any, conf *Config) (ms MatchState, result any, err error) {
	b, err := stream.ReadSourceLocal(filename, src)
	if err != nil {
		return
	}
	if conf == nil {
		conf = &Config{}
	}
	s := conf.Scanner
	if s == nil {
		s = new(scanner.Scanner)
	}
	fset := conf.Fset
	if fset == nil {
		fset = token.NewFileSet()
	}
	f := fset.AddFile(filename, fset.Base(), len(b))
	s.Init(f, b, conf.ScanErrorHandler, conf.ScanMode)
	n := (len(b) >> 3) &^ 7
	if n < 8 {
		n = 8
	}
	toks := make([]*Token, 0, n)
	for {
		t := s.Scan()
		if t.Tok == token.EOF {
			break
		}
		toks = append(toks, &t)
	}
	ms.Ctx = matcher.NewContext(fset, token.Pos(f.Base()+len(b)), toks)
	ms.N, result, err = p.Doc.Match(toks, ms.Ctx)
	ms.Ctx.SetLastError(len(toks)-ms.N, err)
	if err != nil {
		return
	}
	ms.Toks = toks
	return
}

// -----------------------------------------------------------------------------

func isEOL(tok token.Token) bool {
	return tok == token.SEMICOLON || tok == token.EOF
}

func isPlain(result []any) bool {
	for _, v := range result {
		if _, ok := scalar(v); !ok {
			return false
		}
	}
	return true
}

func scalar(v any) (any, bool) {
	if v == nil {
		return nil, true
	}
retry:
	switch result := v.(type) {
	case *Token:
		return v, true
	case []any:
		if len(result) == 2 {
			if isVoid(result[1]) {
				v = result[0]
				goto retry
			}
		}
		return v, len(result) == 0
	}
	return v, false
}

func isVoid(v any) bool {
	if v == nil {
		return true
	}
	switch v := v.(type) {
	case *Token:
		return v.Tok == token.SEMICOLON || v.Tok == token.EOF
	case []any:
		return len(v) == 0
	}
	return false
}

// -----------------------------------------------------------------------------

func Dump(result any, omitSemi ...bool) {
	Fdump(os.Stdout, result, "", "  ", omitSemi != nil && omitSemi[0])
}

func Fdump(w io.Writer, ret any, prefix, indent string, omitSemi bool) {
retry:
	switch result := ret.(type) {
	case *Token:
		if result.Tok != token.SEMICOLON {
			fmt.Fprint(w, prefix, result, "\n")
		} else if !omitSemi {
			fmt.Fprint(w, prefix, ";\n")
		}
	case []any:
		if len(result) == 2 {
			if isVoid(result[1]) {
				ret = result[0]
				goto retry
			}
		}
		if isPlain(result) {
			fmt.Print(prefix, "[")
			for i, v := range result {
				if i > 0 {
					fmt.Print(" ")
				}
				v, _ = scalar(v)
				fmt.Fprint(w, v)
			}
			fmt.Print("]\n")
		} else {
			fmt.Print(prefix, "[\n")
			for _, v := range result {
				Fdump(w, v, prefix+indent, indent, omitSemi)
			}
			fmt.Print(prefix, "]\n")
		}
	case nil:
		fmt.Fprint(w, prefix, "nil\n")
	default:
		panic("unexpected node: " + reflect.TypeOf(ret).String())
	}
}

// -----------------------------------------------------------------------------

// List converts the matching result of (R % ",") to a flat list.
// R % "," means R *("," R)
func List(in []any) []any {
	next := in[1].([]any)
	ret := make([]any, len(next)+1)
	ret[0] = in[0]
	for i, v := range next {
		ret[i+1] = v.([]any)[1]
	}
	return ret
}

// ListOp converts the matching result of (R % ",") to a flat list.
// R % "," means R *("," R)
func ListOp[T any](in []any, fn func(v any) T) []T {
	next := in[1].([]any)
	ret := make([]T, len(next)+1)
	ret[0] = fn(in[0])
	for i, v := range next {
		ret[i+1] = fn(v.([]any)[1])
	}
	return ret
}

// RangeOp travels the matching result of (R % ",") and call fn(result of R).
// R % "," means R *("," R)
func RangeOp(in []any, fn func(v any)) {
	next := in[1].([]any)
	fn(in[0])
	for _, v := range next {
		fn(v.([]any)[1])
	}
}

// BinaryExpr converts the matching result of (X % op) to a binary expression.
// X % op means X *(op X)
func BinaryExpr(recursive bool, in []any) ast.Expr {
	if recursive {
		return BinaryExprR(in)
	}
	return BinaryExprNR(in)
}

func BinaryExprR(in []any) ast.Expr {
	var ret, y ast.Expr
	switch v := in[0].(type) {
	case []any:
		ret = BinaryExprR(v)
	default:
		ret = v.(ast.Expr)
	}
	for _, v := range in[1].([]any) {
		next := v.([]any)
		op := next[0].(*Token)
		switch v := next[1].(type) {
		case []any:
			y = BinaryExprR(v)
		default:
			y = v.(ast.Expr)
		}
		ret = &ast.BinaryExpr{
			X:     ret,
			OpPos: op.Pos,
			Op:    op.Tok,
			Y:     y,
		}
	}
	return ret
}

func BinaryExprNR(in []any) ast.Expr {
	ret := in[0].(ast.Expr)
	for _, v := range in[1].([]any) {
		next := v.([]any)
		op := next[0].(*Token)
		y := next[1].(ast.Expr)
		ret = &ast.BinaryExpr{
			X:     ret,
			OpPos: op.Pos,
			Op:    op.Tok,
			Y:     y,
		}
	}
	return ret
}

func BinaryOp(recursive bool, in []any, fn func(op *Token, x, y any) any) any {
	if recursive {
		return BinaryOpR(in, fn)
	}
	return BinaryOpNR(in, fn)
}

func BinaryOpR(in []any, fn func(op *Token, x, y any) any) any {
	ret := in[0]
	if v, ok := ret.([]any); ok {
		ret = BinaryOpR(v, fn)
	}
	for _, v := range in[1].([]any) {
		next := v.([]any)
		op := next[0].(*Token)
		y := next[1]
		if v, ok := y.([]any); ok {
			y = BinaryOpR(v, fn)
		}
		ret = fncall(fn, op, ret, y)
	}
	return ret
}

func BinaryOpNR(in []any, fn func(op *Token, x, y any) any) any {
	ret := in[0]
	for _, v := range in[1].([]any) {
		next := v.([]any)
		op := next[0].(*Token)
		y := next[1]
		ret = fncall(fn, op, ret, y)
	}
	return ret
}

func fncall(fn func(op *Token, x, y any) any, op *Token, x, y any) any {
	defer func() {
		if e := recover(); e != nil {
			panic(toErr(e, op))
		}
	}()
	return fn(op, x, y)
}

func toErr(e any, op *Token) error {
	switch e := e.(type) {
	case *matcher.Error:
		return e
	case string:
		return &matcher.Error{
			Pos: op.Pos,
			Msg: e,
			Dyn: true,
		}
	}
	return e.(error)
}

// UnaryExpr converts the matching result of (op X) to a unary expression.
func UnaryExpr(in []any) ast.Expr {
	op := in[0].(*Token)
	return &ast.UnaryExpr{
		OpPos: op.Pos,
		Op:    op.Tok,
		X:     in[1].(ast.Expr),
	}
}

// Ident converts the matching result of an identifier to an ast.Ident expression.
func Ident(this any) *ast.Ident {
	v := this.(*Token)
	return &ast.Ident{
		NamePos: v.Pos,
		Name:    v.Lit,
	}
}

// BasicLit converts the matching result of a basic literal to an ast.BasicLit expression.
func BasicLit(this any) *ast.BasicLit {
	v := this.(*Token)
	return &ast.BasicLit{
		ValuePos: v.Pos,
		Kind:     v.Tok,
		Value:    v.Lit,
	}
}

// -----------------------------------------------------------------------------

// Panic panics with a matcher error.
func Panic(pos token.Pos, msg string) {
	err := &matcher.Error{
		Pos: pos,
		Msg: msg,
		Dyn: true,
	}
	panic(err)
}

// -----------------------------------------------------------------------------
