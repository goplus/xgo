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
	"syscall"
	"time"
)

const (
	providerProcessGracePeriod = 2 * time.Second
	providerProcessKillWait    = 2 * time.Second
)

func runProviderProcess(ctx context.Context, cmd *exec.Cmd) (ProcessStatus, error) {
	if err := ctx.Err(); err != nil {
		return ProcessStatus{}, err
	}
	configureProviderProcessGroup(cmd)
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
		if _, cleanupErr := stopProviderProcessGroup(pgid, syscall.SIGTERM, nil); cleanupErr != nil {
			return ProcessStatus{}, cleanupErr
		}
		return processStatus(err)
	case <-ctx.Done():
	}

	initial := providerCancellationSignal(ctx)
	err, cleanupErr := stopProviderProcessGroup(pgid, initial, wait)
	if cleanupErr != nil {
		return ProcessStatus{}, cleanupErr
	}
	return processStatus(err)
}

func configureProviderProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	} else {
		attr := *cmd.SysProcAttr
		cmd.SysProcAttr = &attr
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0
}

func providerCancellationSignal(ctx context.Context) syscall.Signal {
	var cause runtimeSignalCause
	if errors.As(context.Cause(ctx), &cause) && cause.signal != 0 {
		return cause.signal
	}
	return syscall.SIGTERM
}

// stopProviderProcessGroup allows the initial signal a bounded grace period,
// then kills the group and confirms that both its leader and descendants have
// gone. If wait is nil, the leader has already been reaped.
func stopProviderProcessGroup(pgid int, initial syscall.Signal, wait <-chan error) (error, error) {
	state := providerProcessWait{wait: wait, leaderDone: wait == nil}
	groupDone := !providerProcessGroupExists(pgid)
	if state.leaderDone && groupDone {
		return state.leaderErr, nil
	}
	var signalErr error
	if err := signalProviderProcessGroup(pgid, initial); err != nil && !errors.Is(err, syscall.ESRCH) {
		signalErr = fmt.Errorf("signal runtime provider process group: %w", err)
	}

	groupDone = state.waitForProcessGroup(pgid, providerProcessGracePeriod)
	if state.leaderDone && groupDone {
		return state.leaderErr, signalErr
	}
	if err := signalProviderProcessGroup(pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		signalErr = errors.Join(signalErr, fmt.Errorf("kill runtime provider process group: %w", err))
	}
	groupDone = state.waitForProcessGroup(pgid, providerProcessKillWait)
	if !state.leaderDone {
		return state.leaderErr, errors.Join(signalErr, fmt.Errorf("runtime provider did not exit after SIGKILL"))
	}
	if !groupDone {
		return state.leaderErr, errors.Join(signalErr, fmt.Errorf("runtime provider descendants did not exit after SIGKILL"))
	}
	return state.leaderErr, signalErr
}

type providerProcessWait struct {
	wait       <-chan error
	leaderDone bool
	leaderErr  error
}

func (s *providerProcessWait) waitForProcessGroup(pgid int, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	for {
		groupDone := !providerProcessGroupExists(pgid)
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
			return !providerProcessGroupExists(pgid)
		}
	}
}

func signalProviderProcessGroup(pgid int, sig syscall.Signal) error {
	return syscall.Kill(-pgid, sig)
}

func providerProcessGroupExists(pgid int) bool {
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
