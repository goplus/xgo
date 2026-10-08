//go:build darwin

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

const (
	darwinSIGBlock     = 1
	darwinSIGSetMask   = 3
	darwinPID          = 1   // P_PID
	darwinSiginfoBytes = 104 // sizeof(siginfo_t)
)

type darwinWaitidSiginfo struct {
	Signo  int32
	Errno  int32
	Code   int32
	PID    int32
	UID    uint32
	Status int32
	_      [darwinSiginfoBytes - 24]byte
}

func isDriverTerminal(fd int) (bool, error) {
	_, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if errors.Is(err, syscall.ENOTTY) || errors.Is(err, syscall.ENODEV) {
		return false, nil
	}
	return err == nil, err
}

func setTerminalForegroundProcessGroup(fd, pgid int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	blocked := uint32(1 << (uint(syscall.SIGTTOU) - 1))
	var old uint32
	if err := darwinPthreadSigmask(darwinSIGBlock, &blocked, &old); err != nil {
		return err
	}
	setErr := unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, pgid)
	restoreErr := darwinPthreadSigmask(darwinSIGSetMask, &old, nil)
	return errors.Join(setErr, restoreErr)
}

func darwinPthreadSigmask(how int, set, old *uint32) error {
	_, _, errno := syscall.RawSyscall(
		syscall.SYS___PTHREAD_SIGMASK,
		uintptr(how),
		uintptr(unsafe.Pointer(set)),
		uintptr(unsafe.Pointer(old)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func waitidStopped(pid int) (syscall.Signal, bool, error) {
	var info darwinWaitidSiginfo
	_, _, errno := syscall.RawSyscall6(
		syscall.SYS_WAITID,
		darwinPID,
		uintptr(pid),
		uintptr(unsafe.Pointer(&info)),
		unix.WSTOPPED|unix.WNOHANG,
		0,
		0,
	)
	if errno != 0 {
		return 0, false, errno
	}
	return syscall.Signal(info.Status), info.Signo != 0 && info.Code == childStoppedCode, nil
}
