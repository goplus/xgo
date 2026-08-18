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
	"os"
	"os/signal"
	"sync"
)

type runtimeSignalBoundary struct {
	ctx       context.Context
	cancel    context.CancelFunc
	signals   chan os.Signal
	done      chan struct{}
	wait      sync.WaitGroup
	mu        sync.Mutex
	interrupt bool
}

func beginRuntimeSignalBoundary(parent context.Context) *runtimeSignalBoundary {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	b := &runtimeSignalBoundary{ctx: ctx, cancel: cancel, signals: make(chan os.Signal, 4), done: make(chan struct{})}
	signal.Notify(b.signals, os.Interrupt)
	b.wait.Add(1)
	go func() {
		defer b.wait.Done()
		select {
		case <-b.signals:
			b.mu.Lock()
			b.interrupt = true
			b.mu.Unlock()
			b.cancel()
		case <-b.done:
		}
	}()
	return b
}

func (b *runtimeSignalBoundary) Context() context.Context { return b.ctx }

func (b *runtimeSignalBoundary) Finish(status ProcessStatus, err error) (ProcessStatus, error) {
	signal.Stop(b.signals)
	close(b.done)
	b.wait.Wait()
	b.cancel()
	b.mu.Lock()
	interrupted := b.interrupt
	b.mu.Unlock()
	if interrupted {
		return ProcessStatus{Code: 130}, nil
	}
	return status, err
}
