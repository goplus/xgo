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
	"context"
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ntResumeProcess = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtResumeProcess")

const driverJobExitWait = 2 * time.Second

func runDriverProcess(ctx context.Context, cmd *exec.Cmd) (ProcessStatus, error) {
	if err := ctx.Err(); err != nil {
		return ProcessStatus{}, err
	}
	job, err := createDriverJob()
	if err != nil {
		return ProcessStatus{}, err
	}
	defer windows.CloseHandle(job)
	process, err := startDriverInJob(ctx, cmd, job)
	if err != nil {
		return ProcessStatus{}, err
	}
	return waitForDriverProcess(ctx, cmd, job, process)
}

func createDriverJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func startDriverInJob(ctx context.Context, cmd *exec.Cmd, job windows.Handle) (windows.Handle, error) {
	configureSuspendedDriver(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	// os/exec closes the primary thread handle, so resume through the process.
	process, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SUSPEND_RESUME,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		abortSuspendedDriver(cmd, 0)
		return 0, err
	}
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		abortSuspendedDriver(cmd, process)
		_ = windows.CloseHandle(process)
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		terminateDriverJob(job, process)
		_ = cmd.Wait()
		_ = windows.CloseHandle(process)
		return 0, err
	}
	if err := resumeDriverProcess(process); err != nil {
		terminateDriverJob(job, process)
		_ = cmd.Wait()
		_ = windows.CloseHandle(process)
		return 0, err
	}
	return process, nil
}

func waitForDriverProcess(ctx context.Context, cmd *exec.Cmd, job, process windows.Handle) (ProcessStatus, error) {
	done := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-ctx.Done():
			terminateDriverJob(job, process)
		case <-done:
		}
	}()
	err := cmd.Wait()
	close(done)
	<-watchDone
	terminateDriverJob(job, process)
	_ = windows.CloseHandle(process)
	if cleanupErr := waitForDriverJobExit(job); cleanupErr != nil {
		return ProcessStatus{}, cleanupErr
	}
	// The process and cancellation watcher can become ready at the same time.
	// Check the context even when the watcher selected the normal-exit branch.
	return driverExitStatus(ctx, err)
}

func configureSuspendedDriver(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	} else {
		attr := *cmd.SysProcAttr
		cmd.SysProcAttr = &attr
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
}

func resumeDriverProcess(process windows.Handle) error {
	if err := ntResumeProcess.Find(); err != nil {
		return fmt.Errorf("resolve NtResumeProcess: %w", err)
	}
	result, _, _ := ntResumeProcess.Call(uintptr(process))
	status := windows.NTStatus(uint32(result))
	if status != windows.STATUS_SUCCESS {
		return fmt.Errorf("resume suspended driver: %w", status)
	}
	return nil
}

func abortSuspendedDriver(cmd *exec.Cmd, process windows.Handle) {
	if process != 0 {
		_ = windows.TerminateProcess(process, 1)
	} else {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}

func terminateDriverJob(job, process windows.Handle) {
	if err := windows.TerminateJobObject(job, 1); err != nil && process != 0 {
		_ = windows.TerminateProcess(process, 1)
	}
}

func waitForDriverJobExit(job windows.Handle) error {
	deadline := time.Now().Add(driverJobExitWait)
	waited := make(map[uintptr]struct{})
	for {
		terminateDriverJob(job, 0)
		pids, err := driverJobProcessIDs(job)
		if err != nil {
			return err
		}
		found := false
		for _, pid := range pids {
			if _, ok := waited[pid]; ok {
				continue
			}
			found = true
			if err := waitForDriverJobProcess(pid, deadline); err != nil {
				return err
			}
			waited[pid] = struct{}{}
		}
		if !found {
			return nil
		}
	}
}

func waitForDriverJobProcess(pid uintptr, deadline time.Time) error {
	process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open driver job process %d: %w", pid, err)
	}
	defer windows.CloseHandle(process)
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return errors.New("driver job did not exit after termination")
	}
	millis := uint32((remaining + time.Millisecond - 1) / time.Millisecond)
	result, err := windows.WaitForSingleObject(process, millis)
	if err != nil {
		return fmt.Errorf("wait for driver job process %d: %w", pid, err)
	}
	if result != windows.WAIT_OBJECT_0 {
		return errors.New("driver job did not exit after termination")
	}
	return nil
}

func driverJobProcessIDs(job windows.Handle) ([]uintptr, error) {
	capacity := 16
	for {
		buffer := make([]byte, 8+capacity*int(unsafe.Sizeof(uintptr(0))))
		err := windows.QueryInformationJobObject(
			job,
			windows.JobObjectBasicProcessIdList,
			uintptr(unsafe.Pointer(&buffer[0])),
			uint32(len(buffer)),
			nil,
		)
		assigned := *(*uint32)(unsafe.Pointer(&buffer[0]))
		count := *(*uint32)(unsafe.Pointer(&buffer[4]))
		if count < assigned {
			capacity = int(assigned)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect driver job processes: %w", err)
		}
		ids := unsafe.Slice((*uintptr)(unsafe.Pointer(&buffer[8])), int(count))
		return append([]uintptr(nil), ids...), nil
	}
}
