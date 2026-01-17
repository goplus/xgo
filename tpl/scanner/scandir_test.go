/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package scanner_test

import (
	"bytes"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/goplus/xgo/tpl/scanner/scannertest"
	"github.com/qiniu/x/test"
)

func testScan(
	t *testing.T, pkgDir string, in []byte,
	expFile string, scan func(w io.Writer, in []byte)) {
	expect, _ := os.ReadFile(pkgDir + "/" + expFile)
	var b bytes.Buffer
	scan(&b, in)
	out := b.Bytes()
	if test.Diff(t, pkgDir+"/result.txt", out, expect) {
		t.Fatal(expFile, ": unexpect result")
	}
}

func testFrom(t *testing.T, pkgDir, sel string) {
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}
	t.Helper()
	log.Println("Scanning", pkgDir)
	in, err := os.ReadFile(pkgDir + "/in.xgo")
	if err != nil {
		t.Fatal("Scanning", pkgDir, "-", err)
	}
	testScan(t, pkgDir, in, "tpl.expect", scannertest.Scan)
	testScan(t, pkgDir, in, "go.expect", scannertest.GoScan)
	testScan(t, pkgDir, in, "gop.expect", scannertest.XGoScan)
}

func testFromDir(t *testing.T, sel, relDir string) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = path.Join(dir, relDir)
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			pkgDir := dir + "/" + name
			testFrom(t, pkgDir, sel)
		})
	}
}

func TestFromTestdata(t *testing.T) {
	testFromDir(t, "", "./_testdata")
}
