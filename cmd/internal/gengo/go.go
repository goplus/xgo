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

// Package gengo implements the “gop go” command.
package gengo

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/tool"
	"github.com/goplus/xgo/x/xgoprojs"
	"github.com/qiniu/x/errors"
)

// gop go
var Cmd = &base.Command{
	UsageLine: "gop go [-v] [packages|files]",
	Short:     "Convert XGo code into Go code",
}

var (
	flag                 = &Cmd.Flag
	flagVerbose          = flag.Bool("v", false, "print verbose information")
	flagCheckMode        = flag.Bool("t", false, "do check syntax only, no generate xgo_autogen.go")
	flagSingleMode       = flag.Bool("s", false, "run in single file mode for package")
	flagIgnoreNotatedErr = flag.Bool(
		"ignore-notated-error", false, "ignore notated errors, only available together with -t (check mode)")
	flagTags       = flag.String("tags", "", "a comma-separated list of additional build tags to consider satisfied")
	flagMultiFiles = flag.Bool("multi-files", false, "genarate multi files for package")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Panicln("parse input arguments failed:", err)
	}
	pattern := flag.Args()
	if len(pattern) == 0 {
		pattern = []string{"."}
	}

	projs, err := xgoprojs.ParseAll(pattern...)
	if err != nil {
		log.Panicln("xgoprojs.ParseAll:", err)
	}

	if *flagVerbose {
		gogen.SetDebug(gogen.DbgFlagAll &^ gogen.DbgFlagComments)
		cl.SetDebug(cl.DbgFlagAll)
		cl.SetDisableRecover(true)
	}

	conf, err := tool.NewDefaultConf(".", 0, *flagTags)
	if err != nil {
		log.Panicln("tool.NewDefaultConf:", err)
	}
	defer conf.UpdateCache()

	flags := tool.GenFlagPrintError | tool.GenFlagPrompt
	if *flagCheckMode {
		flags |= tool.GenFlagCheckOnly
		if *flagIgnoreNotatedErr {
			conf.IgnoreNotatedError = true
		}
	}
	if *flagSingleMode {
		flags |= tool.GenFlagSingleFile
	}
	if *flagMultiFiles {
		flags |= tool.GenFlagMultiFiles
	}
	for _, proj := range projs {
		switch v := proj.(type) {
		case *xgoprojs.DirProj:
			_, _, err = tool.GenGoEx(v.Dir, conf, true, flags)
		case *xgoprojs.PkgPathProj:
			_, _, err = tool.GenGoPkgPathEx("", v.Path, conf, true, flags)
		case *xgoprojs.FilesProj:
			_, err = tool.GenGoFiles("", v.Files, conf)
		default:
			log.Panicln("`gop go` doesn't support", reflect.TypeOf(v))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "GenGo failed: %d errors.\n", errorNum(err))
			os.Exit(1)
		}
	}
}

func errorNum(err error) int {
	if e, ok := err.(errors.List); ok {
		return len(e)
	}
	return 1
}

// -----------------------------------------------------------------------------
