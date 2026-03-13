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

	"github.com/goplus/gop/x/format"
)

func testClass(t *testing.T, name string, cfg *format.ClassConfig, src, expect string) {
	t.Run(name, func(t *testing.T) {
		result, err := format.GopClassSource([]byte(src), cfg, name)
		if err != nil {
			t.Fatal("format.GopClassSource failed:", err)
		}
		if ret := string(result); ret != expect {
			t.Fatalf("%s => Expect:\n%s\n=> Got:\n%s\n== end", name, expect, ret)
		}
	})
}

func TestClassSpx(t *testing.T) {
	testClass(t, "spx class", &format.ClassConfig{
		PkgPath:   "github.com/goplus/spx",
		ClassName: "Calf",
		Overload:  map[string]string{"OnMsg__1": "OnMsg"},
	}, `package main

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

onStart => {
	say "Hello Go+"
}

onMsg "tap", => {

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
	testClass(t, "spx project", &format.ClassConfig{
		PkgPath:   "github.com/goplus/spx",
		ClassName: "Game",
		Project:   true,
	}, `package main

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
func (this *Game) reset() {
	this.userScore = 0
	calfPlay = false
	calfDie = false
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
	testClass(t, "gox class", &format.ClassConfig{
		ClassName: "Rect",
		Comments:  true,
	}, `package main

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

// Area is call rect area
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

// Area is call rect area
func Area() float64 {
	return Width * Height
}
`)
}

func TestClassGopt(t *testing.T) {
	testClass(t, "test class", &format.ClassConfig{
		PkgPath:   "github.com/goplus/gop/cl/internal/spx",
		ClassName: "Game",
		Project:   true,
		Gopt: map[string]string{
			"Gopt_Sprite_Clone__0": "Clone",
			"Gopt_Sprite_Clone__1": "Clone",
		},
		Overload: map[string]string{"Broadcast__0": "Broadcast"},
	}, `package main

import "github.com/goplus/gop/cl/internal/spx"

type Game struct {
	*spx.MyGame
	Kai Kai
}
func (this *Game) onInit() {
	spx.Gopt_Sprite_Clone__0(this.Kai)
	this.Broadcast__0("msg1")
}
func (this *Game) MainEntry() {
}
func (this *Game) Main() {
	spx.Gopt_MyGame_Main(this)
}
func main() {
	new(Game).Main()
}
`, `var Kai Kai

func onInit() {
	Kai.clone
	broadcast "msg1"
}


`)
	testClass(t, "test class", &format.ClassConfig{
		PkgPath:   "github.com/goplus/gop/cl/internal/spx",
		ClassName: "Kai",
		Gopt: map[string]string{
			"Gopt_Sprite_Clone__0": "Clone",
			"Gopt_Sprite_Clone__1": "Clone",
		},
		Overload: map[string]string{"Broadcast__0": "Broadcast"},
	}, `package main

import "github.com/goplus/gop/cl/internal/spx"

type info struct {
	x int
	y int
}

type Kai struct {
	spx.Sprite
	*Game
	a int
}

func (this *Kai) onInit() {
	this.a = 1
	spx.Gopt_Sprite_Clone__0(this)
	spx.Gopt_Sprite_Clone__1(this, info{1, 2})
	spx.Gopt_Sprite_Clone__1(this, &info{1, 2})
}
func (this *Kai) onCloned() {
	this.Say("Hi")
}
func (this *Kai) Classfname() string {
	return "Kai"
}
func (this *Kai) Main() {
}
`, `var a int

type info struct {
	x int
	y int
}

func onInit() {
	a = 1
	clone
	clone info{1, 2}
	clone &info{1, 2}
}

func onCloned() {
	say "Hi"
}


`)
}
