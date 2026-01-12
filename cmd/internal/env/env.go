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

package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"

	"github.com/goplus/mod"
	"github.com/goplus/mod/modcache"
	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/env"
	"github.com/goplus/xgo/x/gocmd"
)

// Cmd - gop env
var Cmd = &base.Command{
	UsageLine: "gop env [-json] [var ...]",
	Short:     "Prints XGo environment information",
}

var (
	flag    = &Cmd.Flag
	envJson = flag.Bool("json", false, "prints Go environment information.")
)

func init() {
	Cmd.Run = runCmd
}

func runCmd(_ *base.Command, args []string) {
	err := flag.Parse(args)
	if err != nil {
		log.Fatalln("parse input arguments failed:", err)
	}

	var stdout bytes.Buffer

	cmd := exec.Command("go", "env", "-json")
	cmd.Env = os.Environ()
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		log.Fatalln("run go env failed:", err)
	}

	var xgoEnv map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &xgoEnv); err != nil {
		log.Fatal("decode json of go env failed:", err)
	}

	xgoEnv[env.EnvKeyBuildDate] = env.BuildDate()
	xgoEnv[env.EnvKeyXGoVersion] = env.Version()
	xgoEnv[env.EnvKeyXGoRoot] = env.XGOROOT()
	xgoEnv[env.EnvKeyXGoGoCmd] = gocmd.Name()
	xgoEnv[env.EnvKeyGoModCache] = modcache.GOMODCACHE
	xgoEnv[env.EnvKeyGoXMod], _ = mod.GOXMOD("")
	xgoEnv[env.EnvKeyHome] = env.HOME()

	vars := flag.Args()

	outputEnvVars(xgoEnv, vars, *envJson)
}

func outputEnvVars(gopEnv map[string]any, vars []string, outputJson bool) {
	onlyValues := true

	if len(vars) == 0 {
		onlyValues = false

		vars = make([]string, 0, len(gopEnv))
		for k := range gopEnv {
			vars = append(vars, k)
		}
		sort.Strings(vars)
	} else {
		newEnv := make(map[string]any)
		for _, v := range vars {
			if value, ok := gopEnv[v]; ok {
				newEnv[v] = value
			} else {
				newEnv[v] = ""
			}
		}
		gopEnv = newEnv
	}

	if outputJson {
		b, err := json.Marshal(gopEnv)
		if err != nil {
			log.Fatal("encode json of go env failed:", err)
		}

		var out bytes.Buffer
		json.Indent(&out, b, "", "\t")
		fmt.Println(out.String())
	} else {
		for _, k := range vars {
			v := gopEnv[k]
			if onlyValues {
				fmt.Printf("%v\n", v)
			} else {
				fmt.Printf("%s=\"%v\"\n", k, v)
			}
		}
	}
}
