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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func graphCommand(ctx context.Context, policy GraphPolicy, dir string, args ...string) *exec.Cmd {
	cmd := commandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = dir
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork)
	return cmd
}

func runGraphCommand(ctx context.Context, policy GraphPolicy, dir, display string, args ...string) (stdout, stderr []byte, err error) {
	cmd := graphCommand(ctx, policy, dir, args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err = cmd.Run()
	if err != nil {
		return out.Bytes(), errOut.Bytes(), commandError(display, err, errOut.String())
	}
	return out.Bytes(), nil, nil
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
	policy, err := parseRuntimeFlags(cwd, goCommand, "off", ambient, cli)
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

// sanitizeGraphFlags removes graph files that do not exist from discovery.
// They remain rejected runtime flags, so a matched runtime still reports the
// policy error through BuildPolicy while a legacy target can continue with its
// normal command path.
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
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("host Go command %q is not a regular file", path)
	}
	return path, nil
}

func goEnvValue(ctx context.Context, goCommand, dir, key string, clearGOFLAGS bool) (string, error) {
	cmd := commandContext(ctx, goCommand, "env", key)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	// GOFLAGS itself must be read from the ambient environment; GOWORK is
	// queried with GOFLAGS cleared so an ambient graph flag cannot affect it.
	if clearGOFLAGS {
		cmd.Env = replaceEnv(cmd.Env, "GOFLAGS", "")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", commandError("go env "+key, err, string(cmdStderr(cmd)))
	}
	return strings.TrimSpace(string(out)), nil
}

func goEnvWithPolicy(ctx context.Context, policy GraphPolicy, dir, key string) (string, error) {
	cmd := graphCommand(ctx, policy, dir, "env", key)
	// Graph flags do not affect go env values and are deliberately never
	// reconstructed into GOFLAGS; all graph operations pass them via argv.
	out, err := cmd.Output()
	if err != nil {
		return "", commandError("go env "+key, err, string(cmdStderr(cmd)))
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

func commandError(name string, err error, stderr string) error {
	if message := strings.TrimSpace(stderr); message != "" {
		return fmt.Errorf("%s: %w: %s", name, err, message)
	}
	return fmt.Errorf("%s: %w", name, err)
}
