package main

import "github.com/goplus/xgo/cl/internal/spx"

type _init struct {
	spx.Sprite
	*MyGame
}
type MyGame struct {
	*spx.MyGame
}

func (this *MyGame) Main() {
	spx.XGot_MyGame_Main(this)
}
func (this *_init) Main() {
}
func main() {
	new(MyGame).Main()
}
