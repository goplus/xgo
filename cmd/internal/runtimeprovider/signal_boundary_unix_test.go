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
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRuntimeSignalBoundaryRecordsSignalAndCancelsWork(t *testing.T) {
	boundary := beginRuntimeSignalBoundary(context.Background())
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-boundary.Context().Done():
	case <-time.After(5 * time.Second):
		t.Fatal("runtime signal boundary did not cancel work")
	}
	cause, ok := context.Cause(boundary.Context()).(runtimeSignalCause)
	if !ok || cause.signal != syscall.SIGTERM {
		t.Fatalf("context cause = %#v, want SIGTERM runtime signal", context.Cause(boundary.Context()))
	}
	status, err := boundary.Finish(ProcessStatus{}, context.Canceled)
	if err != nil || !status.Signaled || status.Signal != syscall.SIGTERM {
		t.Fatalf("Finish() = (%+v, %v), want SIGTERM status", status, err)
	}
}
