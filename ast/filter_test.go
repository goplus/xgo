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

package ast_test

import (
	"slices"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

func TestFileExportsEnumType(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "enum.xgo", []byte(`package p

type Color const (
	Red = iota
	green
	Blue
)

type hidden const (
	Shown = iota
	hiddenValue
)
`), 0)
	if err != nil {
		t.Fatal(err)
	}

	if !ast.FileExports(file) {
		t.Fatal("FileExports reported no exported declarations")
	}

	names := enumValueNames(t, file, "Color")
	if want := []string{"Red", "Blue"}; !slices.Equal(names, want) {
		t.Fatalf("exported enum values = %q, want %q", names, want)
	}
}

func enumValueNames(t *testing.T, file *ast.File, typeName string) []string {
	t.Helper()

	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			enumType := typeSpec.Type.(*ast.EnumType)
			for _, spec := range enumType.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				names = append(names, valueSpec.Names[0].Name)
			}
			return names
		}
	}
	t.Fatalf("enum type %q not found", typeName)
	return nil
}
