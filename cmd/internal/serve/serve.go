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

// Package serve implements the “gop serve command.
package serve

import (
	"context"

	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/x/jsonrpc2"
	"github.com/goplus/xgo/x/jsonrpc2/stdio"
	"github.com/goplus/xgo/x/langserver"
	"github.com/qiniu/x/log"
)

// gop serve
var Cmd = &base.Command{
	UsageLine: "gop serve [flags]",
	Short:     "Serve as an XGo LangServer",
}

var (
	flag        = &Cmd.Flag
	flagVerbose = flag.Bool("v", false, "print verbose information")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(cmd *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}

	if *flagVerbose {
		jsonrpc2.SetDebug(jsonrpc2.DbgFlagCall)
	}

	listener := stdio.Listener(false)
	defer listener.Close()

	server := langserver.NewServer(context.Background(), listener, nil)
	server.Wait()
}

// -----------------------------------------------------------------------------
