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

package gopget

import (
	"fmt"
	"log"
	"os"

	"github.com/runsys/xmod/modcache"
	"github.com/runsys/xmod/modfetch"
	"github.com/runsys/xmod/modload"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/tool"
)

// -----------------------------------------------------------------------------

// Cmd - gop get
var Cmd = &base.Command{
	UsageLine: "gop get [-v] [packages]",
	Short:     `Add dependencies to current module and install them`,
}

var (
	flag = &Cmd.Flag
	_    = flag.Bool("v", false, "print verbose information.")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}
	narg := flag.NArg()
	if narg < 1 {
		log.Fatalln("TODO: not impl")
	}
	for i := 0; i < narg; i++ {
		get(flag.Arg(i))
	}
}

func get(pkgPath string) {
	modBase := ""
	mod, err := modload.Load(".")
	noMod := tool.NotFound(err)
	if !noMod {
		check(err)
		modBase = mod.Path()
	}

	pkgModVer, _, err := modfetch.GetPkg(pkgPath, modBase)
	check(err)
	if noMod {
		return
	}

	pkgModRoot, err := modcache.Path(pkgModVer)
	check(err)

	pkgMod, err := modload.Load(pkgModRoot)
	check(err)

	check(mod.AddRequire(pkgModVer.Path, pkgModVer.Version, pkgMod.HasProject()))
	fmt.Fprintf(os.Stderr, "gop get: added %s %s\n", pkgModVer.Path, pkgModVer.Version)

	check(mod.Save())
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// -----------------------------------------------------------------------------
