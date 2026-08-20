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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type builtProvider struct {
	path string
	dir  string
	keep bool
}

func (r *Resolver) Run(ctx context.Context, rt *Runtime, appArgs []string, streams Streams) (ProcessStatus, error) {
	policy, err := r.BuildPolicy()
	if err != nil {
		return ProcessStatus{}, err
	}
	return execute(ctx, rt, actionRun, policy, "", "", appArgs, streams)
}

func (r *Resolver) Build(ctx context.Context, rt *Runtime, requestedOutput string, streams Streams) (ProcessStatus, string, error) {
	policy, err := r.BuildPolicy()
	if err != nil {
		return ProcessStatus{}, "", err
	}
	return r.buildWithPolicy(ctx, rt, requestedOutput, policy, streams)
}

func (r *Resolver) buildWithPolicy(ctx context.Context, rt *Runtime, requestedOutput string, policy BuildPolicy, streams Streams) (ProcessStatus, string, error) {
	streams = fillStreams(streams)
	final, err := resolveBuildOutput(r.cwd, requestedOutput, rt.DefaultExecName)
	if err != nil {
		return ProcessStatus{}, "", err
	}
	tx, err := beginOutputTransaction(final, policy.KeepWork)
	if err != nil {
		return ProcessStatus{}, final, err
	}
	defer tx.abort()
	status, err := execute(ctx, rt, actionBuild, policy, tx.staged, final, nil, streams)
	if err != nil || status.Signaled || status.Code != 0 {
		return status, final, err
	}
	if err := tx.commitContext(ctx); err != nil {
		return ProcessStatus{}, final, err
	}
	if policy.KeepWork {
		fmt.Fprintf(streams.Stderr, "XGO_RUNTIME_OUTPUT_WORK=%s\n", tx.dir)
	}
	return successStatus(), final, nil
}

// Install builds one runtime target transactionally into the effective GOBIN.
func (r *Resolver) Install(ctx context.Context, rt *Runtime, streams Streams) (ProcessStatus, string, error) {
	policy, err := r.BuildPolicy()
	if err != nil {
		return ProcessStatus{}, "", err
	}
	bin, err := installBin(ctx, rt.Graph)
	if err != nil {
		return ProcessStatus{}, "", err
	}
	if err := os.MkdirAll(bin, 0755); err != nil {
		return ProcessStatus{}, "", fmt.Errorf("create install directory: %w", err)
	}
	return r.buildWithPolicy(ctx, rt, filepath.Join(bin, rt.DefaultExecName), policy, streams)
}

func installBin(ctx context.Context, graph GraphPolicy) (string, error) {
	cmd := commandContext(ctx, graph.GoCommand, "env", "-json", "GOBIN", "GOPATH")
	cmd.Dir = graph.WorkDir
	cmd.Env = graphEnvironment(os.Environ(), graph.GoWork)
	stdout, err := cmd.Output()
	if err != nil {
		return "", commandError("resolve install directory", err, string(cmdStderr(cmd)))
	}
	values := make(map[string]string)
	if err := json.Unmarshal(stdout, &values); err != nil {
		return "", fmt.Errorf("decode Go install directory: %w", err)
	}
	bin := values["GOBIN"]
	if bin == "" {
		paths := filepath.SplitList(values["GOPATH"])
		if len(paths) == 0 || paths[0] == "" {
			return "", fmt.Errorf("go env GOPATH is empty")
		}
		bin = filepath.Join(paths[0], "bin")
	}
	if !filepath.IsAbs(bin) {
		return "", fmt.Errorf("Go install directory %q is not absolute", bin)
	}
	return filepath.Clean(bin), nil
}

func execute(ctx context.Context, rt *Runtime, act action, policy BuildPolicy, output, finalOutput string, appArgs []string, streams Streams) (ProcessStatus, error) {
	streams = fillStreams(streams)
	provider, err := buildProvider(ctx, rt, policy, streams)
	if err != nil {
		return ProcessStatus{}, err
	}
	defer provider.cleanup()
	args, err := providerArgs(rt, act, policy, output, finalOutput, appArgs)
	if err != nil {
		return ProcessStatus{}, err
	}
	env := providerEnvironment(os.Environ(), rt)
	if err := validateArgv(provider.path, args, env); err != nil {
		return ProcessStatus{}, err
	}
	if policy.Trace {
		fmt.Fprintln(streams.Stderr, redactCommand(provider.path, args))
	}
	cmd := exec.Command(provider.path, args...)
	cmd.Dir = rt.ProjectDir
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = streams.Stdin, streams.Stdout, streams.Stderr
	return runProviderProcess(ctx, cmd)
}

func buildProvider(ctx context.Context, rt *Runtime, policy BuildPolicy, streams Streams) (*builtProvider, error) {
	if goos := os.Getenv("GOOS"); goos != "" && goos != runtime.GOOS {
		return nil, fmt.Errorf("runtime providers are host-only: GOOS=%s, host=%s", goos, runtime.GOOS)
	}
	if goarch := os.Getenv("GOARCH"); goarch != "" && goarch != runtime.GOARCH {
		return nil, fmt.Errorf("runtime providers are host-only: GOARCH=%s, host=%s", goarch, runtime.GOARCH)
	}
	if err := validateProvider(ctx, rt); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "xgo-runtime-provider-")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	provider := &builtProvider{dir: dir, path: filepath.Join(dir, executableName("provider")), keep: policy.KeepWork}
	args := rt.Graph.goArgs("build")
	args = append(args, policy.goBuildFlags()...)
	args = append(args, "-buildmode=exe", "-o", provider.path, rt.ProviderPackage)
	cmd := exec.CommandContext(ctx, rt.Graph.GoCommand, args...)
	cmd.Dir = rt.Graph.WorkDir
	cmd.Env = hostBuildEnvironment(os.Environ(), rt.Graph.GoWork)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = streams.Stdin, streams.Stdout, streams.Stderr
	if policy.Trace {
		fmt.Fprintln(streams.Stderr, redactCommand(rt.Graph.GoCommand, args))
	}
	if err := cmd.Run(); err != nil {
		provider.cleanup()
		return nil, fmt.Errorf("build runtime provider %q: %w", rt.ProviderPackage, err)
	}
	info, err := os.Lstat(provider.path)
	if err != nil {
		provider.cleanup()
		return nil, fmt.Errorf("runtime provider build output: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() == 0 {
		provider.cleanup()
		return nil, fmt.Errorf("runtime provider build did not produce a regular executable")
	}
	if policy.KeepWork {
		fmt.Fprintf(streams.Stderr, "XGO_RUNTIME_PROVIDER_WORK=%s\n", dir)
	}
	return provider, nil
}

func (p *builtProvider) cleanup() {
	if p != nil && !p.keep {
		_ = os.RemoveAll(p.dir)
	}
}

func validateProvider(ctx context.Context, rt *Runtime) error {
	if !moduleContainsPackage(rt.Origin.Selected.Path, rt.ProviderPackage) {
		return fmt.Errorf("runtime provider package %q is outside declaring module %q", rt.ProviderPackage, rt.Origin.Selected.Path)
	}
	args := rt.Graph.goArgs("list", "-json", rt.ProviderPackage)
	cmd := exec.CommandContext(ctx, rt.Graph.GoCommand, args...)
	cmd.Dir = rt.Graph.WorkDir
	cmd.Env = graphEnvironment(os.Environ(), rt.Graph.GoWork)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return commandError("validate runtime provider", err, stderr.String())
	}
	var pkg goListPackage
	if err := json.Unmarshal(stdout.Bytes(), &pkg); err != nil {
		return fmt.Errorf("decode runtime provider package: %w", err)
	}
	if pkg.Error != nil && pkg.Error.Err != "" {
		return fmt.Errorf("runtime provider package: %s", pkg.Error.Err)
	}
	if pkg.ImportPath != rt.ProviderPackage {
		return fmt.Errorf("runtime provider resolved as %q, want %q", pkg.ImportPath, rt.ProviderPackage)
	}
	if pkg.Name != "main" {
		return fmt.Errorf("runtime provider package %q is %q, want command package main", rt.ProviderPackage, pkg.Name)
	}
	if pkg.Module == nil {
		return fmt.Errorf("runtime provider package %q has no module provenance", rt.ProviderPackage)
	}
	module, err := normalizeListedModule(*pkg.Module)
	if err != nil {
		return err
	}
	if !sameResolvedModule(module, rt.Origin) {
		return fmt.Errorf("runtime provider package %q does not match declaring module provenance", rt.ProviderPackage)
	}
	providerDir, err := canonicalExistingDir(pkg.Dir)
	if err != nil {
		return fmt.Errorf("runtime provider directory: %w", err)
	}
	if !pathWithin(rt.Origin.Effective().Dir, providerDir) {
		return fmt.Errorf("runtime provider directory escapes declaring module")
	}
	return nil
}

func hostBuildEnvironment(base []string, goWork string) []string {
	env := graphEnvironment(base, goWork)
	env = replaceEnv(env, "GOOS", runtime.GOOS)
	env = replaceEnv(env, "GOARCH", runtime.GOARCH)
	return env
}

func providerEnvironment(base []string, rt *Runtime) []string {
	env := hostBuildEnvironment(base, rt.Graph.GoWork)
	return replaceEnv(env, runtimeGuardEnv, runtimeGuard(rt.ProjectDir, rt.ProviderPackage))
}

func fillStreams(streams Streams) Streams {
	if streams.Stdin == nil {
		streams.Stdin = os.Stdin
	}
	if streams.Stdout == nil {
		streams.Stdout = os.Stdout
	}
	if streams.Stderr == nil {
		streams.Stderr = os.Stderr
	}
	return streams
}

func processStatus(err error) (ProcessStatus, error) {
	if err == nil {
		return successStatus(), nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return ProcessStatus{}, err
	}
	status := ProcessStatus{Code: exitErr.ExitCode()}
	if signal, ok := exitSignal(exitErr); ok {
		status.Signal, status.Signaled = signal, true
	}
	return status, nil
}

// statusUnlessCanceled rejects a successful provider result after cancellation.
func statusUnlessCanceled(ctx context.Context, status ProcessStatus) (ProcessStatus, error) {
	if !status.Signaled && status.Code == 0 {
		if cause := context.Cause(ctx); cause != nil {
			return ProcessStatus{}, cause
		}
	}
	return status, nil
}

func providerExitStatus(ctx context.Context, waitErr error) (ProcessStatus, error) {
	status, err := processStatus(waitErr)
	if err != nil {
		return ProcessStatus{}, err
	}
	return statusUnlessCanceled(ctx, status)
}
