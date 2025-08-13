/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package test

import (
	"os"
	"testing"
)

const (
	GopPackage = true
)

// -----------------------------------------------------------------------------

type testingT = testing.T

// Case represents an XGo testcase.
type Case struct {
	t *testingT
}

func (p *Case) initCase(t *testing.T) {
	p.t = t
}

// T returns the *testing.T object .
func (p Case) T() *testing.T { return p.t }

// Run runs f as a subtest of t called name. It runs f in a separate goroutine
// and blocks until f returns or calls t.Parallel to become a parallel test.
// Run reports whether f succeeded (or at least did not fail before calling t.Parallel).
func (p Case) Run(name string, f func(t *testing.T)) bool {
	return p.t.Run(name, f)
}

// Gopt_Case_TestMain is required by XGo compiler as the test case entry.
func Gopt_Case_TestMain(c interface{ initCase(t *testing.T) }, t *testing.T) {
	c.initCase(t)
	c.(interface{ Main() }).Main()
}

// -----------------------------------------------------------------------------

// App represents an XGo testing main application.
type App struct {
	m *testing.M
}

func (p *App) initApp(m *testing.M) {
	p.m = m
}

// M returns the *testing.M object.
func (p App) M() *testing.M { return p.m }

// Gopt_App_TestMain is required by XGo compiler as the entry of an XGo testing project.
func Gopt_App_TestMain(app interface{ initApp(m *testing.M) }, m *testing.M) {
	app.initApp(m)
	if me, ok := app.(interface{ MainEntry() }); ok {
		me.MainEntry()
	}
	os.Exit(m.Run())
}

// -----------------------------------------------------------------------------
