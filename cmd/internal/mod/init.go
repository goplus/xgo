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

package mod

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goplus/mod/modload"
	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/env"
)

// gop mod init
var CmdInit = &base.Command{
	UsageLine: "gop mod init [-llgo -tinygo] module-path",
	Short:     "initialize new module in current directory",
}

var (
	flagInit   = &CmdInit.Flag
	flagLLGo   = flagInit.Bool("llgo", false, "use llgo as the compiler")
	flagTinyGo = flagInit.Bool("tinygo", false, "use tinygo as the compiler")
)

func init() {
	CmdInit.Run = runInit
}

func runInit(cmd *base.Command, args []string) {
	err := flagInit.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}
	args = flagInit.Args()
	switch len(args) {
	case 0:
		fatal(`Example usage:
	'gop mod init example.com/m' to initialize a v0 or v1 module
	'gop mod init example.com/m/v2' to initialize a v2 module

Run 'gop help mod init' for more information.`)
	case 1:
	default:
		fatal("gop mod init: too many arguments")
	}

	modPath := args[0]
	mod, err := modload.Create(".", modPath, goMainVer(), env.MainVersion())
	check(err)

	if *flagLLGo {
		mod.AddCompiler("llgo", "1.0")
		mod.AddRequire("github.com/goplus/lib", llgoLibVer(), false)
	} else if *flagTinyGo {
		mod.AddCompiler("tinygo", "0.32")
	}

	err = mod.Save()
	check(err)
}

func goMainVer() string {
	ver := strings.TrimPrefix(runtime.Version(), "go")
	if pos := strings.Index(ver, "."); pos > 0 {
		pos++
		if pos2 := strings.Index(ver[pos:], "."); pos2 > 0 {
			ver = ver[:pos+pos2]
		}
	}
	return ver
}

func llgoLibVer() string {
	if modGop, e1 := xgomod.LoadFrom(filepath.Join(env.XGOROOT(), "go.mod"), ""); e1 == nil {
		if pkg, e2 := modGop.Lookup("github.com/goplus/lib"); e2 == nil {
			return pkg.Real.Version
		}
	}
	return "v0.2.0"
}
