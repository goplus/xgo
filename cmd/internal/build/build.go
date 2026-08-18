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

// Package build implements the “gop build” command.
package build

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/cmd/internal/runtimeprovider"
	"github.com/goplus/xgo/tool"
	"github.com/goplus/xgo/x/gocmd"
	"github.com/goplus/xgo/x/xgoprojs"
)

// gop build
var Cmd = &base.Command{
	UsageLine: "gop build [-debug -o output] [packages]",
	Short:     "Build XGo files",
}

var (
	flag       = &Cmd.Flag
	flagDebug  = flag.Bool("debug", false, "print debug information")
	flagOutput = flag.String("o", "", "gop build output file")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	pass := base.PassBuildFlags(cmd)
	err := flag.Parse(args)
	if err != nil {
		log.Panicln("parse input arguments failed:", err)
	}

	if *flagDebug {
		gogen.SetDebug(gogen.DbgFlagAll &^ gogen.DbgFlagComments)
		cl.SetDebug(cl.DbgFlagAll)
		cl.SetDisableRecover(true)
	}

	args = flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	proj, args, err := xgoprojs.ParseOne(args...)
	if err != nil {
		log.Panicln(err)
	}
	if len(args) != 0 {
		log.Panicln("too many arguments:", args)
	}
	runtimeResult, runtimeErr := tryRuntime(proj, pass.Args, *flagOutput)
	if runtimeErr != nil {
		fmt.Fprintln(os.Stderr, runtimeErr)
		os.Exit(1)
	}
	if runtimeResult.Handled {
		if runtimeResult.Status.Signaled || runtimeResult.Status.Code != 0 {
			runtimeprovider.Exit(runtimeResult.Status)
		}
		return
	}

	conf, err := tool.NewDefaultConf(".", tool.ConfFlagNoTestFiles, pass.Tags())
	if err != nil {
		log.Panicln("tool.NewDefaultConf:", err)
	}
	defer conf.UpdateCache()

	confCmd := conf.NewGoCmdConf()
	if *flagOutput != "" {
		output, err := filepath.Abs(*flagOutput)
		if err != nil {
			log.Panicln(err)
		}
		confCmd.Flags = []string{"-o", output}
	}
	confCmd.Flags = append(confCmd.Flags, pass.Args...)
	build(proj, conf, confCmd)
}

func tryRuntime(proj xgoprojs.Proj, flags []string, output string) (runtimeprovider.DispatchResult, error) {
	return runtimeprovider.TryBuild(context.Background(), "", proj, flags, output, runtimeprovider.Streams{})
}

func build(proj xgoprojs.Proj, conf *tool.Config, build *gocmd.BuildConfig) {
	const flags = tool.GenFlagPrompt
	var obj string
	var err error
	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		obj = v.Dir
		err = tool.BuildDir(obj, conf, build, flags)
	case *xgoprojs.PkgPathProj:
		obj = v.Path
		err = tool.BuildPkgPath("", v.Path, conf, build, flags)
	case *xgoprojs.FilesProj:
		err = tool.BuildFiles(v.Files, conf, build)
	default:
		log.Panicln("`gop build` doesn't support", reflect.TypeOf(v))
	}
	if tool.NotFound(err) {
		fmt.Fprintf(os.Stderr, "gop build %v: not found\n", obj)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		return
	}
	os.Exit(1)
}

// -----------------------------------------------------------------------------
