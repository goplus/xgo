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

package format

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"
)

// -----------------------------------------------------------------------------

func TestFromTestdata(t *testing.T) {
	sel := ""
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = path.Join(dir, "./_testdata")
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testFrom(t, dir+"/"+name, sel)
		})
	}
}

func testFrom(t *testing.T, pkgDir, sel string) {
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}
	log.Println("Formatting", pkgDir)
	file := pkgDir + "/index.xgo"
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := XGostyleSource(src, file)
	if err != nil {
		t.Fatal(err)
	}
	expect, err := os.ReadFile(pkgDir + "/format.expect")
	if err != nil {
		t.Fatal(err)
	}
	diffBytes(t, pkgDir+"/format.result", ret, expect)
}

func diffBytes(t *testing.T, outfile string, dst, src []byte) {
	line := 1
	offs := 0 // line offset
	for i := 0; i < len(dst) && i < len(src); i++ {
		d := dst[i]
		s := src[i]
		if d != s {
			os.WriteFile(outfile, dst, 0644)
			t.Errorf("dst:%d: %s\n", line, dst[offs:])
			t.Errorf("src:%d: %s\n", line, src[offs:])
			return
		}
		if s == '\n' {
			line++
			offs = i + 1
		}
	}
	if len(dst) != len(src) {
		t.Errorf("len(dst) = %d, len(src) = %d\ndst = %q\nsrc = %q", len(dst), len(src), dst, src)
	}
}

// -----------------------------------------------------------------------------
