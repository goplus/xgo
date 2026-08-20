//go:build !windows

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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

const runtimeProviderHelperEnv = "XGO_RUNTIME_PROVIDER_PROCESS_HELPER"

type providerProcessResult struct {
	status ProcessStatus
	err    error
}

func TestRunProviderProcessForwardsRuntimeSignal(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	cmd := runtimeProviderHelperCommand("handle", ready, received, strconv.Itoa(int(syscall.SIGHUP)))
	boundary := beginRuntimeSignalBoundary(context.Background())
	result := make(chan providerProcessResult, 1)
	go func() {
		status, err := runProviderProcess(boundary.Context(), cmd)
		result <- providerProcessResult{status: status, err: err}
	}()
	pid := waitForHelperPID(t, ready)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	boundary.signals <- syscall.SIGHUP
	got := waitForProviderProcessResult(t, result)
	var cause runtimeSignalCause
	if !errors.As(got.err, &cause) || cause.signal != syscall.SIGHUP {
		t.Fatalf("runProviderProcess() = (%+v, %v), want SIGHUP cancellation cause", got.status, got.err)
	}
	status, err := boundary.Finish(got.status, got.err)
	if err != nil || !status.Signaled || status.Signal != syscall.SIGHUP {
		t.Fatalf("Finish(runProviderProcess()) = (%+v, %v), want SIGHUP status", status, err)
	}
	if signal := waitForHelperSignal(t, received); signal != syscall.SIGHUP {
		t.Fatalf("provider received %v, want SIGHUP", signal)
	}
}

func TestRunProviderProcessReturnsCancellationAfterGracefulExit(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	cmd := runtimeProviderHelperCommand("handle", ready, received, strconv.Itoa(int(syscall.SIGTERM)))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan providerProcessResult, 1)
	go func() {
		status, err := runProviderProcess(ctx, cmd)
		result <- providerProcessResult{status: status, err: err}
	}()
	pid := waitForHelperPID(t, ready)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	cancel()
	got := waitForProviderProcessResult(t, result)
	if !errors.Is(got.err, context.Canceled) {
		t.Fatalf("runProviderProcess() = (%+v, %v), want context cancellation", got.status, got.err)
	}
	if signal := waitForHelperSignal(t, received); signal != syscall.SIGTERM {
		t.Fatalf("provider received %v, want SIGTERM", signal)
	}
}

func TestStatusUnlessCanceledRejectsSuccessfulExitAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	status, err := statusUnlessCanceled(ctx, successStatus())
	if !errors.Is(err, context.Canceled) || status != (ProcessStatus{}) {
		t.Fatalf("statusUnlessCanceled() = (%+v, %v), want cancellation", status, err)
	}
}

func TestRunProviderProcessEscalatesIgnoredCancellation(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	cmd := runtimeProviderHelperCommand("resist", ready, received, strconv.Itoa(int(syscall.SIGTERM)))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan providerProcessResult, 1)
	go func() {
		status, err := runProviderProcess(ctx, cmd)
		result <- providerProcessResult{status: status, err: err}
	}()
	pid := waitForHelperPID(t, ready)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	cancel()
	got := waitForProviderProcessResult(t, result)
	if got.err != nil || !got.status.Signaled || got.status.Signal != syscall.SIGKILL {
		t.Fatalf("runProviderProcess() = (%+v, %v), want SIGKILL after grace period", got.status, got.err)
	}
	if signal := waitForHelperSignal(t, received); signal != syscall.SIGTERM {
		t.Fatalf("provider received %v before escalation, want SIGTERM", signal)
	}
}

func TestRunProviderProcessCleansDescendantsAfterLeaderExit(t *testing.T) {
	dir := t.TempDir()
	childPID := filepath.Join(dir, "child-pid")
	childReady := filepath.Join(dir, "child-ready")
	childSignal := filepath.Join(dir, "child-signal")
	cmd := runtimeProviderHelperCommand("spawn-descendant", childPID, childReady, childSignal)

	status, err := runProviderProcess(context.Background(), cmd)
	pid := waitForHelperPID(t, childPID)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if err != nil || status.Signaled || status.Code != 0 {
		t.Fatalf("runProviderProcess() = (%+v, %v), want successful leader status", status, err)
	}
	if signal := waitForHelperSignal(t, childSignal); signal != syscall.SIGTERM {
		t.Fatalf("descendant received %v before cleanup escalation, want SIGTERM", signal)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("same-group descendant %d still exists after provider return: %v", pid, err)
	}
}

func TestRuntimeProviderProcessHelper(t *testing.T) {
	if os.Getenv(runtimeProviderHelperEnv) != "1" {
		return
	}
	args := runtimeProviderHelperArgs()
	if len(args) == 0 {
		t.Fatal("missing helper mode")
	}
	switch args[0] {
	case "handle":
		runRuntimeProviderSignalHelper(t, args[1:], true)
	case "resist":
		runRuntimeProviderSignalHelper(t, args[1:], false)
	case "spawn-descendant":
		if len(args) != 4 {
			t.Fatalf("spawn-descendant args = %q", args)
		}
		child := runtimeProviderHelperCommand("resist", args[2], args[3], strconv.Itoa(int(syscall.SIGTERM)))
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		if _, err := waitForHelperFile(args[2], 5*time.Second); err != nil {
			_ = child.Process.Kill()
			t.Fatal(err)
		}
		if err := os.WriteFile(args[1], []byte(strconv.Itoa(child.Process.Pid)), 0o600); err != nil {
			_ = child.Process.Kill()
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown helper mode %q", args[0])
	}
}

func runRuntimeProviderSignalHelper(t *testing.T, args []string, exitAfterSignal bool) {
	if len(args) != 3 {
		t.Fatalf("signal helper args = %q", args)
	}
	value, err := strconv.Atoi(args[2])
	if err != nil {
		t.Fatal(err)
	}
	want := syscall.Signal(value)
	signals := make(chan os.Signal, 8)
	signal.Notify(signals, want)
	defer signal.Stop(signals)
	if err := os.WriteFile(args[0], []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	for {
		received := <-signals
		unixSignal, ok := received.(syscall.Signal)
		if !ok {
			continue
		}
		if err := os.WriteFile(args[1], []byte(strconv.Itoa(int(unixSignal))), 0o600); err != nil {
			t.Fatal(err)
		}
		if exitAfterSignal {
			return
		}
	}
}

func runtimeProviderHelperCommand(args ...string) *exec.Cmd {
	commandArgs := []string{"-test.run=^TestRuntimeProviderProcessHelper$", "--"}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command(os.Args[0], commandArgs...)
	cmd.Env = append(os.Environ(), runtimeProviderHelperEnv+"=1")
	return cmd
}

func runtimeProviderHelperArgs() []string {
	for i, arg := range os.Args {
		if arg == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

func waitForProviderProcessResult(t *testing.T, result <-chan providerProcessResult) providerProcessResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("runtime provider process did not return")
		return providerProcessResult{}
	}
}

func waitForHelperPID(t *testing.T, path string) int {
	t.Helper()
	data, err := waitForHelperFile(path, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("parse helper pid %q: %v", data, err)
	}
	return pid
}

func waitForHelperSignal(t *testing.T, path string) syscall.Signal {
	t.Helper()
	data, err := waitForHelperFile(path, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	value, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("parse helper signal %q: %v", data, err)
	}
	return syscall.Signal(value)
}

func waitForHelperFile(path string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for %s", filepath.Base(path))
		}
		time.Sleep(10 * time.Millisecond)
	}
}
