package main

import "github.com/goplus/xgo/cl/internal/spx"

type Game struct {
	*spx.MyGame
}
type Kai struct {
	spx.Sprite
	*Game
}

func (this *Game) onInit() {
	for {
		spx.SchedNow()
	}
}
func (this *Game) MainEntry() {
	this.InitGameApp()
}
func (this *Game) Main() {
	spx.XGot_MyGame_Main(this)
}
func (this *Kai) onMsg(msg string) {
	for {
		spx.Sched()
		this.Say("Hi")
	}
}
func (this *Kai) Main() {
}
func main() {
	new(Game).Main()
}
