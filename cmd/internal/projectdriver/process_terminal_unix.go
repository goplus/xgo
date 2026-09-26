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
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// Serialize terminal handoffs without blocking context cancellation.
var driverTerminalLease = func() chan struct{} {
	lease := make(chan struct{}, 1)
	lease <- struct{}{}
	return lease
}()

type unixDriverTerminal struct {
	fd                      int
	parentPgrp              int
	initialParentForeground bool
	driverOwnsForeground    bool
}

const childStoppedCode = 5 // CLD_STOPPED

// Observe stops without competing with cmd.Wait for exit status.
func driverStoppedSignal(pid int) (syscall.Signal, bool, error) {
	for {
		stop, ok, err := waitidStopped(pid)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if errors.Is(err, syscall.ECHILD) || errors.Is(err, syscall.ESRCH) {
			return 0, false, nil
		}
		if err != nil {
			return 0, false, err
		}
		return stop, ok, nil
	}
}

func acquireDriverTerminal(ctx context.Context, stdin any) (driverTerminal, error) {
	file, ok := stdin.(*os.File)
	if !ok {
		return nil, nil
	}
	fd, err := duplicateDriverTerminalFD(file)
	if err != nil {
		return nil, err
	}

	_, terminal, err := terminalForegroundProcessGroup(fd)
	if err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	if !terminal {
		_ = unix.Close(fd)
		return nil, nil
	}

	if err := acquireDriverTerminalLease(ctx); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	foreground, terminal, err := terminalForegroundProcessGroup(fd)
	if err != nil || !terminal {
		releaseDriverTerminalLease()
		_ = unix.Close(fd)
		return nil, err
	}
	parentPgrp := syscall.Getpgrp()
	return &unixDriverTerminal{
		fd: fd, parentPgrp: parentPgrp, initialParentForeground: foreground == parentPgrp,
	}, nil
}

func duplicateDriverTerminalFD(file *os.File) (int, error) {
	raw, err := file.SyscallConn()
	if err != nil {
		return -1, err
	}
	fd := -1
	var duplicateErr error
	if err := raw.Control(func(source uintptr) {
		fd, duplicateErr = unix.FcntlInt(source, unix.F_DUPFD_CLOEXEC, 0)
	}); err != nil {
		return -1, err
	}
	return fd, duplicateErr
}

func acquireDriverTerminalLease(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-driverTerminalLease:
		// Both select cases may become ready together.
		if err := ctx.Err(); err != nil {
			releaseDriverTerminalLease()
			return err
		}
		return nil
	}
}

func releaseDriverTerminalLease() {
	driverTerminalLease <- struct{}{}
}

func terminalForegroundProcessGroup(fd int) (pgid int, terminal bool, err error) {
	terminal, err = isDriverTerminal(fd)
	if err != nil || !terminal {
		return 0, terminal, err
	}
	pgid, err = unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if errors.Is(err, syscall.ENOTTY) || errors.Is(err, syscall.ENODEV) {
		return 0, false, nil
	}
	return pgid, err == nil, err
}

func (t *unixDriverTerminal) configure(attr *syscall.SysProcAttr) {
	if !t.initialParentForeground {
		// A background invocation must not steal the terminal.
		return
	}
	// Foreground makes setpgid and TIOCSPGRP atomic with child startup.
	attr.Foreground = true
	attr.Ctty = t.fd
	t.driverOwnsForeground = true
}

func (t *unixDriverTerminal) restore() error {
	if !t.driverOwnsForeground {
		return nil
	}
	if err := setTerminalForegroundProcessGroup(t.fd, t.parentPgrp); err != nil {
		return err
	}
	t.driverOwnsForeground = false
	return nil
}

func (t *unixDriverTerminal) handoffIfParentForeground(pgid int) (bool, error) {
	if t.driverOwnsForeground {
		return false, nil
	}
	foreground, terminal, err := terminalForegroundProcessGroup(t.fd)
	if err != nil {
		return false, err
	}
	if !terminal || foreground != t.parentPgrp {
		// The shell kept this job in the background.
		return false, nil
	}
	if err := setTerminalForegroundProcessGroup(t.fd, pgid); err != nil {
		return false, err
	}
	t.driverOwnsForeground = true
	return true, nil
}

func (t *unixDriverTerminal) suspendParent(stop syscall.Signal) error {
	return syscall.Kill(-t.parentPgrp, stop)
}

func (t *unixDriverTerminal) release() {
	_ = unix.Close(t.fd)
	releaseDriverTerminalLease()
}
