package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/spx"
)

type Game struct {
	*spx.MyGame
	Kai   Kai
	score int
	name  string
}
type Kai struct {
	spx.Sprite
	*Game
	hp    int
	level int
}

func (this *Game) MainEntry() {
	this.XGo_Init()
	fmt.Println(this.score, this.name)
}
func (this *Game) XGo_Init() *Game {
	this.score = 100
	this.name = "Game"
	return this
}
func (this *Game) Main() {
	spx.Gopt_MyGame_Main(this)
}
func (this *Kai) Main() {
	this.XGo_Init()
	fmt.Println(this.hp, this.level)
}
func (this *Kai) XGo_Init() *Kai {
	this.hp = 50
	this.level = 1
	return this
}
func main() {
	new(Game).Main()
}
