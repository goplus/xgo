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
	"path/filepath"
	"sort"
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
	flags := make([]string, 0, len(p.Flags)+3)
	if p.Verbose {
		flags = append(flags, "-v"+enabledSuffix)
	}
	if p.Trace {
		flags = append(flags, "-x"+enabledSuffix)
	}
	if p.KeepWork {
		flags = append(flags, "-work"+enabledSuffix)
	}
	return append(flags, p.Flags...)
}

// parseRuntimeFlags extracts the policy needed for discovery. Rejected build
// flags are retained and reported only after a runtime project is matched, so
// ordinary projects keep their existing behavior.
func parseRuntimeFlags(projectDir, goCommand, goWork, ambient string, cli []string) (parsedFlags, error) {
	ambientArgs, err := splitQuotedFields(ambient)
	if err != nil {
		return parsedFlags{}, fmt.Errorf("invalid GOFLAGS: %w", err)
	}
	all := append(ambientArgs, cli...)
	graphValues := make(map[string]string, 3)
	graphSeen := make(map[string]int, 3)
	var ret parsedFlags
	ret.graph.GoCommand = goCommand
	ret.graph.GoWork = goWork
	for i, arg := range all {
		name, value, ok := splitCanonicalFlag(arg)
		if !ok {
			ret.rejected = append(ret.rejected, arg)
			continue
		}
		switch name {
		case "mod":
			switch value {
			case "mod", "readonly", "vendor":
			default:
				// Keep malformed runtime-only policy deferred until a
				// runtime project is selected. Legacy Go/XGo commands are
				// still allowed to report the flag error themselves.
				ret.rejected = append(ret.rejected, arg)
				continue
			}
			graphValues[name] = value
			graphSeen[name] = i
			ret.graph.ModMode = value
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
			graphValues[name] = path
			graphSeen[name] = i
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
			if err != nil || !v {
				ret.rejected = append(ret.rejected, "-"+name)
				continue
			}
			ret.build.Flags = replaceBuildFlag(ret.build.Flags, name, "-trimpath=true")
		case "buildvcs":
			if value != "false" {
				ret.rejected = append(ret.rejected, "-"+name)
				continue
			}
			ret.build.Flags = replaceBuildFlag(ret.build.Flags, name, "-buildvcs=false")
		default:
			ret.rejected = append(ret.rejected, "-"+name)
		}
	}
	type graphFlag struct {
		name  string
		value string
		pos   int
	}
	ordered := make([]graphFlag, 0, len(graphValues))
	for name, value := range graphValues {
		ordered = append(ordered, graphFlag{name: name, value: value, pos: graphSeen[name]})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].pos < ordered[j].pos })
	for _, flag := range ordered {
		ret.graph.Flags = append(ret.graph.Flags, "-"+flag.name+"="+flag.value)
	}
	return ret, nil
}

func replaceBuildFlag(flags []string, name, value string) []string {
	prefix := "-" + name
	for i, flag := range flags {
		if flag == prefix || strings.HasPrefix(flag, prefix+"=") {
			flags[i] = value
			return flags
		}
	}
	return append(flags, value)
}

func (p parsedFlags) validateRuntime() error {
	if len(p.rejected) == 0 {
		return nil
	}
	return fmt.Errorf("runtime provider v1 does not support flag %s", p.rejected[0])
}

func splitCanonicalFlag(arg string) (name, value string, ok bool) {
	if !strings.HasPrefix(arg, "-") || arg == "-" || strings.HasPrefix(arg, "--") {
		return "", "", false
	}
	arg = strings.TrimPrefix(arg, "-")
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

// splitQuotedFields implements the quoting accepted by GOFLAGS without
// invoking a shell. Quotes are removed and backslash quotes the next byte.
func splitQuotedFields(s string) ([]string, error) {
	var fields []string
	for i := 0; i < len(s); {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
			i++
		}
		if i == len(s) {
			break
		}
		var b strings.Builder
		quote := byte(0)
		for i < len(s) {
			c := s[i]
			if quote == 0 && (c == ' ' || c == '\t' || c == '\r' || c == '\n') {
				break
			}
			switch c {
			case '\'', '"':
				if quote == 0 {
					quote = c
				} else if quote == c {
					quote = 0
				} else {
					b.WriteByte(c)
				}
			case '\\':
				i++
				if i == len(s) {
					return nil, fmt.Errorf("trailing backslash")
				}
				b.WriteByte(s[i])
			default:
				b.WriteByte(c)
			}
			i++
		}
		if quote != 0 {
			return nil, fmt.Errorf("unterminated quote")
		}
		fields = append(fields, b.String())
	}
	return fields, nil
}
