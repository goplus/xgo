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

package cltest

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/runsys/xmod/modfile"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/parser/fsx"
	"github.com/goplus/xgo/parser/fsx/memfs"
	"github.com/goplus/xgo/scanner"
	"github.com/qiniu/x/test"
)

func spxParserConf() parser.Config {
	return parser.Config{
		ClassKind: func(fname string) (isProj bool, ok bool) {
			ext := modfile.ClassExt(fname)
			c, ok := LookupClass(ext)
			if ok {
				isProj = c.IsProj(ext, fname)
			}
			return
		},
	}
}

func Spx(t *testing.T, gmx, spxcode, expected string) {
	SpxEx(t, gmx, spxcode, expected, "index.tgmx", "bar.tspx")
}

func SpxEx(t *testing.T, gmx, spxcode, expected, gmxfile, spxfile string) {
	SpxWithConf(t, "gopSpxTest", Conf, gmx, spxcode, expected, gmxfile, spxfile, "")
}

func SpxEx2(t *testing.T, gmx, spxcode, expected, gmxfile, spxfile, resultFile string) {
	SpxWithConf(t, "gopSpxTest", Conf, gmx, spxcode, expected, gmxfile, spxfile, resultFile)
}

func SpxWithConf(t *testing.T, name string, conf *cl.Config, gmx, spxcode, expected, gmxfile, spxfile, resultFile string) {
	t.Run(name, func(t *testing.T) {
		fs := memfs.TwoFiles("/foo", spxfile, spxcode, gmxfile, gmx)
		if gmxfile == "" {
			fs = memfs.SingleFile("/foo", spxfile, spxcode)
		}
		SpxTest(t, fs, "/foo", spxParserConf(), conf, expected, resultFile)
	})
}

func SpxTest(t *testing.T, fs fsx.FileSystem, dir string, parseConf parser.Config, clConf *cl.Config, exp any, resultFile string) {
	cl.SetDisableRecover(true)
	defer cl.SetDisableRecover(false)

	pkgs, err := parser.ParseFSDir(Conf.Fset, fs, dir, parseConf)
	if err != nil {
		scanner.PrintError(os.Stderr, err)
		t.Fatal("ParseFSDir:", err)
	}
	bar := pkgs["main"]
	pkg, err := cl.NewPackage("", bar, clConf)
	if err != nil {
		t.Fatal("NewPackage:", err)
	}
	var b bytes.Buffer
	err = pkg.WriteTo(&b, resultFile)
	if err != nil {
		t.Fatal("gogen.WriteTo failed:", err)
	}
	if expected, ok := exp.(string); ok {
		result := b.String()
		if result != expected {
			t.Fatalf("\nResult:\n%s\nExpected:\n%s\n", result, expected)
		}
	} else if test.Diff(t, dir+"/result.txt", b.Bytes(), exp.([]byte)) {
		t.Fatal(dir, ": unexpect result")
	}
}

func SpxErrorFS(t *testing.T, msg string, fs parser.FileSystem) {
	pkgs, err := parser.ParseFSDir(Conf.Fset, fs, "/foo", spxParserConf())
	if err != nil {
		scanner.PrintError(os.Stderr, err)
		t.Fatal("ParseFSDir:", err)
	}
	conf := *Conf
	conf.RelativeBase = "/foo"
	conf.Recorder = nil
	conf.NoFileLine = false
	bar := pkgs["main"]
	_, err = cl.NewPackage("", bar, &conf)
	if err == nil {
		t.Fatal("no error?")
	}
	if ret := err.Error(); ret != msg {
		t.Fatalf("\nError: \"%s\"\nExpected: \"%s\"\n", ret, msg)
	}
}

func SpxErrorEx(t *testing.T, msg, gmx, spxcode, gmxfile, spxfile string) {
	fs := memfs.TwoFiles("/foo", spxfile, spxcode, gmxfile, gmx)
	pkgs, err := parser.ParseFSDir(Conf.Fset, fs, "/foo", spxParserConf())
	if err != nil {
		scanner.PrintError(os.Stderr, err)
		t.Fatal("ParseFSDir:", err)
	}
	conf := *Conf
	conf.RelativeBase = "/foo"
	conf.Recorder = nil
	conf.NoFileLine = false
	bar := pkgs["main"]
	_, err = cl.NewPackage("", bar, &conf)
	if err == nil {
		t.Fatal("no error?")
	}
	if ret := err.Error(); ret != msg {
		t.Fatalf("\nError: \"%s\"\nExpected: \"%s\"\n", ret, msg)
	}
}

func SpxFromDir(t *testing.T, sel, relDir string) {
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
		if !fi.IsDir() || strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testSpxFrom(t, dir+"/"+name, sel)
		})
	}
}

func testSpxFrom(t *testing.T, pkgDir, sel string) {
	if sel != "" && !strings.Contains(pkgDir, sel) {
		return
	}
	log.Println("Parsing", pkgDir)
	parseConf := spxParserConf()
	parseConf.Filter = func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), ".go")
	}
	expect, _ := os.ReadFile(pkgDir + "/out.go")
	SpxTest(t, fsx.Local, pkgDir, parseConf, Conf, expect, "")
}
