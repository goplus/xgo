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

package parsertest

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
	tpltoken "github.com/goplus/xgo/tpl/token"
	"github.com/qiniu/x/test"
)

func sortedKeys(m any) []string {
	iter := reflect.ValueOf(m).MapRange()
	keys := make([]string, 0, 8)
	for iter.Next() {
		key := iter.Key()
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	return keys
}

var (
	tyNode      = reflect.TypeOf((*ast.Node)(nil)).Elem()
	tyString    = reflect.TypeOf("")
	tyBytes     = reflect.TypeOf([]byte(nil))
	tyToken     = reflect.TypeOf(token.Token(0))
	tyObjectPtr = reflect.TypeOf((*ast.Object)(nil))
	tyScopePtr  = reflect.TypeOf((*ast.Scope)(nil))
	tplToken    = reflect.TypeOf(tpltoken.Token(0))
)

// FprintNode prints a XGo AST node.
func FprintNode(w io.Writer, lead string, v any, prefix, indent string) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice:
		n := val.Len()
		if n > 0 && lead != "" {
			io.WriteString(w, lead)
		}
		for i := 0; i < n; i++ {
			FprintNode(w, "", val.Index(i).Interface(), prefix, indent)
		}
	case reflect.Ptr:
		t := val.Type()
		if val.IsNil() || t == tyObjectPtr || t == tyScopePtr {
			return
		}
		if t.Implements(tyNode) {
			if lead != "" {
				io.WriteString(w, lead)
			}
			elem, tyElem := val.Elem(), t.Elem()
			fmt.Fprintf(w, "%s%v:\n", prefix, tyElem)
			n := elem.NumField()
			prefix += indent
			for i := 0; i < n; i++ {
				sf := tyElem.Field(i)
				sfv := elem.Field(i).Interface()
				switch sf.Type {
				case tyString, tyToken, tplToken:
					fmt.Fprintf(w, "%s%v: %v\n", prefix, sf.Name, sfv)
				case tyBytes: // skip
				default:
					FprintNode(w, fmt.Sprintf("%s%v:\n", prefix, sf.Name), sfv, prefix+indent, indent)
				}
			}
			if m, ok := v.(*ast.MatrixLit); ok {
				fmt.Fprintf(w, "%sNElt: %d\n", prefix, len(m.Elts))
			}
		} else if lit, ok := v.(*ast.StringLitEx); ok {
			fmt.Fprintf(w, "%sExtra:\n", prefix)
			prefix += indent
			for _, part := range lit.Parts {
				if val, ok := part.(string); ok {
					fmt.Fprintf(w, "%s%v\n", prefix, val)
				} else {
					FprintNode(w, "", part, prefix, indent)
				}
			}
		} else if lit, ok := v.(*ast.DomainTextLitEx); ok {
			fmt.Fprintf(w, "%sExtra: args=%d\n", prefix, len(lit.Args))
			prefix += indent
			for _, arg := range lit.Args {
				FprintNode(w, "", arg, prefix, indent)
			}
			fmt.Fprintf(w, "%s%v\n", prefix, lit.Raw)
		} else {
			log.Panicln("FprintNode unexpected type:", t)
		}
	case reflect.Int, reflect.Bool, reflect.Invalid:
		// skip
	default:
		log.Panicln("FprintNode unexpected kind:", val.Kind(), "type:", val.Type())
	}
}

// Fprint prints a XGo package.
func Fprint(w io.Writer, pkg *ast.Package) {
	fmt.Fprintf(w, "package %s\n", pkg.Name)
	paths := sortedKeys(pkg.Files)
	for _, fpath := range paths {
		fmt.Fprintf(w, "\nfile %s\n", filepath.Base(fpath))
		file := pkg.Files[fpath]
		if file.HasShadowEntry() {
			fmt.Fprintf(w, "noEntrypoint\n")
		}
		FprintNode(w, "", file.Decls, "", "  ")
	}
}

// Expect asserts a XGo AST package equals output or not.
func Expect(t *testing.T, pkg *ast.Package, expected string) {
	b := bytes.NewBuffer(nil)
	Fprint(b, pkg)
	output := b.String()
	if expected != output {
		fmt.Fprint(os.Stderr, output)
		t.Fatal("xgo.Parser: unexpect result")
	}
}

func ExpectEx(t *testing.T, outfile string, pkg *ast.Package, expected []byte) {
	b := bytes.NewBuffer(nil)
	Fprint(b, pkg)
	if test.Diff(t, outfile, b.Bytes(), expected) {
		t.Fatal("xgo.Parser: unexpect result")
	}
}

// ExpectNode asserts a XGo AST node equals output or not.
func ExpectNode(t *testing.T, node any, expected string) {
	b := bytes.NewBuffer(nil)
	FprintNode(b, "", node, "", "  ")
	output := b.String()
	if expected != output {
		fmt.Fprint(os.Stderr, output)
		t.Fatal("xgo.Parser: unexpect result")
	}
}
