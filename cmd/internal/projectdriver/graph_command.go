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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func graphCommand(ctx context.Context, policy GraphPolicy, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = dir
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork)
	return cmd
}

func runGraphCommand(ctx context.Context, policy GraphPolicy, dir, display string, args ...string) (stdout, stderr []byte, err error) {
	stdout, stderr, err = runCapturedCommand(graphCommand(ctx, policy, dir, args...), display)
	if err != nil {
		return stdout, stderr, err
	}
	return stdout, nil, nil
}

func preparePolicies(ctx context.Context, cwd string, cli []string) (parsedFlags, error) {
	goCommand, err := hostGoCommand()
	if err != nil {
		return parsedFlags{}, err
	}
	ambient, err := goEnvValue(ctx, goCommand, cwd, "GOFLAGS", false)
	if err != nil {
		return parsedFlags{}, err
	}
	policy, err := parseDriverFlags(cwd, goCommand, "off", ambient, cli)
	if err != nil {
		return parsedFlags{}, err
	}
	policy, err = sanitizeGraphFlags(policy)
	if err != nil {
		return parsedFlags{}, err
	}
	// GOWORK does not depend on -mod/-modfile/-overlay. Keep graph flags out of
	// GOFLAGS entirely: their canonical paths are passed as distinct argv
	// elements to every graph command, which also preserves spaces and Windows
	// path separators without a second quoting grammar.
	goWork, err := goEnvValue(ctx, goCommand, cwd, "GOWORK", true)
	if err != nil {
		return parsedFlags{}, err
	}
	if goWork == "" || goWork == "off" {
		policy.graph.GoWork = "off"
	} else {
		goWork, err = canonicalExistingFile(goWork)
		if err != nil {
			return parsedFlags{}, fmt.Errorf("effective go.work: %w", err)
		}
		policy.graph.GoWork = goWork
	}
	return policy, nil
}

// sanitizeGraphFlags defers missing graph files as driver-only policy errors.
func sanitizeGraphFlags(policy parsedFlags) (parsedFlags, error) {
	if path := policy.graph.ModFile; path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			policy.rejected = append(policy.rejected, "-modfile="+path)
			policy.graph.ModFile = ""
		} else if err != nil {
			return parsedFlags{}, fmt.Errorf("inspect -modfile input %q: %w", path, err)
		}
	}
	if path := policy.graph.Overlay; path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			policy.rejected = append(policy.rejected, "-overlay="+path)
			policy.graph.Overlay = ""
		} else if err != nil {
			return parsedFlags{}, fmt.Errorf("inspect -overlay input %q: %w", path, err)
		}
	}
	return policy, nil
}

func hostGoCommand() (string, error) {
	path, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("host Go command: %w", err)
	}
	canonical, err := canonicalExistingFile(path)
	if err != nil {
		return "", fmt.Errorf("host Go command %q: %w", path, err)
	}
	return canonical, nil
}

func goEnvValue(ctx context.Context, goCommand, dir, key string, clearGOFLAGS bool) (string, error) {
	cmd := exec.CommandContext(ctx, goCommand, "env", key)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	// GOFLAGS itself must be read from the ambient environment; GOWORK is
	// queried with GOFLAGS cleared so an ambient graph flag cannot affect it.
	if clearGOFLAGS {
		cmd.Env = replaceEnv(cmd.Env, "GOFLAGS", "")
	}
	return runGoEnv(cmd, key)
}

// goModForDir returns the physical owner; -modfile changes content, not roots.
func goModForDir(ctx context.Context, policy GraphPolicy, dir string) (string, error) {
	cmd := graphCommand(ctx, policy, dir, "env", "GOMOD")
	return runGoEnv(cmd, "GOMOD")
}

func runGoEnv(cmd *exec.Cmd, key string) (string, error) {
	out, _, err := runCapturedCommand(cmd, "go env "+key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func graphEnvironment(base []string, goWork string) []string {
	// Graph policy is passed as argv. Clearing inherited GOFLAGS prevents an
	// ambient unsupported flag from changing the authoritative graph command.
	env := replaceEnv(base, "GOFLAGS", "")
	if goWork != "" {
		env = replaceEnv(env, "GOWORK", goWork)
	}
	return env
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	ret := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			ret = append(ret, item)
		}
	}
	return append(ret, prefix+value)
}
