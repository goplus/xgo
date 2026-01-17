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

package spx3

const (
	XGoPackage = true
)

type Game struct {
}

func New() *Game {
	return nil
}

func (p *Game) initGame() {}

func (p *Game) Run() {}

type Sprite struct {
}

func (p *Sprite) Name() string {
	return "sprite"
}

func (p *Sprite) Main(name string) {}

type Handler interface {
	Main(name string)
	Classfname() string
	Classclone() Handler
}

func XGot_Game_Main(game interface{ initGame() }, workers ...Handler) {
}
