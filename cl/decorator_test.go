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
	goast "go/ast"
	goparser "go/parser"
	"go/types"
	"testing"

	"github.com/goplus/xgo/token"
)

func TestGetDecoratorForm(t *testing.T) {
	for _, tt := range []struct {
		name, declarations, parameters string
		wantCount, wantKind            int
	}{
		{"Plain", "", "value int, fn func()", 1, decoFunc},
		{"PlainError", "", "value int, fn func() error", 1, decoFuncError},
		{"CallbackAlias", "type Callback = func()", "value int, fn Callback", 1, decoFunc},
		{"CallbackAliasChain", "type Callback = func()\ntype Next = Callback", "value int, fn Next", 1, decoFunc},
		{"ErrorAlias", "type Failure = error", "value int, fn func() Failure", 1, decoFuncError},
		{"ErrorAliasChain", "type Failure = error\ntype Next = Failure", "value int, fn func() Next", 1, decoFuncError},
		{"CallbackAndErrorAliases", "type Failure = error\ntype Callback = func() Failure", "value int, fn Callback", 1, decoFuncError},
		{"GenericCallbackAlias", "type Callback[T any] = func() T", "value int, fn Callback[error]", 1, decoFuncError},
		{"NoExplicitArguments", "type Callback = func()", "fn Callback", 0, decoFunc},
		{"MultipleExplicitArguments", "type Callback = func()", "value int, label string, fn Callback", 2, decoFunc},
		{"NoCallback", "", "", 0, decoInvalid},
		{"NonFunctionAlias", "type Callback = int", "value int, fn Callback", 0, decoInvalid},
		{"CallbackWithArguments", "type Callback = func(int)", "value int, fn Callback", 0, decoInvalid},
		{"VariadicCallback", "type Callback = func(...int)", "value int, fn Callback", 0, decoInvalid},
		{"WrongResult", "type Callback = func() int", "value int, fn Callback", 0, decoInvalid},
		{"MultipleResults", "type Callback = func() (int, error)", "value int, fn Callback", 0, decoInvalid},
		{"DefinedCallback", "type Callback func()", "value int, fn Callback", 0, decoInvalid},
		{"AliasToDefinedCallback", "type Callback func()\ntype Next = Callback", "value int, fn Next", 0, decoInvalid},
		{"DefinedError", "type Failure interface { Error() string }", "value int, fn func() Failure", 0, decoInvalid},
		{"AliasToDefinedError", "type Failure interface { Error() string }\ntype Next = Failure", "value int, fn func() Next", 0, decoInvalid},
		{"AnonymousErrorAlias", "type Failure = interface { Error() string }", "value int, fn func() Failure", 0, decoInvalid},
		{"VariadicDecorator", "", "value int, fn ...func()", 0, decoInvalid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := goparser.ParseFile(fset, "decorator.go", "package decorator\n"+tt.declarations+"\nfunc Decorate("+tt.parameters+") {}\n", 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg, err := new(types.Config).Check("decorator", fset, []*goast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			count, kind := getDecoratorForm(pkg.Scope().Lookup("Decorate").Type())
			if count != tt.wantCount || kind != tt.wantKind {
				t.Errorf("getDecoratorForm = (%d, %d), want (%d, %d)", count, kind, tt.wantCount, tt.wantKind)
			}
		})
	}

	t.Run("NonFunction", func(t *testing.T) {
		if count, kind := getDecoratorForm(types.Typ[types.Int]); count != 0 || kind != decoInvalid {
			t.Errorf("getDecoratorForm = (%d, %d), want (0, %d)", count, kind, decoInvalid)
		}
	})
}
