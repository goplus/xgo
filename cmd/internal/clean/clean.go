/*
 * Copyright (c) 2021-2021 The XGo Authors (xgo.dev). All rights reserved.
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

package clean

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/xgo/cmd/internal/base"
)

const (
	autoGenFilePrefix   = "xgo_autogen_"
	autoGenFileSuffix   = "_autogen.go"
	autoGenGopTestFile  = "gop_autogen_test.go"
	autoGen2GopTestFile = "gop_autogen2_test.go"
	autoGenXgoTestFile  = "xgo_autogen_test.go"
	autoGen2XgoTestFile = "xgo_autogen2_test.go"
)

// -----------------------------------------------------------------------------

func cleanAGFiles(dir string, execAct bool) {
	fis, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		fname := fi.Name()
		if strings.HasPrefix(fname, "_") {
			continue
		}
		if fi.IsDir() {
			pkgDir := filepath.Join(dir, fname)
			if fname == ".xgo" || fname == ".gop" {
				removeGopDir(pkgDir, execAct)
			} else {
				cleanAGFiles(pkgDir, execAct)
			}
			continue
		}
		if strings.HasSuffix(fname, autoGenFileSuffix) || strings.HasPrefix(fname, autoGenFilePrefix) {
			file := filepath.Join(dir, fname)
			fmt.Printf("Cleaning %s ...\n", file)
			if execAct {
				os.Remove(file)
			}
		}
	}
	autogens := []string{
		autoGenGopTestFile, autoGen2GopTestFile,
		autoGenXgoTestFile, autoGen2XgoTestFile,
	}
	for _, autogen := range autogens {
		file := filepath.Join(dir, autogen)
		if _, err = os.Stat(file); err == nil {
			fmt.Printf("Cleaning %s ...\n", file)
			if execAct {
				os.Remove(file)
			}
		}
	}
}

func removeGopDir(dir string, execAct bool) {
	fis, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		fname := fi.Name()
		if strings.HasSuffix(fname, ".xgo.go") || strings.HasSuffix(fname, ".gop.go") {
			genfile := filepath.Join(dir, fname)
			fmt.Printf("Cleaning %s ...\n", genfile)
			if execAct {
				os.Remove(genfile)
			}
		}
	}
	if execAct {
		os.Remove(dir)
	}
}

// -----------------------------------------------------------------------------

// Cmd - gop clean
var Cmd = &base.Command{
	UsageLine: "gop clean [flags] <gopSrcDir>",
	Short:     "Clean all XGo auto generated files",
}

var (
	flag = &Cmd.Flag

	_        = flag.Bool("v", false, "print verbose information.")
	testMode = flag.Bool("t", false, "test mode: display files to clean but don't clean them.")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(_ *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}
	var dir string
	if flag.NArg() == 0 {
		dir = "."
	} else {
		dir = flag.Arg(0)
	}
	cleanAGFiles(dir, !*testMode)
}

// -----------------------------------------------------------------------------
