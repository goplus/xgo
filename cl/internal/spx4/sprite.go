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

package spx4

import (
	"github.com/goplus/xgo/cl/internal/spx/pkg"
)

type Sprite struct {
	pos pkg.Vector
	Entry
}

type Entry struct {
	vec pkg.Vector
}

func (p *Entry) Vector() *pkg.Vector {
	return &p.vec
}

func (p *Sprite) SetCostume(costume any) {
}

func (p *Sprite) Say(msg string, secs ...float64) {
}

func (p *Sprite) Position() *pkg.Vector {
	return &p.pos
}

type Mesher interface {
	Name() string
}

func XGot_Sprite_Clone__0(sprite any) {
}

func XGot_Sprite_Clone__1(sprite any, data any) {
}

func XGot_Sprite_OnKey__0(sprite any, a string, fn func()) {
}

func XGot_Sprite_OnKey__1(sprite any, a string, fn func(key string)) {
}

func XGot_Sprite_OnKey__2(sprite any, a []string, fn func()) {
}

func XGot_Sprite_OnKey__3(sprite any, a []string, fn func(key string)) {
}

func XGot_Sprite_OnKey__4(sprite any, a []Mesher, fn func()) {
}

func XGot_Sprite_OnKey__5(sprite any, a []Mesher, fn func(key Mesher)) {
}

func XGot_Sprite_OnKey__6(sprite any, a []string, b []string, fn func(key string)) {
}

func XGot_Sprite_OnKey__7(sprite any, a []string, b []Mesher, fn func(key string)) {
}

func XGot_Sprite_OnKey__8(sprite any, x int, y int) {
}

func XGot_Sprite_OnKey2(sprite any, a string, fn func(key string)) {
}
