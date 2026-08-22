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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type driverSignalCause struct {
	signal syscall.Signal
}

func (c driverSignalCause) Error() string {
	return fmt.Sprintf("driver interrupted by %s", c.signal)
}

type driverSignalBoundary struct {
	ctx     context.Context
	cancel  context.CancelCauseFunc
	signals chan os.Signal
	done    chan struct{}
	wait    sync.WaitGroup
	mu      sync.Mutex
	signal  syscall.Signal
}

func beginDriverSignalBoundary(parent context.Context) *driverSignalBoundary {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancelCause(parent)
	b := &driverSignalBoundary{
		ctx: ctx, cancel: cancel, signals: make(chan os.Signal, 8), done: make(chan struct{}),
	}
	signal.Notify(b.signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	b.wait.Add(1)
	go func() {
		defer b.wait.Done()
		for {
			select {
			case received := <-b.signals:
				unixSignal, ok := received.(syscall.Signal)
				if !ok {
					continue
				}
				b.mu.Lock()
				if b.signal == 0 {
					b.signal = unixSignal
					b.cancel(driverSignalCause{signal: unixSignal})
				}
				b.mu.Unlock()
			case <-b.done:
				return
			}
		}
	}()
	return b
}

func (b *driverSignalBoundary) Context() context.Context { return b.ctx }

func (b *driverSignalBoundary) Finish(status ProcessStatus, err error) (ProcessStatus, error) {
	signal.Stop(b.signals)
	close(b.done)
	b.wait.Wait()
	b.cancel(nil)
	b.mu.Lock()
	received := b.signal
	b.mu.Unlock()
	if received != 0 {
		return ProcessStatus{Signal: received, Signaled: true}, nil
	}
	return status, err
}
