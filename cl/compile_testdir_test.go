/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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

package cl_test

import (
	"runtime"
	"testing"

	"github.com/goplus/gogen/target"
	"github.com/goplus/xgo/cl/cltest"
)

func TestTestspx(t *testing.T) {
	if target.Kind == target.Go {
		cltest.SpxFromDir(t, "", "./_testspx")
	}
}

func TestTestjs(t *testing.T) {
	if runtime.GOOS != "windows" {
		cltest.FromDirEx(t, "", "./_testjs", false, true)
	}
}

func TestTestgop(t *testing.T) {
	cltest.FromDir(t, "", "./_testgop")
}

func TestTestc(t *testing.T) {
	cltest.FromDir(t, "", "./_testc")
}

func TestTestpy(t *testing.T) {
	cltest.FromDir(t, "", "./_testpy")
}
