/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package cl

import (
	"strings"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

func TestParseTypeEmbedName(t *testing.T) {
	for _, tt := range []struct {
		name, embedded, want string
	}{
		{name: "Alias", embedded: "Alias", want: "Alias"},
		{name: "Pointer", embedded: "*Alias", want: "Alias"},
		{name: "Qualified", embedded: "sdk.Alias", want: "Alias"},
		{name: "Generic", embedded: "Box[int]", want: "Box"},
		{name: "QualifiedGenericPointer", embedded: "*sdk.Box[int]", want: "Box"},
		{name: "MultipleTypeArgs", embedded: "Pair[int, string]", want: "Pair"},
		{name: "QualifiedMultipleTypeArgs", embedded: "*sdk.Pair[int, string]", want: "Pair"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := "type Holder struct {\n\t" + tt.embedded + "\n}\n"
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "main.xgo", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			var fields []*ast.Field
			var want *ast.Ident
			ast.Inspect(file, func(node ast.Node) bool {
				if struc, ok := node.(*ast.StructType); ok {
					fields = struc.Fields.List
				}
				if ident, ok := node.(*ast.Ident); ok && ident.Name == tt.want {
					want = ident
				}
				return true
			})
			if len(fields) != 1 || want == nil {
				t.Fatal("missing embedded field or source identifier")
			}
			got := parseTypeEmbedName(fields[0].Type)
			if got != want || fset.Position(got.Pos()).Offset != strings.Index(source, tt.want) {
				t.Errorf("embedded name = %v at %v, want original %s identifier", got, fset.Position(got.Pos()), tt.want)
			}
		})
	}
}
