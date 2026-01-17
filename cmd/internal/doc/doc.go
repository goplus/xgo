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

package doc

import (
	"fmt"
	"go/types"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cl/outline"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/tool"
	"github.com/goplus/xgo/x/xgoenv"
	"github.com/goplus/xgo/x/xgoprojs"
)

// -----------------------------------------------------------------------------

// gop doc
var Cmd = &base.Command{
	UsageLine: "gop doc [-u -all -debug] [pkgPath]",
	Short:     "Show documentation for package or symbol",
}

var (
	flag    = &Cmd.Flag
	withDoc = flag.Bool("all", false, "Show all the documentation for the package.")
	debug   = flag.Bool("debug", false, "Print debug information.")
	unexp   = flag.Bool("u", false, "Show documentation for unexported as well as exported symbols, methods, and fields.")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}

	pattern := flag.Args()
	if len(pattern) == 0 {
		pattern = []string{"."}
	}

	proj, _, err := xgoprojs.ParseOne(pattern...)
	if err != nil {
		log.Panicln("xgoprojs.ParseOne:", err)
	}

	if *debug {
		gogen.SetDebug(gogen.DbgFlagAll &^ gogen.DbgFlagComments)
		cl.SetDebug(cl.DbgFlagAll)
		cl.SetDisableRecover(true)
	}

	xgo := xgoenv.Get()
	conf := &tool.Config{XGo: xgo}
	outlinePkg(proj, conf)
}

func outlinePkg(proj xgoprojs.Proj, conf *tool.Config) {
	var obj string
	var out outline.Package
	var err error
	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		obj = v.Dir
		out, err = tool.Outline(obj, conf)
	case *xgoprojs.PkgPathProj:
		obj = v.Path
		out, err = tool.OutlinePkgPath("", obj, conf, true)
	default:
		log.Panicln("`gop doc` doesn't support", reflect.TypeOf(v))
	}
	if tool.NotFound(err) {
		fmt.Fprintf(os.Stderr, "gop doc %v: not Go/XGo files found\n", obj)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		outlineDoc(out.Outline(*unexp), *unexp, *withDoc)
	}
}

const (
	indent = "    "
	ln     = "\n"
)

func outlineDoc(out *outline.All, all, withDoc bool) {
	pkg := out.Pkg()
	fmt.Printf("package %s // import %s\n\n", pkg.Name(), strconv.Quote(pkg.Path()))
	if withDoc && len(out.Consts) > 0 {
		fmt.Print("CONSTANTS\n\n")
	}
	for _, o := range out.Consts {
		printObject(pkg, o, withDoc)
	}
	if withDoc && len(out.Vars) > 0 {
		fmt.Print("VARIABLES\n\n")
	}
	for _, o := range out.Vars {
		printObject(pkg, o, withDoc)
	}
	if withDoc && len(out.Funcs) > 0 {
		fmt.Print("FUNCTIONS\n\n")
	}
	for _, fn := range out.Funcs {
		printObject(pkg, fn, withDoc)
	}
	if withDoc && len(out.Types) > 0 {
		fmt.Print("TYPES\n\n")
	}
	for _, t := range out.Types {
		if !(all || t.IsUsed()) {
			continue
		}
		typName := t.ObjWith(all)
		fmt.Print(objectString(pkg, typName), ln)
		for _, o := range t.Consts {
			fmt.Print(indent, constShortString(o.Const), ln)
		}
		if withDoc {
			printDoc(t)
		}
		printFuncsForType(pkg, t.Creators, withDoc)
		printFuncsForType(pkg, t.XGotFuncs, withDoc)
		printFuncsForType(pkg, t.Helpers, withDoc)
		if !typName.IsAlias() {
			typ := t.Type()
			if named, ok := typ.CheckNamed(out.Package); ok {
				for _, fn := range named.Methods() {
					if o := fn.Obj(); all || o.Exported() {
						if withDoc {
							fmt.Print(objectString(pkg, o), ln)
							printDoc(fn)
						} else {
							fmt.Print(indent, objectString(pkg, o), ln)
						}
					}
				}
			}
		}
	}
}

type object interface {
	Obj() types.Object
	Doc() string
}

func printObject(pkg *types.Package, o object, withDoc bool) {
	fmt.Print(objectString(pkg, o.Obj()), ln)
	if withDoc {
		printDoc(o)
	}
}

func printDoc(o object) {
	if doc := o.Doc(); doc != "" {
		fmt.Print(indent, strings.ReplaceAll(doc, "\n", "\n"+indent), ln)
	} else {
		fmt.Println()
	}
}

func printFuncsForType(pkg *types.Package, fns []outline.Func, withDoc bool) {
	for _, fn := range fns {
		if withDoc {
			printObject(pkg, fn, true)
		} else {
			fmt.Print(indent, objectString(pkg, fn.Obj()), ln)
		}
	}
}

func objectString(pkg *types.Package, obj types.Object) string {
	if name, fn, ok := outline.CheckOverload(obj); ok {
		obj = types.NewFunc(fn.Pos(), fn.Pkg(), name, fn.Type().(*types.Signature))
	}
	return types.ObjectString(obj, qualifier(pkg))
}

func constShortString(obj *types.Const) string {
	return "const " + obj.Name()
}

func qualifier(pkg *types.Package) types.Qualifier {
	return func(other *types.Package) string {
		if pkg == other {
			return "" // same package; unqualified
		}
		return other.Name()
	}
}

// -----------------------------------------------------------------------------
