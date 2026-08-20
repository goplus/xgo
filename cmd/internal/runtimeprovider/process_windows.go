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

package runtimeprovider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ntResumeProcess = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess")

func runProviderProcess(ctx context.Context, cmd *exec.Cmd) (ProcessStatus, error) {
	if err := ctx.Err(); err != nil {
		return ProcessStatus{}, err
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return ProcessStatus{}, err
	}
	defer windows.CloseHandle(job)
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		return ProcessStatus{}, err
	}
	configureSuspendedProvider(cmd)
	if err := cmd.Start(); err != nil {
		return ProcessStatus{}, err
	}
	// os/exec closes the primary thread handle before Start returns. Opening the
	// still-suspended process and resuming it with NtResumeProcess lets us retain
	// os/exec's exact argv/environment/stdio behavior without an execution window
	// before Job assignment.
	process, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SUSPEND_RESUME,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		abortSuspendedProvider(cmd, 0)
		return ProcessStatus{}, err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		abortSuspendedProvider(cmd, process)
		return ProcessStatus{}, err
	}
	if err := ctx.Err(); err != nil {
		terminateProviderJob(job, process)
		_ = cmd.Wait()
		return ProcessStatus{}, err
	}
	if err := resumeProviderProcess(process); err != nil {
		terminateProviderJob(job, process)
		_ = cmd.Wait()
		return ProcessStatus{}, err
	}
	done := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-ctx.Done():
			terminateProviderJob(job, process)
		case <-done:
		}
	}()
	err = cmd.Wait()
	close(done)
	<-watchDone
	status, err := processStatus(err)
	if err != nil {
		return ProcessStatus{}, err
	}
	// The process and cancellation watcher can become ready at the same time.
	// Check the context even when the watcher selected the normal-exit branch.
	return statusUnlessCanceled(ctx, status)
}

func configureSuspendedProvider(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	} else {
		attr := *cmd.SysProcAttr
		cmd.SysProcAttr = &attr
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
}

func resumeProviderProcess(process windows.Handle) error {
	if err := ntResumeProcess.Find(); err != nil {
		return fmt.Errorf("resolve NtResumeProcess: %w", err)
	}
	result, _, _ := ntResumeProcess.Call(uintptr(process))
	status := windows.NTStatus(uint32(result))
	if status != windows.STATUS_SUCCESS {
		return fmt.Errorf("resume suspended runtime provider: %w", status)
	}
	return nil
}

func abortSuspendedProvider(cmd *exec.Cmd, process windows.Handle) {
	if process != 0 {
		_ = windows.TerminateProcess(process, 1)
	} else {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}

func terminateProviderJob(job, process windows.Handle) {
	if err := windows.TerminateJobObject(job, 1); err != nil && process != 0 {
		_ = windows.TerminateProcess(process, 1)
	}
}

func exitSignal(*exec.ExitError) (os.Signal, bool) { return nil, false }
