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
	"os/exec"
)

// Exit terminates a command handler with the exact driver status. Callers
// must invoke it only after all driver/output cleanup has completed.
func Exit(status ProcessStatus) {
	if status.Signaled {
		exitWithSignal(status.Signal)
	}
	os.Exit(status.Code)
}

func processStatus(err error) (ProcessStatus, error) {
	if err == nil {
		return successStatus(), nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return ProcessStatus{}, err
	}
	status := ProcessStatus{Code: exitErr.ExitCode()}
	if signal, ok := exitSignal(exitErr); ok {
		status.Signal, status.Signaled = signal, true
	}
	return status, nil
}

func statusUnlessCanceled(ctx context.Context, status ProcessStatus) (ProcessStatus, error) {
	if !status.Signaled && status.Code == 0 {
		if cause := context.Cause(ctx); cause != nil {
			return ProcessStatus{}, cause
		}
	}
	return status, nil
}

func driverExitStatus(ctx context.Context, waitErr error) (ProcessStatus, error) {
	status, err := processStatus(waitErr)
	if err != nil {
		return ProcessStatus{}, err
	}
	return statusUnlessCanceled(ctx, status)
}
