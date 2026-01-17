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

// Package run implements the “gop run” command.
package run

import (
	"fmt"
	"os"
	"reflect"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/tool"
	"github.com/goplus/xgo/x/gocmd"
	"github.com/goplus/xgo/x/xgoprojs"
	"github.com/qiniu/x/log"
)

// gop run
var Cmd = &base.Command{
	UsageLine: "gop run [-nc -asm -quiet -debug -prof] package [arguments...]",
	Short:     "Run an XGo program",
}

var (
	flag        = &Cmd.Flag
	flagAsm     = flag.Bool("asm", false, "generates `asm` code of XGo bytecode backend")
	flagDebug   = flag.Bool("debug", false, "print debug information")
	flagQuiet   = flag.Bool("quiet", false, "don't generate any compiling stage log")
	flagNoChdir = flag.Bool("nc", false, "don't change dir (only for `gop run pkgPath`)")
	flagProf    = flag.Bool("prof", false, "do profile and generate profile report")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	pass := base.PassBuildFlags(cmd)
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}
	if flag.NArg() < 1 {
		cmd.Usage(os.Stderr)
	}

	proj, args, err := xgoprojs.ParseOne(flag.Args()...)
	if err != nil {
		log.Fatalln(err)
	}

	if *flagQuiet {
		log.SetOutputLevel(0x7000)
	} else if *flagDebug {
		gogen.SetDebug(gogen.DbgFlagAll &^ gogen.DbgFlagComments)
		cl.SetDebug(cl.DbgFlagAll)
		cl.SetDisableRecover(true)
	} else if *flagAsm {
		gogen.SetDebug(gogen.DbgFlagInstruction)
	}

	if *flagProf {
		panic("TODO: profile not impl")
	}

	noChdir := *flagNoChdir
	conf, err := tool.NewDefaultConf(".", tool.ConfFlagNoTestFiles, pass.Tags())
	if err != nil {
		log.Panicln("tool.NewDefaultConf:", err)
	}
	defer conf.UpdateCache()

	if !conf.Mod.HasModfile() { // if no go.mod, check XGoDeps
		conf.XGoDeps = new(int)
	}
	confCmd := conf.NewGoCmdConf()
	confCmd.Flags = pass.Args
	run(proj, args, !noChdir, conf, confCmd)
}

func run(proj xgoprojs.Proj, args []string, chDir bool, conf *tool.Config, run *gocmd.RunConfig) {
	const flags = 0
	var obj string
	var err error
	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		obj = v.Dir
		err = tool.RunDir(obj, args, conf, run, flags)
	case *xgoprojs.PkgPathProj:
		obj = v.Path
		err = tool.RunPkgPath(v.Path, args, chDir, conf, run, flags)
	case *xgoprojs.FilesProj:
		err = tool.RunFiles("", v.Files, args, conf, run)
	default:
		log.Panicln("`gop run` doesn't support", reflect.TypeOf(v))
	}
	if tool.NotFound(err) {
		fmt.Fprintf(os.Stderr, "gop run %v: not found\n", obj)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		return
	}
	os.Exit(1)
}

// -----------------------------------------------------------------------------
