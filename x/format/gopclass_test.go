/*
 * Copyright (c) 2024 The GoPlus Authors (goplus.org). All rights reserved.
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

package format_test

import (
	"testing"

	"github.com/goplus/xgo/x/format"
)

func testClass(t *testing.T, name string, pkg string, class string, proj bool, src, expect string) {
	t.Run(name, func(t *testing.T) {
		result, err := format.XGoClassSource([]byte(src), pkg, class, proj, name)
		if err != nil {
			t.Fatal("format.XGoClassSource failed:", err)
		}
		if ret := string(result); ret != expect {
			t.Fatalf("%s => Expect:\n%s\n=> Got:\n%s\n", name, expect, ret)
		}
	})
}

func TestClassSpx(t *testing.T) {
	testClass(t, "spx class", "github.com/goplus/spx", "Calf", false, `package main

import (
	"github.com/goplus/spx"
	"fmt"
	"log"
)

type Calf struct {
	spx.Sprite
	*Game
	index int
	info  string
}
func (this *Calf) Update() {
	this.index++
}
func (this *Calf) Dump() {
	log.Println(this.info)
}
func (this *Calf) Main() {
	this.OnStart(func() {
		this.Say("Hello Go+")
	})
//line Calf.spx:38:1
	this.OnMsg__1("tap", func() {
//line Calf.spx:39:1
		for calfPlay {
			spx.Sched()
//line Calf.spx:40:1
			for !(this.KeyPressed(spx.KeySpace) || this.MousePressed()) {
				spx.Sched()
//line Calf.spx:41:1
				this.Wait(0.01)
			}
//line Calf.spx:43:1
			calfGravity = 0.8
//line Calf.spx:44:1
			for
//line Calf.spx:44:1
			i := 0; i < 10;
//line Calf.spx:44:1
			i++ {
				spx.Sched()
//line Calf.spx:45:1
				this.ChangeYpos(3.5)
//line Calf.spx:46:1
				this.Wait(0.03)
			}
//line Calf.spx:48:1
			this.Wait(0.03)
		}
	})
}
func (this *Calf) Classfname() string {
	return "Calf"
}
`, `import (
	"log"
)

var (
	index int
	info  string
)

func Update() {
	index++
}

func Dump() {
	log.println info
}

onStart func() {
	say "Hello Go+"
}

onMsg__1 "tap", func() {

	for calfPlay {

		for !(keyPressed(KeySpace) || mousePressed()) {

			wait 0.01
		}

		calfGravity = 0.8

		for i := 0; i < 10; i++ {

			changeYpos 3.5

			wait 0.03
		}

		wait 0.03
	}
}
`)
}

func TestClassProj(t *testing.T) {
	testClass(t, "spx project", "github.com/goplus/spx", "Game", true, `package main

import "github.com/goplus/spx"
import "log"

type Game struct {
	spx.Game
	MyAircraft MyAircraft
	Bullet     Bullet
}

var calfPlay = false
var calfDie = false
var calfGravity = 0.0
//line main.spx:30:1
func (this *Game) reset() {
//line main.spx:31:1
	this.userScore = 0
//line main.spx:32:1
	calfPlay = false
//line main.spx:33:1
	calfDie = false
//line main.spx:34:1
	calfGravity = 0.0
}
func (this *Game) MainEntry() {
	log.Println("MainEntry")
}
func (this *Game) Main() {
	spx.Gopt_Game_Main(this, new(Bullet), new(MyAircraft))
}
func main() {
	new(Game).Main()
}
`, `import "log"

var (
	MyAircraft MyAircraft
	Bullet     Bullet
)

var calfPlay = false
var calfDie = false
var calfGravity = 0.0

func reset() {

	userScore = 0

	calfPlay = false

	calfDie = false

	calfGravity = 0.0
}

log.println "MainEntry"
`)
}

func TestClassGox(t *testing.T) {
	testClass(t, "gox class", "", "Rect", false, `package main

type BaseClass struct {
	x int
	y int
}
type AggClass struct {
}
type Rect struct {
	BaseClass
	Width  float64
	Height float64
	*AggClass
}

func (this *Rect) Area() float64 {
	return this.Width * this.Height
}
`, `var (
	BaseClass
	Width  float64
	Height float64
	*AggClass
)

type BaseClass struct {
	x int
	y int
}

type AggClass struct {
}

func Area() float64 {
	return Width * Height
}
`)
}
