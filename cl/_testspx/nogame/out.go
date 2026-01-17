package main

import "github.com/goplus/xgo/cl/internal/spx"

type bar struct {
	spx.Sprite
	*MyGame
}
type MyGame struct {
	*spx.MyGame
}

func (this *MyGame) Main() {
	spx.XGot_MyGame_Main(this)
}
func (this *bar) Main() {
}
func main() {
	new(MyGame).Main()
}
