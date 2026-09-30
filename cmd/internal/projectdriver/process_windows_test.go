//go:build windows

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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestConfigureSuspendedDriverPreservesFlags(t *testing.T) {
	cmd := exec.Command("driver.exe")
	cmd.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	original := cmd.SysProcAttr
	configureSuspendedDriver(cmd)
	if cmd.SysProcAttr == original {
		t.Fatal("configureSuspendedDriver mutated caller-owned SysProcAttr")
	}
	want := uint32(windows.CREATE_NO_WINDOW | windows.CREATE_SUSPENDED)
	if got := cmd.SysProcAttr.CreationFlags; got != want {
		t.Fatalf("creation flags = %#x, want %#x", got, want)
	}
}

func TestRunDriverProcessWindows(t *testing.T) {
	if os.Getenv("XGO_TEST_WINDOWS_DRIVER_CHILD") == "1" {
		inJob, err := currentProcessInJob()
		if err != nil || !inJob {
			fmt.Fprintf(os.Stderr, "job=%v err=%v\n", inJob, err)
			os.Exit(91)
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			os.Exit(92)
		}
		cwd, err := os.Getwd()
		if err != nil {
			os.Exit(93)
		}
		fmt.Printf("stdin=%s|env=%s|cwd=%s|args=%s", input, os.Getenv("XGO_TEST_VALUE"), filepath.Base(cwd), strings.Join(os.Args[len(os.Args)-2:], ","))
		fmt.Fprint(os.Stderr, "driver-stderr")
		return
	}

	work := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunDriverProcessWindows$", "--", "a b", `c"d`)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "XGO_TEST_WINDOWS_DRIVER_CHILD=1", "XGO_TEST_VALUE=present")
	cmd.Stdin = strings.NewReader("driver-stdin")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	status, err := runDriverProcess(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if status.Code != 0 || status.Signaled {
		t.Fatalf("status = %#v", status)
	}
	for _, want := range []string{"stdin=driver-stdin", "env=present", "cwd=" + filepath.Base(work), `args=a b,c"d`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout %q does not contain %q", stdout.String(), want)
		}
	}
	if stderr.String() != "driver-stderr" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunDriverProcessWindowsCleansDescendantAfterLeaderExit(t *testing.T) {
	testWindowsDriverDescendant(t, false)
}

func TestRunDriverProcessWindowsCleansDescendantOnCancellation(t *testing.T) {
	testWindowsDriverDescendant(t, true)
}

func testWindowsDriverDescendant(t *testing.T, cancelDriver bool) {
	t.Helper()
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "descendant-pid")
	readyFile := filepath.Join(dir, "descendant-ready")
	releaseFile := filepath.Join(dir, "release-leader")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunDriverProcessWindowsDescendantHelper$")
	cmd.Env = append(os.Environ(),
		windowsDriverTreeMode+"=leader",
		windowsDriverTreePID+"="+pidFile,
		windowsDriverTreeReady+"="+readyFile,
		windowsDriverTreeRelease+"="+releaseFile,
	)
	result := make(chan windowsDriverResult, 1)
	go func() {
		status, err := runDriverProcess(ctx, cmd)
		result <- windowsDriverResult{status: status, err: err}
	}()

	pid := waitForWindowsDriverPID(t, pidFile)
	waitForWindowsDriverFile(t, readyFile)
	process, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = windows.TerminateProcess(process, 1)
		_ = windows.CloseHandle(process)
	})
	if cancelDriver {
		cancel()
	} else if err := os.WriteFile(releaseFile, []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}

	var got windowsDriverResult
	select {
	case got = <-result:
	case <-time.After(15 * time.Second):
		t.Fatal("driver process did not exit")
	}
	if cancelDriver {
		if got.err != nil && !errors.Is(got.err, context.Canceled) {
			t.Fatalf("runDriverProcess() cancellation error = %v", got.err)
		}
		if got.err == nil && got.status == successStatus() {
			t.Fatal("canceled driver reported success")
		}
	} else if got.err != nil || got.status != successStatus() {
		t.Fatalf("runDriverProcess() = (%+v, %v), want success", got.status, got.err)
	}
	if result, err := windows.WaitForSingleObject(process, 0); err != nil {
		t.Fatal(err)
	} else if result != windows.WAIT_OBJECT_0 {
		t.Fatalf("descendant process remains after driver return (wait result %#x)", result)
	}
}

type windowsDriverResult struct {
	status ProcessStatus
	err    error
}

const (
	windowsDriverTreeMode    = "XGO_TEST_WINDOWS_DRIVER_TREE_MODE"
	windowsDriverTreePID     = "XGO_TEST_WINDOWS_DRIVER_TREE_PID"
	windowsDriverTreeReady   = "XGO_TEST_WINDOWS_DRIVER_TREE_READY"
	windowsDriverTreeRelease = "XGO_TEST_WINDOWS_DRIVER_TREE_RELEASE"
)

func TestRunDriverProcessWindowsDescendantHelper(t *testing.T) {
	switch os.Getenv(windowsDriverTreeMode) {
	case "leader":
		cmd := exec.Command(os.Args[0], "-test.run=^TestRunDriverProcessWindowsDescendantHelper$")
		cmd.Env = append(os.Environ(), windowsDriverTreeMode+"=descendant")
		if err := cmd.Start(); err != nil {
			os.Exit(91)
		}
		pid := strconv.Itoa(cmd.Process.Pid)
		if err := os.WriteFile(os.Getenv(windowsDriverTreePID), []byte(pid), 0o600); err != nil {
			os.Exit(92)
		}
		if !waitForWindowsDriverPath(os.Getenv(windowsDriverTreeRelease), 15*time.Second) {
			os.Exit(93)
		}
		return
	case "descendant":
		inJob, err := currentProcessInJob()
		if err != nil || !inJob {
			os.Exit(94)
		}
		if err := os.WriteFile(os.Getenv(windowsDriverTreeReady), []byte("ready"), 0o600); err != nil {
			os.Exit(95)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
}

func waitForWindowsDriverPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var data []byte
	for time.Now().Before(deadline) {
		var err error
		data, err = os.ReadFile(path)
		if err == nil {
			if pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data))); parseErr == nil && pid > 0 {
				return pid
			}
		} else if !os.IsNotExist(err) {
			t.Fatalf("read descendant PID %q: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("invalid descendant PID %q", data)
	return 0
}

func waitForWindowsDriverFile(t *testing.T, path string) {
	t.Helper()
	if !waitForWindowsDriverPath(path, 15*time.Second) {
		t.Fatalf("wait for %q", path)
	}
}

func waitForWindowsDriverPath(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return true
		} else if !os.IsNotExist(err) || time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
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

func currentProcessInJob() (bool, error) {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")
	if err := proc.Find(); err != nil {
		return false, err
	}
	var result int32
	ok, _, callErr := proc.Call(uintptr(windows.CurrentProcess()), 0, uintptr(unsafe.Pointer(&result)))
	if ok == 0 {
		return false, callErr
	}
	return result != 0, nil
}
