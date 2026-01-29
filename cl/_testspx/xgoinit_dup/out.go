package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/spx3"
)

type Spr struct {
	spx3.Sprite
	*Game
}
type Game struct {
	spx3.Game
	score int
	Spr   Spr
}

func (this *Game) MainEntry() {
	this.XGo_Init()
	this.Run()
}
func (this *Game) Main() {
	_xgo_obj0 := &Spr{Game: this}
	spx3.Gopt_Game_Main(this, _xgo_obj0)
}
func (this *Game) XGo_Init() *Game {
	this.score = 100
	return this
}
func (this *Spr) Main(_xgo_arg0 string) {
	this.Sprite.Main(_xgo_arg0)
	fmt.Println("sprite main called")
}
func (this *Spr) Classfname() string {
	return "Spr"
}
func (this *Spr) Classclone() spx3.Handler {
	_xgo_ret := *this
	return &_xgo_ret
}
func main() {
	new(Game).Main()
}
