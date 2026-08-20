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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type builtDriver struct {
	path string
	dir  string
	keep bool
}

func (r *Resolver) Run(ctx context.Context, drv *Driver, appArgs []string, streams Streams) (ProcessStatus, error) {
	policy, err := r.BuildPolicy()
	if err != nil {
		return ProcessStatus{}, err
	}
	return execute(ctx, drv, actionRun, policy, "", "", appArgs, streams)
}

func (r *Resolver) Build(ctx context.Context, drv *Driver, requestedOutput string, streams Streams) (ProcessStatus, string, error) {
	policy, err := r.BuildPolicy()
	if err != nil {
		return ProcessStatus{}, "", err
	}
	return r.buildWithPolicy(ctx, drv, requestedOutput, policy, streams)
}

func (r *Resolver) buildWithPolicy(ctx context.Context, drv *Driver, requestedOutput string, policy BuildPolicy, streams Streams) (ProcessStatus, string, error) {
	streams = fillStreams(streams)
	final, err := resolveBuildOutput(r.cwd, requestedOutput, drv.DefaultExecName)
	if err != nil {
		return ProcessStatus{}, "", err
	}
	tx, err := beginOutputTransaction(final, policy.KeepWork)
	if err != nil {
		return ProcessStatus{}, final, err
	}
	defer tx.abort()
	status, err := execute(ctx, drv, actionBuild, policy, tx.staged, final, nil, streams)
	if err != nil || status.Signaled || status.Code != 0 {
		return status, final, err
	}
	if err := tx.commitContext(ctx); err != nil {
		return ProcessStatus{}, final, err
	}
	if policy.KeepWork {
		fmt.Fprintf(streams.Stderr, "XGO_DRIVER_OUTPUT_WORK=%s\n", tx.dir)
	}
	return successStatus(), final, nil
}

// Install builds one driver-backed target transactionally into the effective GOBIN.
func (r *Resolver) Install(ctx context.Context, drv *Driver, streams Streams) (ProcessStatus, string, error) {
	policy, err := r.BuildPolicy()
	if err != nil {
		return ProcessStatus{}, "", err
	}
	bin, err := installBin(ctx, drv.Graph)
	if err != nil {
		return ProcessStatus{}, "", err
	}
	if err := os.MkdirAll(bin, 0755); err != nil {
		return ProcessStatus{}, "", fmt.Errorf("create install directory: %w", err)
	}
	return r.buildWithPolicy(ctx, drv, filepath.Join(bin, drv.DefaultExecName), policy, streams)
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

func execute(ctx context.Context, drv *Driver, act action, policy BuildPolicy, output, finalOutput string, appArgs []string, streams Streams) (ProcessStatus, error) {
	streams = fillStreams(streams)
	driver, err := buildDriver(ctx, drv, policy, streams)
	if err != nil {
		return ProcessStatus{}, err
	}
	defer driver.cleanup()
	args, err := driverArgs(drv, act, policy, output, finalOutput, appArgs)
	if err != nil {
		return ProcessStatus{}, err
	}
	env := driverEnvironment(os.Environ(), drv)
	if err := validateArgv(driver.path, args, env); err != nil {
		return ProcessStatus{}, err
	}
	if policy.Trace {
		fmt.Fprintln(streams.Stderr, redactCommand(driver.path, args))
	}
	cmd := exec.Command(driver.path, args...)
	cmd.Dir = drv.ProjectDir
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = streams.Stdin, streams.Stdout, streams.Stderr
	return runDriverProcess(ctx, cmd)
}

func buildDriver(ctx context.Context, drv *Driver, policy BuildPolicy, streams Streams) (*builtDriver, error) {
	if goos := os.Getenv("GOOS"); goos != "" && goos != runtime.GOOS {
		return nil, fmt.Errorf("drivers are host-only: GOOS=%s, host=%s", goos, runtime.GOOS)
	}
	if goarch := os.Getenv("GOARCH"); goarch != "" && goarch != runtime.GOARCH {
		return nil, fmt.Errorf("drivers are host-only: GOARCH=%s, host=%s", goarch, runtime.GOARCH)
	}
	if err := validateDriver(ctx, drv); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "xgo-driver-")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	driver := &builtDriver{dir: dir, path: filepath.Join(dir, executableName("driver")), keep: policy.KeepWork}
	args := drv.Graph.goArgs("build")
	args = append(args, policy.goBuildFlags()...)
	args = append(args, "-buildmode=exe", "-o", driver.path, drv.DriverPackage)
	cmd := exec.CommandContext(ctx, drv.Graph.GoCommand, args...)
	cmd.Dir = drv.Graph.WorkDir
	cmd.Env = hostBuildEnvironment(os.Environ(), drv.Graph.GoWork)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = streams.Stdin, streams.Stdout, streams.Stderr
	if policy.Trace {
		fmt.Fprintln(streams.Stderr, redactCommand(drv.Graph.GoCommand, args))
	}
	if err := cmd.Run(); err != nil {
		driver.cleanup()
		return nil, fmt.Errorf("build driver %q: %w", drv.DriverPackage, err)
	}
	info, err := os.Lstat(driver.path)
	if err != nil {
		driver.cleanup()
		return nil, fmt.Errorf("driver build output: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() == 0 {
		driver.cleanup()
		return nil, fmt.Errorf("driver build did not produce a regular executable")
	}
	if policy.KeepWork {
		fmt.Fprintf(streams.Stderr, "XGO_DRIVER_WORK=%s\n", dir)
	}
	return driver, nil
}

func (p *builtDriver) cleanup() {
	if p != nil && !p.keep {
		_ = os.RemoveAll(p.dir)
	}
}

func validateDriver(ctx context.Context, drv *Driver) error {
	if !moduleContainsPackage(drv.Origin.Selected.Path, drv.DriverPackage) {
		return fmt.Errorf("driver package %q is outside declaring module %q", drv.DriverPackage, drv.Origin.Selected.Path)
	}
	args := drv.Graph.goArgs("list", "-json", drv.DriverPackage)
	cmd := exec.CommandContext(ctx, drv.Graph.GoCommand, args...)
	cmd.Dir = drv.Graph.WorkDir
	cmd.Env = graphEnvironment(os.Environ(), drv.Graph.GoWork)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return commandError("validate driver", err, stderr.String())
	}
	var pkg goListPackage
	if err := json.Unmarshal(stdout.Bytes(), &pkg); err != nil {
		return fmt.Errorf("decode driver package: %w", err)
	}
	if pkg.Error != nil && pkg.Error.Err != "" {
		return fmt.Errorf("driver package: %s", pkg.Error.Err)
	}
	if pkg.ImportPath != drv.DriverPackage {
		return fmt.Errorf("driver resolved as %q, want %q", pkg.ImportPath, drv.DriverPackage)
	}
	if pkg.Name != "main" {
		return fmt.Errorf("driver package %q is %q, want command package main", drv.DriverPackage, pkg.Name)
	}
	if pkg.Module == nil {
		return fmt.Errorf("driver package %q has no module provenance", drv.DriverPackage)
	}
	module, err := normalizeListedModule(*pkg.Module)
	if err != nil {
		return err
	}
	if !sameResolvedModule(module, drv.Origin) {
		return fmt.Errorf("driver package %q does not match declaring module provenance", drv.DriverPackage)
	}
	driverDir, err := canonicalExistingDir(pkg.Dir)
	if err != nil {
		return fmt.Errorf("driver directory: %w", err)
	}
	if !pathWithin(drv.Origin.Effective().Dir, driverDir) {
		return fmt.Errorf("driver directory escapes declaring module")
	}
	return nil
}

func hostBuildEnvironment(base []string, goWork string) []string {
	env := graphEnvironment(base, goWork)
	env = replaceEnv(env, "GOOS", runtime.GOOS)
	env = replaceEnv(env, "GOARCH", runtime.GOARCH)
	return env
}

func driverEnvironment(base []string, drv *Driver) []string {
	env := hostBuildEnvironment(base, drv.Graph.GoWork)
	return replaceEnv(env, driverGuardEnv, driverGuard(drv.ProjectDir, drv.DriverPackage))
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

// statusUnlessCanceled rejects a successful driver result after cancellation.
func statusUnlessCanceled(ctx context.Context, status ProcessStatus) (ProcessStatus, error) {
	if !status.Signaled && status.Code == 0 {
		if cause := context.Cause(ctx); cause != nil {
			return ProcessStatus{}, cause
		}
	}
	return status, nil
}

func driverExitStatus(ctx context.Context, waitErr error) (ProcessStatus, error) {
	status, err := processStatus(waitErr)
	if err != nil {
		return ProcessStatus{}, err
	}
	return statusUnlessCanceled(ctx, status)
}
