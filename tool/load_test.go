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

package tool

import (
	"testing"

	"github.com/goplus/mod/xgomod"
)

// TestAfterLoadNilPackage guards against the regression reported in
// https://github.com/goplus/xgo/issues/2861: when compilation fails,
// LoadFiles/LoadDir may leave the resulting *gogen.Package nil. afterLoad
// (and thus checkGopDeps -> gogen.(*Package).ForEachFile) must not be reached
// with a nil package, otherwise it panics with a nil pointer dereference.
func TestAfterLoadNilPackage(t *testing.T) {
	// A real (empty) module whose Path() is not xgoMod, so afterLoad proceeds
	// past the "XGo itself" short-circuit and would reach checkGopDeps.
	mod := &xgomod.Module{}

	// A non-nil XGoDeps forces the checkGopDeps path in afterLoad, which is
	// exactly the `xgo run` code path from the panic in issue #2861.
	deps := 0
	conf := &Config{XGoDeps: &deps}

	// This must return cleanly instead of panicking on a nil package.
	afterLoad(mod, nil, nil, nil, conf)

	if deps != 0 {
		t.Fatalf("XGoDeps = %d, want 0 for a nil package", deps)
	}
}
