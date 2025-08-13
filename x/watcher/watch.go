/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

// Package watcher monitors code changes in an XGo workspace.
package watcher

import (
	"github.com/goplus/xgo/x/fsnotify"
)

var (
	debugMod bool
)

const (
	DbgFlagMod = 1 << iota
	DbgFlagAll = DbgFlagMod
)

func SetDebug(dbgFlags int) {
	debugMod = (dbgFlags & DbgFlagMod) != 0
}

// -----------------------------------------------------------------------------------------

type Runner struct {
	w fsnotify.Watcher
	c *Changes
}

func New(root string) Runner {
	w := fsnotify.New()
	c := NewChanges(root)
	return Runner{w: w, c: c}
}

func (p Runner) Fetch(fullPath bool) (dir string) {
	return p.c.Fetch(fullPath)
}

func (p Runner) Run() error {
	root := p.c.root
	root = root[:len(root)-1]
	return p.w.Run(root, p.c, p.c.Ignore)
}

func (p *Runner) Close() error {
	return p.w.Close()
}

// -----------------------------------------------------------------------------------------
