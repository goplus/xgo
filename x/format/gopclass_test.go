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

func testClass(t *testing.T, name string, pkg string, class string, entry string, src, expect string) {
	t.Run(name, func(t *testing.T) {
		result, err := format.GopClassSource([]byte(src), pkg, class, entry, name)
		if err != nil {
			t.Fatal("format.GopClassSource failed:", err)
		}
		if ret := string(result); ret != expect {
			t.Fatalf("%s => Expect:\n%s\n=> Got:\n%s\n", name, expect, ret)
		}
	})
}

func TestClassSpx(t *testing.T) {
	testClass(t, "spx class", "github.com/goplus/spx", "Calf", "Main", `package main

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
`)

}
