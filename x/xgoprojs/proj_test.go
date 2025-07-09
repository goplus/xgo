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

package xgoprojs

import (
	"testing"
)

// -----------------------------------------------------------------------------

func TestIsLocal(t *testing.T) {
	if !isLocal(".") || !isLocal("/") {
		t.Fatal(`isLocal(".") || isLocal("/")`)
	}
	if !isLocal("c:/foo") {
		t.Fatal(`isLocal("c:/foo")`)
	}
	if !isLocal("C:/foo") {
		t.Fatal(`isLocal("C:/foo")`)
	}
	if isLocal("") {
		t.Fatal(`isLocal("")`)
	}
}

func TestParseOne(t *testing.T) {
	proj, next, err := ParseOne("proj.go", "proj_test.go", "abc")
	if err != nil || len(next) != 1 || next[0] != "abc" {
		t.Fatal("ParseOne failed:", proj, next, err)
	}
}

func TestParseAll_wildcard1(t *testing.T) {
	projs, err := ParseAll("proj_test.go")
	if err != nil || len(projs) != 1 {
		t.Fatal("ParseAll failed:", projs, err)
	}
	if proj, ok := projs[0].(*FilesProj); !ok || len(proj.Files) != 1 || proj.Files[0] != "proj_test.go" {
		t.Fatal("ParseAll failed:", projs)
	}
}

func TestParseAll_wildcard2(t *testing.T) {
	projs, err := ParseAll("../xgoenv/env.go")
	if err != nil || len(projs) != 1 {
		t.Fatal("ParseAll failed:", projs, err)
	}
	if proj, ok := projs[0].(*FilesProj); !ok || len(proj.Files) != 1 || proj.Files[0] != "../xgoenv/env.go" {
		t.Fatal("ParseAll failed:", projs)
	}
}

func TestParseAll_multiFiles(t *testing.T) {
	projs, err := ParseAll("proj.go", "proj_test.go")
	if err != nil || len(projs) != 1 {
		t.Fatal("ParseAll failed:", projs, err)
	}
	if proj, ok := projs[0].(*FilesProj); !ok || len(proj.Files) != 2 || proj.Files[0] != "proj.go" {
		t.Fatal("ParseAll failed:", proj)
	}
	projs[0].projObj()
}

func TestParseAll_multiProjs(t *testing.T) {
	projs, err := ParseAll("a/...", "./a/...", "/a")
	if err != nil || len(projs) != 3 {
		t.Fatal("ParseAll failed:", projs, err)
	}
	if proj, ok := projs[0].(*PkgPathProj); !ok || proj.Path != "a/..." {
		t.Fatal("ParseAll failed:", proj)
	}
	if proj, ok := projs[1].(*DirProj); !ok || proj.Dir != "./a/..." {
		t.Fatal("ParseAll failed:", proj)
	}
	if proj, ok := projs[2].(*DirProj); !ok || proj.Dir != "/a" {
		t.Fatal("ParseAll failed:", proj)
	}
	for _, proj := range projs {
		proj.projObj()
	}
}

func TestParseAllErr(t *testing.T) {
	_, err := ParseAll("a/...", "./a/...", "/a", "proj_test.go")
	if err != ErrMixedFilesProj {
		t.Fatal("ParseAll:", err)
	}
}

// -----------------------------------------------------------------------------
