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

package projectdriver

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

const driverHelperEnv = "XGO_DRIVER_PROCESS_HELPER"

type driverProcessResult struct {
	status ProcessStatus
	err    error
}

func TestRunDriverProcessForwardsDriverSignal(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	cmd := driverHelperCommand("handle", ready, received, strconv.Itoa(int(syscall.SIGHUP)))
	boundary := beginDriverSignalBoundary(context.Background())
	result := make(chan driverProcessResult, 1)
	go func() {
		status, err := runDriverProcess(boundary.Context(), cmd)
		result <- driverProcessResult{status: status, err: err}
	}()
	pid := waitForHelperPID(t, ready)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	boundary.signals <- syscall.SIGHUP
	got := waitForDriverProcessResult(t, result)
	var cause driverSignalCause
	if !errors.As(got.err, &cause) || cause.signal != syscall.SIGHUP {
		t.Fatalf("runDriverProcess() = (%+v, %v), want SIGHUP cancellation cause", got.status, got.err)
	}
	status, err := boundary.Finish(got.status, got.err)
	if err != nil || !status.Signaled || status.Signal != syscall.SIGHUP {
		t.Fatalf("Finish(runDriverProcess()) = (%+v, %v), want SIGHUP status", status, err)
	}
	if signal := waitForHelperSignal(t, received); signal != syscall.SIGHUP {
		t.Fatalf("driver received %v, want SIGHUP", signal)
	}
}

func TestRunDriverProcessReturnsCancellationAfterGracefulExit(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	cmd := driverHelperCommand("handle", ready, received, strconv.Itoa(int(syscall.SIGTERM)))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan driverProcessResult, 1)
	go func() {
		status, err := runDriverProcess(ctx, cmd)
		result <- driverProcessResult{status: status, err: err}
	}()
	pid := waitForHelperPID(t, ready)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	cancel()
	got := waitForDriverProcessResult(t, result)
	if !errors.Is(got.err, context.Canceled) {
		t.Fatalf("runDriverProcess() = (%+v, %v), want context cancellation", got.status, got.err)
	}
	if signal := waitForHelperSignal(t, received); signal != syscall.SIGTERM {
		t.Fatalf("driver received %v, want SIGTERM", signal)
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

func TestRunDriverProcessEscalatesIgnoredCancellation(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	cmd := driverHelperCommand("resist", ready, received, strconv.Itoa(int(syscall.SIGTERM)))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan driverProcessResult, 1)
	go func() {
		status, err := runDriverProcess(ctx, cmd)
		result <- driverProcessResult{status: status, err: err}
	}()
	pid := waitForHelperPID(t, ready)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	cancel()
	got := waitForDriverProcessResult(t, result)
	if got.err != nil || !got.status.Signaled || got.status.Signal != syscall.SIGKILL {
		t.Fatalf("runDriverProcess() = (%+v, %v), want SIGKILL after grace period", got.status, got.err)
	}
	if signal := waitForHelperSignal(t, received); signal != syscall.SIGTERM {
		t.Fatalf("driver received %v before escalation, want SIGTERM", signal)
	}
}

func TestRunDriverProcessCleansDescendantsAfterLeaderExit(t *testing.T) {
	dir := t.TempDir()
	childPID := filepath.Join(dir, "child-pid")
	childReady := filepath.Join(dir, "child-ready")
	childSignal := filepath.Join(dir, "child-signal")
	cmd := driverHelperCommand("spawn-descendant", childPID, childReady, childSignal)

	status, err := runDriverProcess(context.Background(), cmd)
	pid := waitForHelperPID(t, childPID)
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if err != nil || status.Signaled || status.Code != 0 {
		t.Fatalf("runDriverProcess() = (%+v, %v), want successful leader status", status, err)
	}
	if signal := waitForHelperSignal(t, childSignal); signal != syscall.SIGTERM {
		t.Fatalf("descendant received %v before cleanup escalation, want SIGTERM", signal)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("same-group descendant %d still exists after driver return: %v", pid, err)
	}
}

func TestDriverProcessHelper(t *testing.T) {
	if os.Getenv(driverHelperEnv) != "1" {
		return
	}
	args := driverHelperArgs()
	if len(args) == 0 {
		t.Fatal("missing helper mode")
	}
	switch args[0] {
	case "handle":
		runDriverSignalHelper(t, args[1:], true)
	case "resist":
		runDriverSignalHelper(t, args[1:], false)
	case "spawn-descendant":
		if len(args) != 4 {
			t.Fatalf("spawn-descendant args = %q", args)
		}
		child := driverHelperCommand("resist", args[2], args[3], strconv.Itoa(int(syscall.SIGTERM)))
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

func runDriverSignalHelper(t *testing.T, args []string, exitAfterSignal bool) {
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

func driverHelperCommand(args ...string) *exec.Cmd {
	commandArgs := []string{"-test.run=^TestDriverProcessHelper$", "--"}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command(os.Args[0], commandArgs...)
	cmd.Env = append(os.Environ(), driverHelperEnv+"=1")
	return cmd
}

func driverHelperArgs() []string {
	for i, arg := range os.Args {
		if arg == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

func waitForDriverProcessResult(t *testing.T, result <-chan driverProcessResult) driverProcessResult {
	t.Helper()
	select {
	case got := <-result:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("driver process did not return")
		return driverProcessResult{}
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
