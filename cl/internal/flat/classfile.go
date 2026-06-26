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

// Package flat provides a simple flat-mode classfile framework for testing.
package flat

const (
	GopPackage = true
)

// App is the project base class for the flat-mode test framework.
type App struct{}

func (p *App) Init() {}
func (p *App) Run()  {}

// XGot_App_Main is the template receiver method for the flat-mode App class.
// workMain is the synthesized _xgo_WorkMain method, or nil if no fragment files.
func XGot_App_Main(app *App, workMain func()) {
	app.Init()
	if workMain != nil {
		workMain()
	}
	app.Run()
}
