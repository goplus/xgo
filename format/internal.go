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

// TODO(gri): This file and the file src/cmd/gofmt/internal.go are
// the same (but for this comment and the package name). Do not modify
// one without the other. Determine if we can factor out functionality
// in a public API. See also #11844 for context.

package format

import (
	"bytes"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/printer"
	"github.com/goplus/xgo/token"
)

// parse parses src, which was read from the named file,
// as a Go source file, declaration, or statement list.
func parse(fset *token.FileSet, filename string, src []byte, class, _ /* fragmentOk */ bool) (
	file *ast.File,
	sourceAdj func(src []byte, indent int) []byte,
	indentAdj int,
	err error,
) {
	mode := parserMode
	if class {
		mode |= parser.ParseXGoClass
	}
	file, err = parser.ParseFile(fset, filename, src, mode)
	return
}

// format formats the given package file originally obtained from src
// and adjusts the result based on the original source via sourceAdj
// and indentAdj.
func format(
	fset *token.FileSet,
	file *ast.File,
	sourceAdj func(src []byte, indent int) []byte,
	indentAdj int,
	src []byte,
	cfg printer.Config,
) ([]byte, error) {
	if sourceAdj == nil {
		// Complete source file.
		var buf bytes.Buffer
		err := cfg.Fprint(&buf, fset, file)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// Partial source file.
	// Determine and prepend leading space.
	i, j := 0, 0
	for j < len(src) && isSpace(src[j]) {
		if src[j] == '\n' {
			i = j + 1 // byte offset of last line in leading space
		}
		j++
	}
	var res []byte
	res = append(res, src[:i]...)

	// Determine and prepend indentation of first code line.
	// Spaces are ignored unless there are no tabs,
	// in which case spaces count as one tab.
	indent := 0
	hasSpace := false
	for _, b := range src[i:j] {
		switch b {
		case ' ':
			hasSpace = true
		case '\t':
			indent++
		}
	}
	if indent == 0 && hasSpace {
		indent = 1
	}
	for i := 0; i < indent; i++ {
		res = append(res, '\t')
	}

	// Format the source.
	// Write it without any leading and trailing space.
	cfg.Indent = indent + indentAdj
	var buf bytes.Buffer

	err := cfg.Fprint(&buf, fset, file)
	if err != nil {
		return nil, err
	}

	out := sourceAdj(buf.Bytes(), cfg.Indent)

	// If the adjusted output is empty, the source
	// was empty but (possibly) for white space.
	// The result is the incoming source.
	if len(out) == 0 {
		return src, nil
	}

	// Otherwise, append output to leading space.
	res = append(res, out...)
	return append(res, '\n'), nil
}

// isSpace reports whether the byte is a space character.
// isSpace defines a space as being among the following bytes: ' ', '\t', '\n' and '\r'.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
