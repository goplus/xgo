//go:build !darwin && !linux

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
	"os"
	"os/exec"
	"os/signal"
)

type driverSignalBoundary struct {
	ctx       context.Context
	cancel    context.CancelFunc
	signals   chan os.Signal
	done      chan struct{}
	watchDone chan struct{}
	interrupt bool
}

func beginDriverSignalBoundary(parent context.Context) *driverSignalBoundary {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	b := &driverSignalBoundary{
		ctx: ctx, cancel: cancel, signals: make(chan os.Signal, 1), done: make(chan struct{}), watchDone: make(chan struct{}),
	}
	signal.Notify(b.signals, os.Interrupt)
	go func() {
		defer close(b.watchDone)
		select {
		case <-b.signals:
			b.interrupt = true
			b.cancel()
		case <-b.done:
		}
	}()
	return b
}

func (b *driverSignalBoundary) Context() context.Context { return b.ctx }

func (b *driverSignalBoundary) Finish(status ProcessStatus, err error) (ProcessStatus, error) {
	signal.Stop(b.signals)
	close(b.done)
	<-b.watchDone
	b.cancel()
	if b.interrupt {
		return ProcessStatus{Code: 130}, nil
	}
	return status, err
}

func exitWithSignal(os.Signal) { os.Exit(1) }

func exitSignal(*exec.ExitError) (os.Signal, bool) { return nil, false }
