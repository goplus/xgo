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
	"path/filepath"
	"strconv"
	"strings"
)

type parsedFlags struct {
	graph    GraphPolicy
	build    BuildPolicy
	rejected []string
}

func (p BuildPolicy) goBuildFlags() []string {
	return p.formatFlags("")
}

func (p BuildPolicy) protocolFlags() []string {
	return p.formatFlags("=true")
}

func (p BuildPolicy) formatFlags(enabledSuffix string) []string {
	flags := make([]string, 0, 5)
	if p.Verbose {
		flags = append(flags, "-v"+enabledSuffix)
	}
	if p.Trace {
		flags = append(flags, "-x"+enabledSuffix)
	}
	if p.KeepWork {
		flags = append(flags, "-work"+enabledSuffix)
	}
	if p.TrimPath {
		flags = append(flags, "-trimpath=true")
	}
	if p.DisableBuildVCS {
		flags = append(flags, "-buildvcs=false")
	}
	return flags
}

func (p GraphPolicy) goFlags() []string {
	flags := make([]string, 0, 3)
	if p.ModMode != "" {
		flags = append(flags, "-mod="+string(p.ModMode))
	}
	if p.ModFile != "" {
		flags = append(flags, "-modfile="+p.ModFile)
	}
	if p.Overlay != "" {
		flags = append(flags, "-overlay="+p.Overlay)
	}
	return flags
}

func (p GraphPolicy) goArgs(command string, args ...string) []string {
	ret := make([]string, 1, 1+len(args)+3)
	ret[0] = command
	ret = append(ret, p.goFlags()...)
	return append(ret, args...)
}

// parseDriverFlags extracts discovery policy and defers rejected flags.
func parseDriverFlags(projectDir, goCommand, goWork, ambient string, cli []string) (parsedFlags, error) {
	ambientArgs, err := splitQuotedFields(ambient)
	if err != nil {
		return parsedFlags{}, fmt.Errorf("invalid GOFLAGS: %w", err)
	}
	all := append(ambientArgs, cli...)
	var ret parsedFlags
	ret.graph.GoCommand = goCommand
	ret.graph.GoWork = goWork
	for _, arg := range all {
		name, value, ok := splitCanonicalFlag(arg)
		if !ok {
			ret.rejected = append(ret.rejected, arg)
			continue
		}
		switch name {
		case "mod":
			mode := modMode(value)
			switch mode {
			case modModeMod, modModeReadonly, modModeVendor:
			default:
				// Keep malformed driver-only policy deferred until a
				// driver-backed project is selected. Legacy Go/XGo commands are
				// still allowed to report the flag error themselves.
				ret.rejected = append(ret.rejected, arg)
				continue
			}
			ret.graph.ModMode = mode
		case "modfile", "overlay":
			if value == "" {
				ret.rejected = append(ret.rejected, arg)
				continue
			}
			path, err := canonicalFlagPath(projectDir, value)
			if err != nil {
				ret.rejected = append(ret.rejected, arg)
				continue
			}
			if name == "modfile" {
				ret.graph.ModFile = path
			} else {
				ret.graph.Overlay = path
				// Discover through the overlay; reject it for matched drivers.
				ret.rejected = append(ret.rejected, arg)
			}
		case "v":
			v, err := strconv.ParseBool(value)
			if err != nil {
				return parsedFlags{}, fmt.Errorf("invalid -v value %q", value)
			}
			ret.build.Verbose = v
		case "x":
			v, err := strconv.ParseBool(value)
			if err != nil {
				return parsedFlags{}, fmt.Errorf("invalid -x value %q", value)
			}
			ret.build.Trace = v
		case "work":
			v, err := strconv.ParseBool(value)
			if err != nil {
				return parsedFlags{}, fmt.Errorf("invalid -work value %q", value)
			}
			ret.build.KeepWork = v
		case "trimpath":
			v, err := strconv.ParseBool(value)
			if err != nil {
				ret.rejected = append(ret.rejected, "-"+name)
				continue
			}
			ret.build.TrimPath = v
		case "buildvcs":
			if value != "false" && value != "auto" {
				ret.rejected = append(ret.rejected, "-"+name)
				continue
			}
			ret.build.DisableBuildVCS = value == "false"
		default:
			ret.rejected = append(ret.rejected, "-"+name)
		}
	}
	return ret, nil
}

func (p parsedFlags) validateDriver() error {
	if len(p.rejected) == 0 {
		return nil
	}
	return fmt.Errorf("driver v1 does not support flag %s", p.rejected[0])
}

func splitCanonicalFlag(arg string) (name, value string, ok bool) {
	if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
		return "", "", false
	}
	arg = strings.TrimPrefix(arg, "-")
	if strings.HasPrefix(arg, "-") {
		arg = strings.TrimPrefix(arg, "-")
	}
	if at := strings.IndexByte(arg, '='); at >= 0 {
		name, value = arg[:at], arg[at+1:]
	} else {
		name, value = arg, "true"
	}
	return name, value, name != ""
}

func canonicalFlagPath(base, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	return abs, nil
}

func isQuotedFieldSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// splitQuotedFields mirrors the Go command's GOFLAGS parser.
func splitQuotedFields(s string) ([]string, error) {
	var fields []string
	for len(s) > 0 {
		for len(s) > 0 && isQuotedFieldSpace(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		if s[0] == '\'' || s[0] == '"' {
			quote := s[0]
			s = s[1:]
			i := 0
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unterminated %c string", quote)
			}
			fields = append(fields, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isQuotedFieldSpace(s[i]) {
			i++
		}
		fields = append(fields, s[:i])
		s = s[i:]
	}
	return fields, nil
}
