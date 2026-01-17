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

package scannertest

import (
	"fmt"
	"io"

	"github.com/goplus/xgo/tpl/scanner"
	"github.com/goplus/xgo/tpl/token"

	goscanner "go/scanner"
	gotoken "go/token"

	gopscanner "github.com/goplus/xgo/scanner"
	goptoken "github.com/goplus/xgo/token"
)

// -----------------------------------------------------------------------------

// Scan scans the input and writes the tokens to the writer.
func Scan(w io.Writer, in []byte) {
	var s scanner.Scanner
	fset := gotoken.NewFileSet()
	f := fset.AddFile("", -1, len(in))
	s.Init(f, in, nil, scanner.ScanComments)
	for {
		t := s.Scan()
		if t.Tok == token.EOF {
			break
		}
		fmt.Fprintln(w, t.Pos, t.Tok, t.Lit)
	}
}

// -----------------------------------------------------------------------------

// XGoScan scans the input using the XGo standard library scanner and
// writes the tokens to the writer.
func XGoScan(w io.Writer, in []byte) {
	var s gopscanner.Scanner
	fset := gotoken.NewFileSet()
	f := fset.AddFile("", -1, len(in))
	s.Init(f, in, nil, gopscanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		if tok == goptoken.EOF {
			break
		}
		fmt.Fprintln(w, pos, tok, lit)
	}
}

// -----------------------------------------------------------------------------

// GoScan scans the input using the Go standard library scanner and
// writes the tokens to the writer.
func GoScan(w io.Writer, in []byte) {
	var s goscanner.Scanner
	fset := gotoken.NewFileSet()
	f := fset.AddFile("", -1, len(in))
	s.Init(f, in, nil, goscanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		if tok == gotoken.EOF {
			break
		}
		fmt.Fprintln(w, pos, tok, lit)
	}
}

// -----------------------------------------------------------------------------
