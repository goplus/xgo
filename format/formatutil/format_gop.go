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

package formatutil

import (
	"github.com/goplus/xgo/format"
	"github.com/goplus/xgo/scanner"
	"github.com/goplus/xgo/token"
)

// RearrangeFuncs rearranges functions in src.
func RearrangeFuncs(src []byte, filename ...string) ([]byte, error) {
	var fname string
	if filename != nil {
		fname = filename[0]
	}

	fset := token.NewFileSet()
	base := fset.Base()
	f := fset.AddFile(fname, base, len(src))

	var s scanner.Scanner
	s.Init(f, src, nil, scanner.ScanComments)
	stmts := splitStmts(&s)
	first := firstNonDecl(stmts)
	if first < 0 { // no non-decl stmt
		return src, nil
	}

	ret := make([]byte, 0, len(src))
	off := int(stmts[first].words[0].pos) - base
	ret = append(ret, src[:off]...)

	rest := stmts[first:]
	for i, s := range rest {
		if s.isFuncDecl() {
			ret = append(ret, codeOf(src, base, i, rest)...)
		}
	}
	for i, s := range rest {
		if !s.isFuncDecl() {
			ret = append(ret, codeOf(src, base, i, rest)...)
		}
	}
	return ret, nil
}

func codeOf(src []byte, base, i int, rest []aStmt) []byte {
	from := int(rest[i].words[0].pos) - base
	to := 0
	if i == len(rest)-1 {
		to = len(src)
	} else {
		to = int(rest[i+1].words[0].pos) - base
	}
	return src[from:to]
}

func firstNonDecl(stmts []aStmt) int {
	for i, stmt := range stmts {
		if !stmt.isDecl() {
			return i
		}
	}
	return -1
}

func splitStmts(s *scanner.Scanner) (stmts []aStmt) {
	var level int
	var stmt aStmt
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			return
		}
		stmt.words = append(stmt.words, aWord{pos, tok})
		switch tok {
		case token.LBRACE:
			level++
		case token.RBRACE:
			level--
		}
		if tok == token.SEMICOLON && level == 0 {
			stmt.tok, stmt.at = tokOf(stmt.words)
			stmts = append(stmts, stmt)
			stmt = aStmt{}
			continue
		}
	}
}

type aWord struct {
	pos token.Pos
	tok token.Token
}

type aStmt struct {
	words []aWord
	tok   token.Token
	at    int
}

func (s aStmt) isFuncDecl() bool {
	return s.tok == token.FUNC && isFuncDecl(s.words[s.at+1:])
}

func (s aStmt) isDecl() bool {
	switch s.tok {
	case token.CONST, token.TYPE, token.VAR:
		return true
	case token.FUNC:
		return isFuncDecl(s.words[s.at+1:])
	}
	return false
}

func tokOf(words []aWord) (tok token.Token, at int) {
	for i, w := range words {
		if w.tok != token.COMMENT {
			return w.tok, i
		}
	}
	return words[0].tok, 0
}

func isFuncDecl(words []aWord) bool {
	if startWith(words, token.LPAREN) { // func (
		words = seekAfter(words[1:], token.RPAREN, token.LPAREN) // func (...)
		if startWith(words, token.LBRACE) {                      // func (...) {
			return false
		}
	}
	return true
}

func seekAfter(words []aWord, tokR, tokL token.Token) []aWord {
	level := 0
	for i, w := range words {
		switch w.tok {
		case tokR:
			if level == 0 {
				return words[i+1:]
			}
			level--
		case tokL:
			level++
		}
	}
	return nil
}

func startWith(words []aWord, tok token.Token) bool {
	for _, w := range words {
		switch w.tok {
		case token.COMMENT:
			continue
		case tok:
			return true
		}
		break
	}
	return false
}

// SourceEx formats src in canonical xgofmt style and returns the result
// or an (I/O or syntax) error. src is expected to be a syntactically
// correct XGo source file, or a list of XGo declarations or statements.
//
// If src is a partial source file, the leading and trailing space of src
// is applied to the result (such that it has the same leading and trailing
// space as src), and the result is indented by the same amount as the first
// line of src containing code. Imports are not sorted for partial source files.
func SourceEx(src []byte, class bool, filename ...string) (formatted []byte, err error) {
	formatted, err = format.Source(src, class, filename...)
	if err == nil {
		return
	}
	src, err = RearrangeFuncs(src, filename...)
	if err == nil {
		formatted, err = format.Source(src, class, filename...)
	}
	return
}
