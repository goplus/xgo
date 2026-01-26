/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
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

package spx

const (
	GopPackage = "github.com/goplus/xgo/cl/internal/spx/pkg"
	Gop_sched  = "Sched,SchedNow"
)

type Sound string

type MyGame struct {
}

func Gopt_MyGame_Main(game any) {
}

func (p *MyGame) Ls(n int) {}

func (p *MyGame) Capout(doSth func()) (string, error) {
	return "", nil
}

func (p *MyGame) Gop_Env(name string) int {
	return 0
}

func (p *MyGame) Gop_Exec(name string, args ...any) {
}

func (p *MyGame) InitGameApp(args ...string) {
}

func (p *MyGame) Broadcast__0(msg string) {
}

func (p *MyGame) Broadcast__1(msg string, wait bool) {
}

func (p *MyGame) Broadcast__2(msg string, data any, wait bool) {
}

func (p *MyGame) Play(media string, wait ...bool) {
}

func (p *MyGame) sendMessage(data any) {
}

func (p *MyGame) SendMessage(data any) {
	p.sendMessage(data)
}

func Gopt_MyGame_Run(game any, resource string) error {
	return nil
}

func Sched() {
}

func SchedNow() {
}

func Rand__0(int) int {
	return 0
}

func Rand__1(float64) float64 {
	return 0
}

var (
	TestIntValue int
)
