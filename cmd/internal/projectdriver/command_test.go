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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRunCapturedCommand(t *testing.T) {
	if os.Getenv("XGO_TEST_CAPTURED_COMMAND") == "1" {
		fmt.Fprint(os.Stdout, "captured stdout")
		fmt.Fprintln(os.Stderr, "captured stderr")
		os.Exit(17)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunCapturedCommand$")
	cmd.Env = append(os.Environ(), "XGO_TEST_CAPTURED_COMMAND=1")
	stdout, stderr, err := runCapturedCommand(cmd, "test command")
	if string(stdout) != "captured stdout" || string(stderr) != "captured stderr\n" {
		t.Fatalf("captured output = %q, %q", stdout, stderr)
	}
	if err == nil || !strings.Contains(err.Error(), "test command: exit status 17: captured stderr") {
		t.Fatalf("captured error = %v", err)
	}
}

func TestCommandErrorWithoutStderr(t *testing.T) {
	cause := errors.New("failed")
	err := commandError("test command", cause, " \n\t")
	if got, want := err.Error(), "test command: failed"; got != want {
		t.Fatalf("command error = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("command error does not wrap cause: %v", err)
	}
}
