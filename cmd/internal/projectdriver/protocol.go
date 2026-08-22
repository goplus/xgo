/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package projectdriver

import (
	"fmt"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/goplus/mod/driverprotocol"
	"github.com/goplus/mod/xgomod"
)

func driverArgs(drv *Driver, act action, policy BuildPolicy, output, finalOutput string, appArgs []string) ([]string, error) {
	var pack *driverprotocol.Pack
	if drv.PackDir != "" || drv.PackIndex != "" {
		pack = &driverprotocol.Pack{Directory: drv.PackDir, IndexFile: drv.PackIndex}
	}
	request := driverprotocol.Request{
		Version: drv.Protocol,
		Action:  act,
		Project: driverprotocol.Project{
			Dir:           drv.ProjectDir,
			File:          drv.ProjectFile,
			ModuleRoot:    drv.ModuleRoot,
			Extension:     drv.ProjectExt,
			FullExtension: drv.ProjectFullExt,
			Pack:          pack,
		},
		DriverPackage: drv.DriverPackage,
		DriverOrigin:  drv.Origin,
		Declaration:   xgomod.FileIdentity{Path: drv.GoxMod, SHA256: drv.GoxModSHA256},
		Graph: driverprotocol.Graph{
			GoCommand: drv.Graph.GoCommand,
			WorkDir:   drv.Graph.WorkDir,
			GoWork:    drv.Graph.GoWork,
			Flags:     drv.Graph.goFlags(),
		},
		BuildFlags:      policy.protocolFlags(),
		ApplicationArgs: append([]string(nil), appArgs...),
	}
	if output != "" || finalOutput != "" {
		request.Output = &driverprotocol.BuildOutput{Staging: output, Final: finalOutput}
	}
	return driverprotocol.Encode(request)
}

func validateArgv(executable string, args, env []string) error {
	if runtime.GOOS == "windows" {
		// CreateProcessW has a 32,767 UTF-16 code-unit command-line limit. Keep
		// headroom for quoting performed by os/exec.
		n := len(utf16.Encode([]rune(executable))) + 1
		for _, arg := range args {
			// os/exec may quote the argument and double every backslash before
			// a quote or the end quote. Two code units per input unit plus the
			// surrounding syntax is a safe upper bound for CommandLineToArgvW.
			n += 2*len(utf16.Encode([]rune(arg))) + 3
		}
		if n > 30_000 {
			return ErrDriverArgvTooLarge
		}
		envUnits := 1
		for _, item := range env {
			envUnits += len(utf16.Encode([]rune(item))) + 1
		}
		if envUnits > 32_767 {
			return ErrDriverArgvTooLarge
		}
		return nil
	}
	// 128 KiB is below the smallest ARG_MAX supported by XGo's host set and
	// accounts for both argv and the inherited environment.
	n := len(executable) + 1
	for _, arg := range args {
		n += len(arg) + 1
	}
	for _, item := range env {
		n += len(item) + 1
	}
	// execve also consumes one native pointer per argv/env entry. Account for
	// 64-bit pointers, the largest supported host representation.
	n += 8 * (len(args) + len(env) + 3)
	if n > 128<<10 {
		return ErrDriverArgvTooLarge
	}
	return nil
}

func redactCommand(executable string, args []string) string {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, quoteForDisplay(executable))
	for _, arg := range args {
		quoted = append(quoted, quoteForDisplay(arg))
	}
	return strings.Join(quoted, " ")
}

func quoteForDisplay(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t\r\n\"'") {
		return value
	}
	return fmt.Sprintf("%q", value)
}
