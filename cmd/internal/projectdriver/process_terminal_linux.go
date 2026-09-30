//go:build linux

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
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func isDriverTerminal(fd int) (bool, error) {
	_, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if errors.Is(err, syscall.ENOTTY) || errors.Is(err, syscall.ENODEV) {
		return false, nil
	}
	return err == nil, err
}

func setTerminalForegroundProcessGroup(fd, pgid int) error {
	// Block SIGTTOU on this thread while changing the foreground group.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var blocked, old unix.Sigset_t
	blocked.Val[0] = 1 << (uint(syscall.SIGTTOU) - 1)
	if err := unix.PthreadSigmask(unix.SIG_BLOCK, &blocked, &old); err != nil {
		return err
	}
	setErr := unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, pgid)
	restoreErr := unix.PthreadSigmask(unix.SIG_SETMASK, &old, nil)
	return errors.Join(setErr, restoreErr)
}

func waitidStopped(pid int) (syscall.Signal, bool, error) {
	var info unix.Siginfo
	err := unix.Waitid(unix.P_PID, pid, &info, unix.WSTOPPED|unix.WNOHANG, nil)
	// The SIGCHLD union follows three ints and is pointer-aligned; si_status
	// follows its pid and uid fields.
	offset := uintptr(12)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		offset = 16
	}
	status := *(*int32)(unsafe.Add(unsafe.Pointer(&info), offset+8))
	return syscall.Signal(status), info.Signo != 0 && info.Code == childStoppedCode, err
}
