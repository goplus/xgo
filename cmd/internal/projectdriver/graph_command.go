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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// A non-empty no-op masks GOFLAGS persisted by go env -w.
const neutralGOFLAGS = "-x=false"

func runCapturedCommand(cmd *exec.Cmd, display string) (stdout, stderr []byte, err error) {
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return out.Bytes(), errOut.Bytes(), commandError(display, err, errOut.String())
	}
	return out.Bytes(), errOut.Bytes(), nil
}

func commandError(name string, err error, stderr string) error {
	if message := strings.TrimSpace(stderr); message != "" {
		return fmt.Errorf("%s: %w: %s", name, err, message)
	}
	return fmt.Errorf("%s: %w", name, err)
}

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
	ambient, err := ambientGOFLAGS(ctx, goCommand, cwd)
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
	// GOWORK is independent of graph flags, which stay in argv to preserve
	// canonical paths without introducing a second quoting grammar.
	goWork, err := goEnvValueWithEnvironment(ctx, goCommand, cwd, "GOWORK", withNeutralGOFLAGS(os.Environ()))
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

func ambientGOFLAGS(ctx context.Context, goCommand, dir string) (string, error) {
	// Read non-empty OS values directly so malformed flags remain deferred.
	if value := os.Getenv("GOFLAGS"); value != "" {
		return value, nil
	}
	return goEnvValue(ctx, goCommand, dir, "GOFLAGS")
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

func goEnvValue(ctx context.Context, goCommand, dir, key string) (string, error) {
	return goEnvValueWithEnvironment(ctx, goCommand, dir, key, os.Environ())
}

func goEnvValueWithEnvironment(ctx context.Context, goCommand, dir, key string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, goCommand, "env", key)
	cmd.Dir = dir
	cmd.Env = env
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
	// Graph policy travels in argv; the no-op masks ambient and persisted
	// GOFLAGS from the authoritative graph command.
	env := withNeutralGOFLAGS(base)
	if goWork != "" {
		env = replaceEnv(env, "GOWORK", goWork)
	}
	return env
}

func withNeutralGOFLAGS(env []string) []string {
	return replaceEnv(env, "GOFLAGS", neutralGOFLAGS)
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
