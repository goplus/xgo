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
	"strings"
	"testing"
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
