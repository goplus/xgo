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

package golang

import (
	"go/parser"
	"strings"

	"github.com/goplus/xgo/dql/golang"
)

const (
	XGoPackage = "github.com/goplus/xgo/dql/golang"
)

// Object represents a Go File.
type Object = golang.File

// New parses Go source code from the given text, returning a Go File object.
// An optional parser Mode can be provided to customize the parsing behavior.
func New(text string, mode ...parser.Mode) (f *Object, err error) {
	var conf []golang.Config
	if len(mode) > 0 {
		conf = []golang.Config{{Mode: mode[0]}}
	}
	return golang.ParseFile("", strings.NewReader(text), conf...)
}
