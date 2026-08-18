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

package runtimeprovider

import (
	"fmt"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/goplus/mod/runtimeprotocol"
	"github.com/goplus/mod/xgomod"
)

func providerArgs(rt *Runtime, act action, policy BuildPolicy, output, finalOutput string, appArgs []string) ([]string, error) {
	if rt.Protocol != protocolV1 {
		return nil, fmt.Errorf("unsupported runtime provider protocol %q", rt.Protocol)
	}
	var pack *runtimeprotocol.Pack
	if rt.PackDir != "" || rt.PackIndex != "" {
		if rt.PackDir == "" || rt.PackIndex == "" {
			return nil, fmt.Errorf("runtime pack metadata must contain both directory and index")
		}
		pack = &runtimeprotocol.Pack{Directory: rt.PackDir, IndexFile: rt.PackIndex}
	}
	request := runtimeprotocol.Request{
		Version: protocolV1,
		Action:  act,
		Project: runtimeprotocol.Project{
			Dir:           rt.ProjectDir,
			File:          rt.ProjectFile,
			ModuleRoot:    rt.ModuleRoot,
			Extension:     rt.ProjectExt,
			FullExtension: rt.ProjectFullExt,
			Pack:          pack,
		},
		ProviderPackage: rt.ProviderPackage,
		ProviderOrigin:  rt.Origin,
		Declaration:     xgomod.FileIdentity{Path: rt.GoxMod, SHA256: rt.GoxModSHA256},
		Graph: runtimeprotocol.Graph{
			GoCommand: rt.Graph.GoCommand,
			WorkDir:   rt.Graph.WorkDir,
			GoWork:    rt.Graph.GoWork,
			Flags:     append([]string(nil), rt.Graph.Flags...),
		},
		BuildFlags:      policy.protocolFlags(),
		ApplicationArgs: append([]string(nil), appArgs...),
	}
	switch act {
	case actionRun:
		if output != "" || finalOutput != "" {
			return nil, fmt.Errorf("run protocol cannot contain output paths")
		}
	case actionBuild:
		if output == "" || finalOutput == "" {
			return nil, fmt.Errorf("build protocol requires output paths")
		}
		if len(appArgs) != 0 {
			return nil, fmt.Errorf("build protocol cannot contain application arguments")
		}
		request.Output = &runtimeprotocol.BuildOutput{Staging: output, Final: finalOutput}
	default:
		return nil, fmt.Errorf("unsupported runtime provider action %q", act)
	}
	return runtimeprotocol.Encode(request)
}

func (p BuildPolicy) protocolFlags() []string {
	flags := make([]string, 0, len(p.Flags)+3)
	if p.Verbose {
		flags = append(flags, "-v=true")
	}
	if p.Trace {
		flags = append(flags, "-x=true")
	}
	if p.KeepWork {
		flags = append(flags, "-work=true")
	}
	return append(flags, p.Flags...)
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
			return ErrRuntimeArgvTooLarge
		}
		envUnits := 1
		for _, item := range env {
			envUnits += len(utf16.Encode([]rune(item))) + 1
		}
		if envUnits > 32_767 {
			return ErrRuntimeArgvTooLarge
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
		return ErrRuntimeArgvTooLarge
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
