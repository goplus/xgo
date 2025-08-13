/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package watch

import (
	"log"
	"path/filepath"

	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/tool"
	"github.com/goplus/xgo/x/fsnotify"
	"github.com/goplus/xgo/x/watcher"
)

// -----------------------------------------------------------------------------

// gop watch
var Cmd = &base.Command{
	UsageLine: "gop watch [-v -gentest] [dir]",
	Short:     "Monitor code changes in an XGo workspace to generate Go files",
}

var (
	flag       = &Cmd.Flag
	verbose    = flag.Bool("v", false, "print verbose information.")
	debug      = flag.Bool("debug", false, "show all debug information.")
	genTestPkg = flag.Bool("gentest", false, "generate test package.")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}

	if *debug {
		fsnotify.SetDebug(fsnotify.DbgFlagAll)
	}
	if *debug || *verbose {
		watcher.SetDebug(watcher.DbgFlagAll)
	}

	args = flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	root, _ := filepath.Abs(args[0])
	log.Println("Watch", root)
	w := watcher.New(root)
	go w.Run()
	for {
		dir := w.Fetch(true)
		log.Println("GenGo", dir)
		_, _, err := tool.GenGo(dir, nil, *genTestPkg)
		if err != nil {
			log.Println(err)
		}
	}
}

// -----------------------------------------------------------------------------
