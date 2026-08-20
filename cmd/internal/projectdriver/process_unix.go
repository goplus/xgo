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
	"syscall"
	"time"
)

const (
	driverProcessGracePeriod = 2 * time.Second
	driverProcessKillWait    = 2 * time.Second
)

func runDriverProcess(ctx context.Context, cmd *exec.Cmd) (ProcessStatus, error) {
	if err := ctx.Err(); err != nil {
		return ProcessStatus{}, err
	}
	configureDriverProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return ProcessStatus{}, err
	}
	pgid := cmd.Process.Pid
	wait := make(chan error, 1)
	go func() {
		wait <- cmd.Wait()
	}()

	select {
	case err := <-wait:
		if _, cleanupErr := stopDriverProcessGroup(pgid, syscall.SIGTERM, nil); cleanupErr != nil {
			return ProcessStatus{}, cleanupErr
		}
		return driverExitStatus(ctx, err)
	case <-ctx.Done():
	}

	initial := driverCancellationSignal(ctx)
	err, cleanupErr := stopDriverProcessGroup(pgid, initial, wait)
	if cleanupErr != nil {
		return ProcessStatus{}, cleanupErr
	}
	return driverExitStatus(ctx, err)
}

func configureDriverProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	} else {
		attr := *cmd.SysProcAttr
		cmd.SysProcAttr = &attr
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0
}

func driverCancellationSignal(ctx context.Context) syscall.Signal {
	var cause driverSignalCause
	if errors.As(context.Cause(ctx), &cause) && cause.signal != 0 {
		return cause.signal
	}
	return syscall.SIGTERM
}

// stopDriverProcessGroup allows the initial signal a bounded grace period,
// then kills the group and confirms that both its leader and descendants have
// gone. If wait is nil, the leader has already been reaped.
func stopDriverProcessGroup(pgid int, initial syscall.Signal, wait <-chan error) (error, error) {
	state := driverProcessWait{wait: wait, leaderDone: wait == nil}
	groupDone := !driverProcessGroupExists(pgid)
	if state.leaderDone && groupDone {
		return state.leaderErr, nil
	}
	var signalErr error
	if err := signalDriverProcessGroup(pgid, initial); err != nil && !errors.Is(err, syscall.ESRCH) {
		signalErr = fmt.Errorf("signal driver process group: %w", err)
	}

	groupDone = state.waitForProcessGroup(pgid, driverProcessGracePeriod)
	if state.leaderDone && groupDone {
		return state.leaderErr, signalErr
	}
	if err := signalDriverProcessGroup(pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		signalErr = errors.Join(signalErr, fmt.Errorf("kill driver process group: %w", err))
	}
	groupDone = state.waitForProcessGroup(pgid, driverProcessKillWait)
	if !state.leaderDone {
		return state.leaderErr, errors.Join(signalErr, fmt.Errorf("driver did not exit after SIGKILL"))
	}
	if !groupDone {
		return state.leaderErr, errors.Join(signalErr, fmt.Errorf("driver descendants did not exit after SIGKILL"))
	}
	return state.leaderErr, signalErr
}

type driverProcessWait struct {
	wait       <-chan error
	leaderDone bool
	leaderErr  error
}

func (s *driverProcessWait) waitForProcessGroup(pgid int, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	for {
		groupDone := !driverProcessGroupExists(pgid)
		if s.leaderDone && groupDone {
			return true
		}
		select {
		case err := <-s.wait:
			s.leaderDone = true
			s.leaderErr = err
			s.wait = nil
		case <-poll.C:
		case <-timer.C:
			return !driverProcessGroupExists(pgid)
		}
	}
}

func signalDriverProcessGroup(pgid int, sig syscall.Signal) error {
	return syscall.Kill(-pgid, sig)
}

func driverProcessGroupExists(pgid int) bool {
	err := syscall.Kill(-pgid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func exitSignal(err *exec.ExitError) (os.Signal, bool) {
	status, ok := err.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return nil, false
	}
	return status.Signal(), true
}
