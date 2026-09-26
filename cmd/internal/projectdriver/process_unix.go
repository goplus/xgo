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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	driverProcessGracePeriod  = 2 * time.Second
	driverProcessKillWait     = 2 * time.Second
	driverProcessPollInterval = 10 * time.Millisecond
)

func runDriverProcess(ctx context.Context, cmd *exec.Cmd) (status ProcessStatus, retErr error) {
	if err := ctx.Err(); err != nil {
		return ProcessStatus{}, err
	}
	terminal, err := acquireDriverTerminal(ctx, cmd.Stdin)
	if err != nil {
		return ProcessStatus{}, fmt.Errorf("inspect driver terminal: %w", err)
	}
	if terminal != nil {
		defer terminal.release()
		defer func() {
			if err := terminal.restore(); err != nil {
				status = ProcessStatus{}
				retErr = errors.Join(retErr, fmt.Errorf("restore driver terminal foreground process group: %w", err))
			}
		}()
	}
	// Recheck after waiting for the terminal lease.
	if err := ctx.Err(); err != nil {
		return ProcessStatus{}, err
	}
	configureDriverProcessGroup(cmd, terminal)
	if err := cmd.Start(); err != nil {
		return ProcessStatus{}, err
	}
	pgid := cmd.Process.Pid
	wait := make(chan error, 1)
	go func() {
		wait <- cmd.Wait()
	}()
	return waitDriverProcess(ctx, pgid, terminal, wait)
}

func waitDriverProcess(ctx context.Context, pgid int, terminal driverTerminal, wait <-chan error) (ProcessStatus, error) {
	var stopTicks <-chan time.Time
	if terminal != nil {
		stopPoll := time.NewTicker(driverProcessPollInterval)
		defer stopPoll.Stop()
		stopTicks = stopPoll.C
	}
	for {
		select {
		case err := <-wait:
			return finishDriverProcess(ctx, pgid, err)
		case <-ctx.Done():
			return cancelDriverProcess(ctx, pgid, wait)
		case <-stopTicks:
			if ctx.Err() != nil {
				return cancelDriverProcess(ctx, pgid, wait)
			}
			select {
			case err := <-wait:
				return finishDriverProcess(ctx, pgid, err)
			default:
			}
			if err := pollDriverJobControl(ctx, terminal, pgid); err != nil {
				if ctx.Err() != nil {
					return cancelDriverProcess(ctx, pgid, wait)
				}
				return abortDriverProcess(pgid, wait, err)
			}
		}
	}
}

func pollDriverJobControl(ctx context.Context, terminal driverTerminal, pgid int) error {
	handedOff, err := tryForegroundDriver(terminal, pgid)
	if err != nil || handedOff {
		return err
	}
	stop, ok, err := driverStoppedSignal(pgid)
	if err != nil {
		return fmt.Errorf("observe stopped driver: %w", err)
	}
	if !ok {
		return nil
	}
	return suspendDriverJob(ctx, terminal, pgid, stop)
}

type driverTerminal interface {
	configure(*syscall.SysProcAttr)
	restore() error
	handoffIfParentForeground(int) (bool, error)
	suspendParent(syscall.Signal) error
	release()
}

func finishDriverProcess(ctx context.Context, pgid int, waitErr error) (ProcessStatus, error) {
	if _, cleanupErr := stopDriverProcessGroup(pgid, syscall.SIGTERM, nil); cleanupErr != nil {
		return ProcessStatus{}, cleanupErr
	}
	return driverExitStatus(ctx, waitErr)
}

func cancelDriverProcess(ctx context.Context, pgid int, wait <-chan error) (ProcessStatus, error) {
	waitErr, cleanupErr := stopDriverProcessGroup(pgid, driverCancellationSignal(ctx), wait)
	if cleanupErr != nil {
		return ProcessStatus{}, cleanupErr
	}
	return driverExitStatus(ctx, waitErr)
}

func abortDriverProcess(pgid int, wait <-chan error, reason error) (ProcessStatus, error) {
	_, cleanupErr := stopDriverProcessGroup(pgid, syscall.SIGTERM, wait)
	return ProcessStatus{}, errors.Join(reason, cleanupErr)
}

func tryForegroundDriver(terminal driverTerminal, pgid int) (bool, error) {
	handedOff, err := terminal.handoffIfParentForeground(pgid)
	if err != nil || !handedOff {
		return handedOff, err
	}
	// Consume a pending SIGTTIN stop before continuing the group.
	if _, _, err := driverStoppedSignal(pgid); err != nil {
		return false, fmt.Errorf("consume pre-handoff driver stop: %w", err)
	}
	if err := signalDriverProcessGroup(pgid, syscall.SIGCONT); err != nil && !errors.Is(err, syscall.ESRCH) {
		return false, fmt.Errorf("continue foreground driver process group: %w", err)
	}
	return true, nil
}

func suspendDriverJob(ctx context.Context, terminal driverTerminal, pgid int, stop syscall.Signal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := terminal.restore(); err != nil {
		return fmt.Errorf("restore terminal before suspending driver job: %w", err)
	}
	// Cancellation may race with the restore ioctl.
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := terminal.suspendParent(stop); err != nil {
		return fmt.Errorf("suspend driver parent process group: %w", err)
	}
	// After SIGCONT, hand off only if the shell foregrounded this job.
	if err := ctx.Err(); err != nil {
		return err
	}
	handedOff, err := tryForegroundDriver(terminal, pgid)
	if err != nil {
		return fmt.Errorf("resume driver terminal foreground process group: %w", err)
	}
	if !handedOff {
		if err := signalDriverProcessGroup(pgid, syscall.SIGCONT); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("continue background driver process group: %w", err)
		}
	}
	return nil
}

func configureDriverProcessGroup(cmd *exec.Cmd, terminal driverTerminal) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	} else {
		attr := *cmd.SysProcAttr
		cmd.SysProcAttr = &attr
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0
	if terminal != nil {
		terminal.configure(cmd.SysProcAttr)
	}
}

func driverCancellationSignal(ctx context.Context) syscall.Signal {
	var cause driverSignalCause
	if errors.As(context.Cause(ctx), &cause) && cause.signal != 0 {
		return cause.signal
	}
	return syscall.SIGTERM
}

// stopDriverProcessGroup terminates the driver group, escalating after the grace period.
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
	poll := time.NewTicker(driverProcessPollInterval)
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
