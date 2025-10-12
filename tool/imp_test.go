/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package tool

import (
	"go/token"
	"strings"
	"testing"

	"github.com/goplus/mod/env"
	"github.com/goplus/mod/xgomod"
)

func TestCycleImportDetection(t *testing.T) {
	fset := token.NewFileSet()
	xgo := &env.XGo{Version: "1.0", Root: "../.."}
	mod := xgomod.Default
	imp := NewImporter(mod, xgo, fset)

	imp.importStack["test/pkg"] = true

	_, err := imp.Import("test/pkg")
	if err == nil {
		t.Fatal("Expected cycle import error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle import") {
		t.Fatalf("Expected cycle import error message, got: %v", err)
	}
}
