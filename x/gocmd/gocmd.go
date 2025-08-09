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

package gocmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/runsys/xmod/env"
	"github.com/goplus/xgo/x/xgoenv"
)

type XGoEnv = env.XGo

type Config struct {
	XGo   *XGoEnv
	GoCmd string
	Flags []string
	Run   func(cmd *exec.Cmd) error
}

// -----------------------------------------------------------------------------

func doWithArgs(dir, op string, conf *Config, args ...string) (err error) {
	if conf == nil {
		conf = new(Config)
	}
	goCmd := conf.GoCmd
	if goCmd == "" {
		goCmd = Name()
	}
	exargs := make([]string, 1, 16)
	exargs[0] = op
	exargs = appendLdflags(exargs, conf.XGo)
	exargs = append(exargs, conf.Flags...)
	exargs = append(exargs, args...)
	cmd := exec.Command(goCmd, exargs...)
	cmd.Dir = dir
	run := conf.Run
	if run == nil {
		run = runCmd
	}
	return run(cmd)
}

func runCmd(cmd *exec.Cmd) (err error) {
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// -----------------------------------------------------------------------------

const (
	ldFlagVersion   = "-X \"github.com/goplus/xgo/env.buildVersion=%s\""
	ldFlagBuildDate = "-X \"github.com/goplus/xgo/env.buildDate=%s\""
	ldFlagGopRoot   = "-X \"github.com/goplus/xgo/env.defaultXGoRoot=%s\""
)

const (
	ldFlagAll = ldFlagVersion + " " + ldFlagBuildDate + " " + ldFlagGopRoot
)

func loadFlags(env *XGoEnv) string {
	return fmt.Sprintf(ldFlagAll, env.Version, env.BuildDate, env.Root)
}

func appendLdflags(exargs []string, env *XGoEnv) []string {
	if env == nil {
		env = xgoenv.Get()
	}
	return append(exargs, "-ldflags", loadFlags(env))
}

// -----------------------------------------------------------------------------

// Name returns name of the go command.
// It returns value of environment variable `XGO_GOCMD` if not empty.
// If not found, it returns `go`.
func Name() string {
	goCmd := os.Getenv("XGO_GOCMD")
	if goCmd == "" {
		goCmd = "go"
	}
	return goCmd
}

// -----------------------------------------------------------------------------
