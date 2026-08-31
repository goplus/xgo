//go:build darwin || linux

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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const driverTTYHelperEnv = "XGO_DRIVER_TTY_HELPER"

type ttyScenario struct {
	t           *testing.T
	cancel      context.CancelFunc
	input       *os.File
	wait        chan error
	output      bytes.Buffer
	cleanupPIDs []string
	done        bool
}

func startTTYScenario(t *testing.T, timeout time.Duration, mode string, cleanupPIDs []string, paths ...string) *ttyScenario {
	t.Helper()
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script(1) is required for PTY job-control coverage")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	cmd := driverTTYScriptCommand(ctx, script, mode, paths...)
	cmd.Env = append(os.Environ(), driverTTYHelperEnv+"=1")
	input, writer, err := os.Pipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	s := &ttyScenario{t: t, cancel: cancel, input: writer, wait: make(chan error, 1), cleanupPIDs: cleanupPIDs}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = input, &s.output, &s.output
	if err := cmd.Start(); err != nil {
		input.Close()
		writer.Close()
		cancel()
		t.Fatal(err)
	}
	input.Close()
	go func() { s.wait <- cmd.Wait() }()
	t.Cleanup(func() {
		if s.done {
			return
		}
		s.cancel()
		_ = s.input.Close()
		for _, path := range s.cleanupPIDs {
			killRecordedTTYHelper(path)
		}
		select {
		case <-s.wait:
		case <-time.After(5 * time.Second):
		}
	})
	return s
}

func (s *ttyScenario) waitFile(path, description string) []byte {
	s.t.Helper()
	data, err := waitForHelperFile(path, 5*time.Second)
	if err != nil {
		s.t.Fatalf("%s: %v\n%s", description, err, s.output.String())
	}
	return data
}

func (s *ttyScenario) write(data []byte, description string) {
	s.t.Helper()
	if _, err := s.input.Write(data); err != nil {
		s.t.Fatalf("%s: %v\n%s", description, err, s.output.String())
	}
}

func (s *ttyScenario) finish(description string) {
	s.t.Helper()
	_ = s.input.Close()
	select {
	case err := <-s.wait:
		if err != nil {
			s.t.Fatalf("%s: %v\n%s", description, err, s.output.String())
		}
		s.done = true
		s.cancel()
	case <-time.After(5 * time.Second):
		s.t.Fatalf("%s did not exit\n%s", description, s.output.String())
	}
}

// A real PTY catches regressions where SIGTTIN stops the driver.
func TestRunDriverProcessInteractiveTTY(t *testing.T) {
	dir := t.TempDir()
	parentPID := filepath.Join(dir, "parent-pid")
	childPID := filepath.Join(dir, "child-pid")
	ready := filepath.Join(dir, "ready")
	received := filepath.Join(dir, "received")
	s := startTTYScenario(t, 15*time.Second, "parent", []string{parentPID, childPID}, parentPID, childPID, ready, received)
	s.waitFile(ready, "driver did not become foreground-ready")
	const want = "interactive driver input\n"
	s.write([]byte(want), "write PTY input")
	data := s.waitFile(received, "driver did not read PTY input")
	s.finish("PTY helper")
	if string(data) != want {
		t.Fatalf("driver read %q, want %q", data, want)
	}
}

func TestRunDriverProcessSuspendResumeTTY(t *testing.T) {
	dir := t.TempDir()
	shellPID := filepath.Join(dir, "shell-pid")
	parentPID := filepath.Join(dir, "parent-pid")
	childPID := filepath.Join(dir, "child-pid")
	ready := filepath.Join(dir, "ready")
	parentStopped := filepath.Join(dir, "parent-stopped")
	resume := filepath.Join(dir, "resume")
	continued := filepath.Join(dir, "continued")
	received := filepath.Join(dir, "received")
	s := startTTYScenario(t, 20*time.Second, "shell", []string{shellPID, parentPID, childPID},
		shellPID, parentStopped, resume, parentPID, childPID, ready, continued, received)
	s.waitFile(ready, "driver did not become foreground-ready")
	s.write([]byte{0x1a}, "write Ctrl-Z to PTY")
	s.waitFile(parentStopped, "parent job did not stop")
	writeTTYHelperFile(t, resume, "fg")
	s.waitFile(continued, "driver did not receive SIGCONT")
	const want = "input after fg\n"
	s.write([]byte(want), "write resumed PTY input")
	data := s.waitFile(received, "resumed driver did not read PTY input")
	s.finish("suspend/resume PTY helper")
	if string(data) != want {
		t.Fatalf("resumed driver read %q, want %q", data, want)
	}
}

func TestRunDriverProcessInitialBackgroundThenForegroundTTY(t *testing.T) {
	dir := t.TempDir()
	shellPID := filepath.Join(dir, "shell-pid")
	parentPID := filepath.Join(dir, "parent-pid")
	childPID := filepath.Join(dir, "child-pid")
	ready := filepath.Join(dir, "ready")
	parentStopped := filepath.Join(dir, "parent-stopped")
	foreground := filepath.Join(dir, "foreground")
	continued := filepath.Join(dir, "continued")
	received := filepath.Join(dir, "received")
	s := startTTYScenario(t, 20*time.Second, "initial-background-shell", []string{shellPID, parentPID, childPID},
		shellPID, parentStopped, foreground, parentPID, childPID, ready, continued, received)
	s.waitFile(ready, "background driver did not start")
	s.waitFile(parentStopped, "background parent did not propagate SIGTTIN stop")
	writeTTYHelperFile(t, foreground, "fg")
	s.waitFile(continued, "foregrounded background driver did not continue")
	const want = "one fg is enough\n"
	s.write([]byte(want), "write foregrounded driver input")
	data := s.waitFile(received, "foregrounded driver did not read")
	s.finish("initial-background PTY helper")
	if string(data) != want {
		t.Fatalf("foregrounded driver read %q, want %q", data, want)
	}
}

func TestRunDriverProcessBackgroundThenLaterForegroundTTY(t *testing.T) {
	dir := t.TempDir()
	shellPID := filepath.Join(dir, "shell-pid")
	parentPID := filepath.Join(dir, "parent-pid")
	childPID := filepath.Join(dir, "child-pid")
	ready := filepath.Join(dir, "ready")
	parentStopped := filepath.Join(dir, "parent-stopped")
	background := filepath.Join(dir, "background")
	continued := filepath.Join(dir, "continued")
	foreground := filepath.Join(dir, "foreground")
	allowRead := filepath.Join(dir, "allow-read")
	driverForeground := filepath.Join(dir, "driver-foreground")
	received := filepath.Join(dir, "received")
	s := startTTYScenario(t, 20*time.Second, "background-foreground-shell", []string{shellPID, parentPID, childPID},
		shellPID, parentStopped, background, foreground, parentPID, childPID, ready, continued, allowRead, driverForeground, received)
	s.waitFile(ready, "driver did not become foreground-ready")
	s.write([]byte{0x1a}, "write Ctrl-Z to PTY")
	s.waitFile(parentStopped, "parent job did not stop")
	writeTTYHelperFile(t, background, "bg")
	s.waitFile(continued, "backgrounded driver did not continue")
	// A running job can be foregrounded by changing TPGID alone.
	writeTTYHelperFile(t, foreground, "fg")
	writeTTYHelperFile(t, allowRead, "read")
	s.waitFile(driverForeground, "running background driver did not receive TTY handoff")
	const want = "input after later fg\n"
	s.write([]byte(want), "write later-foreground input")
	data := s.waitFile(received, "later-foreground driver did not read")
	s.finish("background/foreground PTY helper")
	if string(data) != want {
		t.Fatalf("later-foreground driver read %q, want %q", data, want)
	}
}

func TestDriverTerminalLeaseCancellation(t *testing.T) {
	if err := acquireDriverTerminalLease(context.Background()); err != nil {
		t.Fatal(err)
	}
	held := true
	defer func() {
		if held {
			releaseDriverTerminalLease()
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- acquireDriverTerminalLease(ctx) }()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lease wait = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled terminal lease wait did not return")
	}
	releaseDriverTerminalLease()
	held = false
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := acquireDriverTerminalLease(ctx); err != nil {
		t.Fatalf("reacquire terminal lease after cancellation: %v", err)
	}
	releaseDriverTerminalLease()
}

func TestAcquireDriverTerminalPreservesFilePoller(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	terminal, err := acquireDriverTerminal(context.Background(), reader)
	if err != nil || terminal != nil {
		t.Fatalf("acquireDriverTerminal(pipe) = (%v, %v)", terminal, err)
	}
	if err := reader.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("stdin deadline stopped working: %v", err)
	}
}

type cancelingTerminal struct {
	cancel          context.CancelFunc
	cancelOnRestore bool
	restores        int
	suspends        int
}

func (*cancelingTerminal) configure(*syscall.SysProcAttr) {}
func (*cancelingTerminal) release()                       {}
func (*cancelingTerminal) handoffIfParentForeground(int) (bool, error) {
	return false, nil
}
func (t *cancelingTerminal) restore() error {
	t.restores++
	if t.cancelOnRestore {
		t.cancel()
	}
	return nil
}
func (t *cancelingTerminal) suspendParent(syscall.Signal) error {
	t.suspends++
	return nil
}

func TestSuspendDriverJobHonorsCancellation(t *testing.T) {
	for _, test := range []struct {
		name            string
		cancelBefore    bool
		cancelOnRestore bool
		wantRestores    int
	}{
		{name: "before restore", cancelBefore: true},
		{name: "after restore", cancelOnRestore: true, wantRestores: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			terminal := &cancelingTerminal{cancel: cancel, cancelOnRestore: test.cancelOnRestore}
			if test.cancelBefore {
				cancel()
			}
			err := suspendDriverJob(ctx, terminal, 12345, syscall.SIGTSTP)
			if !errors.Is(err, context.Canceled) || terminal.restores != test.wantRestores || terminal.suspends != 0 {
				t.Fatalf("suspend = %v, restores = %d, suspends = %d", err, terminal.restores, terminal.suspends)
			}
		})
	}
}

func TestDriverTTYHelper(t *testing.T) {
	if os.Getenv(driverTTYHelperEnv) != "1" {
		return
	}
	args := driverTTYHelperArgs()
	if len(args) == 0 {
		t.Fatal("missing TTY helper mode")
	}
	switch args[0] {
	case "initial-background-shell":
		if len(args) != 9 {
			t.Fatalf("initial-background-shell args = %q", args)
		}
		writeTTYHelperPID(t, args[1])
		process := startTTYHelper(t, false, "background-parent", args[4], args[5], args[6], args[7], args[8])
		defer process.Release()
		status := waitTTYHelper(t, process.Pid, syscall.WUNTRACED)
		if !status.Stopped() || status.StopSignal() != syscall.SIGTTIN {
			t.Fatalf("background parent wait status = %#x, want SIGTTIN stop", int(status))
		}
		assertTTYHelperForeground(t, "background test shell")
		writeTTYHelperFile(t, args[2], "stopped")
		waitTTYHelperFile(t, args[3])
		if err := setTerminalForegroundProcessGroup(int(os.Stdin.Fd()), process.Pid); err != nil {
			t.Fatalf("foreground background parent: %v", err)
		}
		if err := syscall.Kill(-process.Pid, syscall.SIGCONT); err != nil {
			t.Fatalf("continue background parent: %v", err)
		}
		waitTTYHelperExit(t, process, "background parent")
	case "background-foreground-shell":
		if len(args) != 12 {
			t.Fatalf("background-foreground-shell args = %q", args)
		}
		writeTTYHelperPID(t, args[1])
		process := startTTYHelper(t, true, "delayed-parent", args[5], args[6], args[7], args[8], args[9], args[10], args[11])
		defer process.Release()
		status := waitTTYHelper(t, process.Pid, syscall.WUNTRACED)
		if !status.Stopped() || status.StopSignal() != syscall.SIGTSTP {
			t.Fatalf("foreground parent wait status = %#x, want SIGTSTP stop", int(status))
		}
		if err := setTerminalForegroundProcessGroup(int(os.Stdin.Fd()), syscall.Getpgrp()); err != nil {
			t.Fatalf("restore shell before bg: %v", err)
		}
		writeTTYHelperFile(t, args[2], "stopped")
		waitTTYHelperFile(t, args[3])
		if err := syscall.Kill(-process.Pid, syscall.SIGCONT); err != nil {
			t.Fatalf("background parent: %v", err)
		}
		waitTTYHelperFile(t, args[4])
		// A running job may be foregrounded without SIGCONT.
		if err := setTerminalForegroundProcessGroup(int(os.Stdin.Fd()), process.Pid); err != nil {
			t.Fatalf("foreground running parent: %v", err)
		}
		waitTTYHelperExit(t, process, "later-foreground parent")
	case "shell":
		if len(args) != 9 {
			t.Fatalf("shell args = %q", args)
		}
		writeTTYHelperPID(t, args[1])
		process := startTTYHelper(t, true, "suspend-parent", args[4], args[5], args[6], args[7], args[8])
		defer process.Release()
		status := waitTTYHelper(t, process.Pid, syscall.WUNTRACED)
		if !status.Stopped() || status.StopSignal() != syscall.SIGTSTP {
			t.Fatalf("parent wait status = %#x, want SIGTSTP stop", int(status))
		}
		foreground, err := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGPGRP)
		if err != nil {
			t.Fatal(err)
		}
		if foreground != process.Pid {
			t.Fatalf("terminal foreground = %d, want stopped parent %d", foreground, process.Pid)
		}
		if err := setTerminalForegroundProcessGroup(int(os.Stdin.Fd()), syscall.Getpgrp()); err != nil {
			t.Fatalf("restore test shell foreground: %v", err)
		}
		writeTTYHelperFile(t, args[2], "stopped")
		waitTTYHelperFile(t, args[3])
		if err := setTerminalForegroundProcessGroup(int(os.Stdin.Fd()), process.Pid); err != nil {
			t.Fatalf("foreground stopped parent: %v", err)
		}
		if err := syscall.Kill(-process.Pid, syscall.SIGCONT); err != nil {
			t.Fatalf("continue parent: %v", err)
		}
		waitTTYHelperExit(t, process, "parent")
	case "parent":
		runTTYParent(t, args, 5, "reader", "parent")
	case "suspend-parent":
		runTTYParent(t, args, 6, "suspend-reader", "resumed parent")
	case "background-parent":
		runTTYParent(t, args, 6, "background-reader", "foregrounded background parent")
	case "delayed-parent":
		runTTYParent(t, args, 8, "delayed-reader", "later-foreground parent")
	case "reader":
		if len(args) != 4 {
			t.Fatalf("reader args = %q", args)
		}
		writeTTYHelperPID(t, args[1])
		assertTTYHelperForeground(t, "driver")
		writeTTYHelperFile(t, args[2], "ready")
		line := readTTYInput(t)
		writeTTYHelperFile(t, args[3], line)
	case "suspend-reader", "background-reader":
		if len(args) != 5 {
			t.Fatalf("%s args = %q", args[0], args)
		}
		writeTTYHelperPID(t, args[1])
		if args[0] == "suspend-reader" {
			assertTTYHelperForeground(t, args[0])
		}
		continued := make(chan os.Signal, 1)
		signal.Notify(continued, syscall.SIGCONT)
		defer signal.Stop(continued)
		go func() {
			<-continued
			_ = os.WriteFile(args[3], []byte("continued"), 0o600)
		}()
		writeTTYHelperFile(t, args[2], "ready")
		line := readTTYInput(t)
		assertTTYHelperForeground(t, args[0])
		writeTTYHelperFile(t, args[4], line)
	case "delayed-reader":
		if len(args) != 7 {
			t.Fatalf("delayed-reader args = %q", args)
		}
		writeTTYHelperPID(t, args[1])
		assertTTYHelperForeground(t, "delayed driver")
		continued := make(chan os.Signal, 1)
		signal.Notify(continued, syscall.SIGCONT)
		defer signal.Stop(continued)
		go func() {
			<-continued
			_ = os.WriteFile(args[3], []byte("continued"), 0o600)
		}()
		writeTTYHelperFile(t, args[2], "ready")
		waitTTYHelperFile(t, args[4])
		if err := waitForTTYForeground(syscall.Getpgrp(), 5*time.Second); err != nil {
			t.Fatal(err)
		}
		writeTTYHelperFile(t, args[5], "foreground")
		line := readTTYInput(t)
		writeTTYHelperFile(t, args[6], line)
	default:
		t.Fatalf("unknown TTY helper mode %q", args[0])
	}
}

func runTTYParent(t *testing.T, args []string, wantArgs int, childMode, label string) {
	t.Helper()
	if len(args) != wantArgs {
		t.Fatalf("%s args = %q", args[0], args)
	}
	writeTTYHelperPID(t, args[1])
	cmd := driverTTYHelperCommand(append([]string{childMode}, args[2:]...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	status, err := runDriverProcess(context.Background(), cmd)
	if err != nil || !status.Success() {
		t.Fatalf("runDriverProcess() = (%+v, %v)", status, err)
	}
	assertTTYHelperForeground(t, label)
}

func readTTYInput(t *testing.T) string {
	t.Helper()
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func assertTTYHelperForeground(t *testing.T, name string) {
	t.Helper()
	foreground, err := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGPGRP)
	if err != nil {
		t.Fatalf("read terminal foreground process group: %v", err)
	}
	if foreground != syscall.Getpgrp() {
		t.Fatalf("%s process group = %d, terminal foreground = %d", name, syscall.Getpgrp(), foreground)
	}
}

func driverTTYScriptCommand(ctx context.Context, script, mode string, paths ...string) *exec.Cmd {
	args := []string{os.Args[0], "-test.run=^TestDriverTTYHelper$", "--", mode}
	args = append(args, paths...)
	if runtime.GOOS == "darwin" {
		return exec.CommandContext(ctx, script, append([]string{"-q", "/dev/null"}, args...)...)
	}
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
	}
	return exec.CommandContext(ctx, script, "-q", "-e", "-c", strings.Join(quoted, " "), "/dev/null")
}

func startTTYHelper(t *testing.T, foreground bool, args ...string) *os.Process {
	t.Helper()
	commandArgs := []string{os.Args[0], "-test.run=^TestDriverTTYHelper$", "--"}
	commandArgs = append(commandArgs, args...)
	attr := &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	if foreground {
		attr.Foreground = true
		attr.Ctty = int(os.Stdin.Fd())
	}
	process, err := os.StartProcess(os.Args[0], commandArgs, &os.ProcAttr{
		Env:   append(os.Environ(), driverTTYHelperEnv+"=1"),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys:   attr,
	})
	if err != nil {
		t.Fatal(err)
	}
	return process
}

func waitTTYHelper(t *testing.T, pid, options int) syscall.WaitStatus {
	t.Helper()
	for {
		var status syscall.WaitStatus
		got, err := syscall.Wait4(pid, &status, options, nil)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if got != pid {
			t.Fatalf("wait4() = %d, want %d", got, pid)
		}
		return status
	}
}

func waitTTYHelperExit(t *testing.T, process *os.Process, label string) {
	t.Helper()
	status := waitTTYHelper(t, process.Pid, 0)
	if err := setTerminalForegroundProcessGroup(int(os.Stdin.Fd()), syscall.Getpgrp()); err != nil {
		t.Fatalf("restore test shell: %v", err)
	}
	if !status.Exited() || status.ExitStatus() != 0 {
		t.Fatalf("%s wait status = %#x", label, int(status))
	}
}

func waitForTTYForeground(pgid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		foreground, err := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGPGRP)
		if err != nil {
			return err
		}
		if foreground == pgid {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("terminal foreground = %d, want %d", foreground, pgid)
		}
		time.Sleep(driverProcessPollInterval)
	}
}

func driverTTYHelperCommand(args ...string) *exec.Cmd {
	commandArgs := []string{"-test.run=^TestDriverTTYHelper$", "--"}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command(os.Args[0], commandArgs...)
	cmd.Env = append(os.Environ(), driverTTYHelperEnv+"=1")
	return cmd
}

func driverTTYHelperArgs() []string {
	for i, arg := range os.Args {
		if arg == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

func writeTTYHelperPID(t *testing.T, path string) {
	t.Helper()
	writeTTYHelperFile(t, path, strconv.Itoa(os.Getpid()))
}

func writeTTYHelperFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func waitTTYHelperFile(t *testing.T, path string) {
	t.Helper()
	if _, err := waitForHelperFile(path, 5*time.Second); err != nil {
		t.Fatal(err)
	}
}

func killRecordedTTYHelper(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil || pid <= 0 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
