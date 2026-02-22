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

package parser

import (
	"github.com/goplus/xgo/tpl/ast"
	"github.com/goplus/xgo/tpl/scanner"
	"github.com/goplus/xgo/tpl/token"
	"github.com/qiniu/x/stream"
)

// -----------------------------------------------------------------------------

// RetProcParser parses a RetProc.
type RetProcParser = func(file *token.File, src []byte, offset int) (ast.Node, scanner.ErrorList)

// Config configures the behavior of the parser.
type Config struct {
	ParseRetProc RetProcParser
}

// ParseFile parses a file and returns the AST.
func ParseFile(fset *token.FileSet, filename string, src any, conf *Config) (f *ast.File, err error) {
	b, err := stream.ReadSourceLocal(filename, src)
	if err != nil {
		return nil, err
	}
	file := fset.AddFile(filename, -1, len(b))
	f, errs := ParseEx(file, b, 0, conf)
	switch errs.Len() {
	case 0:
	case 1:
		err = errs[0]
	default:
		errs.Sort()
		err = errs
	}
	return
}

// ParseEx parses src[offset:] and returns the AST.
func ParseEx(file *token.File, src []byte, offset int, conf *Config) (f *ast.File, errs scanner.ErrorList) {
	var p parser
	p.init(file, src, offset)
	if conf != nil {
		p.parseRetProc = conf.ParseRetProc
	}
	return p.parseFile(), p.errors
}

// -----------------------------------------------------------------------------

// parser represents a parser.
type parser struct {
	scanner scanner.Scanner
	file    *token.File

	// Current token
	pos token.Pos
	tok token.Token
	lit string

	// Callback to parse RetProc.
	parseRetProc RetProcParser

	// Error handling
	errors scanner.ErrorList
}

func (p *parser) init(file *token.File, src []byte, offset int) {
	p.file = file
	eh := func(pos token.Position, msg string) { p.errors.Add(pos, msg) }
	p.scanner.InitEx(p.file, src, offset, eh, 0)
	p.next() // initialize first token
}

// next advances to the next token.
func (p *parser) next() {
	t := p.scanner.Scan()
	p.pos, p.tok, p.lit = t.Pos, t.Tok, t.Lit
}

func (p *parser) errorExpected(pos token.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		switch {
		case p.tok == token.SEMICOLON && p.lit == "\n":
			msg += ", found newline"
		case len(p.lit) > 0:
			// print 123 rather than 'INT', etc.
			msg += ", found " + p.lit
		default:
			msg += ", found '" + p.tok.String() + "'"
		}
	}
	p.error(pos, msg)
}

// expect consumes the current token if it matches the expected token.
// If not, it adds an error.
func (p *parser) expect(tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
	}
	p.next() // make progress
	return pos
}

func (p *parser) error(pos token.Pos, msg string) {
	epos := p.file.Position(pos)
	p.errors.Add(epos, msg)
}

// parseFile parses a file and returns the AST.
func (p *parser) parseFile() *ast.File {
	file := &ast.File{
		FileStart: p.pos,
	}

	for p.tok != token.EOF {
		rule := p.parseRule()
		if rule == nil {
			break
		}
		file.Decls = append(file.Decls, rule)
	}

	return file
}

func (p *parser) parseIdent() *ast.Ident {
	pos := p.pos
	name := "_"
	if p.tok == token.IDENT {
		name = p.lit
		p.next()
	} else {
		p.errorExpected(p.pos, "'IDENT'")
	}
	return &ast.Ident{NamePos: pos, Name: name}
}

// parseRule parses a rule:
//
//	IDENT '=' expr ';'
//	IDENT '=' expr => { ... } ';'
func (p *parser) parseRule() *ast.Rule {
	if p.tok != token.IDENT {
		p.errorExpected(p.pos, "'IDENT'")
		return nil
	}

	name := p.parseIdent()
	tokPos := p.expect(token.ASSIGN)
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}

	var retProc ast.Node
	if p.tok == token.DRARROW { // => { ... }
		if off, end, ok := p.lambdaExpr(); ok {
			if p.parseRetProc != nil {
				file := p.file
				base := file.Base()
				src := p.scanner.CodeTo(int(end) - base)
				expr, err := p.parseRetProc(file, src, int(off)-base)
				if err == nil {
					retProc = expr
				} else {
					p.errors = append(p.errors, err...)
				}
			}
		}
	}

	p.expect(token.SEMICOLON)
	return &ast.Rule{
		Name:    name,
		TokPos:  tokPos,
		Expr:    expr,
		RetProc: retProc,
	}
}

func (p *parser) lambdaExpr() (start, end token.Pos, ok bool) {
	start = p.pos // => {
	p.next()
	p.expect(token.LBRACE)
	level := 1
	for {
		switch p.tok {
		case token.RBRACE:
			level--
			if level == 0 { // }
				p.next()
				end, ok = p.pos, true
				return
			}
		case token.LBRACE:
			level++
		case token.EOF:
			return
		}
		p.next()
	}
}

// parseExpr: termList % '|'
func (p *parser) parseExpr() ast.Expr {
	termList := p.parseTermList()
	for p.tok != token.OR {
		return termList
	}

	options := make([]ast.Expr, 0, 4)
	options = append(options, termList)
	for p.tok == token.OR {
		p.next()
		termList = p.parseTermList()
		options = append(options, termList)
	}
	return &ast.Choice{Options: options}
}

// parseTermList: +term
func (p *parser) parseTermList() ast.Expr {
	terms := make([]ast.Expr, 0, 1)
	for {
		term, ok := p.parseTerm()
		if !ok {
			break
		}
		terms = append(terms, term)
	}
	switch n := len(terms); n {
	case 1:
		return terms[0]
	case 0:
		p.error(p.pos, "expected factor")
		fallthrough // TODO(xsw): BadExpr
	default:
		return &ast.Sequence{Items: terms}
	}
}

// parseTerm: term2 % '%'
func (p *parser) parseTerm() (ast.Expr, bool) {
	x, ok := p.parseTerm2()
	if !ok {
		return nil, false
	}

	for p.tok == token.REM {
		opPos := p.pos
		p.next()
		y, ok := p.parseTerm2()
		if !ok {
			p.error(p.pos, "expected factor")
			return x, false
		}
		x = &ast.BinaryExpr{
			X:     x,
			OpPos: opPos,
			Op:    token.REM,
			Y:     y,
		}
	}
	return x, true
}

// parseTerm2: factor % "++"
func (p *parser) parseTerm2() (ast.Expr, bool) {
	x, ok := p.parseFactor()
	if !ok {
		return nil, false
	}

	for p.tok == token.INC {
		opPos := p.pos
		p.next()
		y, ok := p.parseFactor()
		if !ok {
			p.error(p.pos, "expected factor")
			return x, false
		}
		x = &ast.BinaryExpr{
			X:     x,
			OpPos: opPos,
			Op:    token.INC,
			Y:     y,
		}
	}
	return x, true
}

// parseFactor: IDENT | CHAR | STRING | ('*' | '+' | '?') factor | '(' expr ')'
func (p *parser) parseFactor() (ast.Expr, bool) {
	switch tok := p.tok; tok {
	case token.IDENT:
		ident := &ast.Ident{
			NamePos: p.pos,
			Name:    p.lit,
		}
		p.next()
		return ident, true

	case token.CHAR, token.STRING:
		lit := &ast.BasicLit{
			ValuePos: p.pos,
			Kind:     tok,
			Value:    p.lit,
		}
		p.next()
		return lit, true

	case token.MUL, token.ADD, token.QUESTION:
		opPos := p.pos
		p.next()

		factor, ok := p.parseFactor()
		if !ok {
			p.error(p.pos, "expected factor")
		}
		ret := &ast.UnaryExpr{
			OpPos: opPos,
			Op:    tok,
			X:     factor,
		}
		return ret, true

	case token.LPAREN:
		p.next()
		expr := p.parseExpr()
		p.expect(token.RPAREN)
		return expr, true

	default:
		return nil, false
	}
}

// -----------------------------------------------------------------------------
