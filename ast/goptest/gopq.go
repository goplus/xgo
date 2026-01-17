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

package goptest

import (
	"go/token"

	"github.com/goplus/xgo/ast/gopq"
	"github.com/goplus/xgo/parser/fsx/memfs"
)

const (
	XGoPackage = "github.com/goplus/xgo/ast/gopq"
)

// -----------------------------------------------------------------------------

// New creates a nodeset object that represents an XGo dom tree.
func New(script string) (gopq.NodeSet, error) {
	fset := token.NewFileSet()
	fs := memfs.SingleFile("/foo", "bar.xgo", script)
	return gopq.FromFSDir(fset, fs, "/foo", nil, 0)
}

// -----------------------------------------------------------------------------
