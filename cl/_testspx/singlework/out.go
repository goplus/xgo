package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/spx3"
	"github.com/goplus/xgo/cl/internal/spx3/jwt"
)

type Kai struct {
	spx3.Sprite
	*Game
}
type Game struct {
	spx3.Game
	Kai Kai
}

func (this *Game) MainEntry() {
	this.Run()
}
func (this *Game) Main() {
	_xgo_obj0 := &Kai{Game: this}
	spx3.XGot_Game_Main(this, _xgo_obj0)
}
func (this *Kai) Main(_xgo_arg0 string) {
	this.Sprite.Main(_xgo_arg0)
	fmt.Println(jwt.Token("Hi"))
}
func (this *Kai) Classfname() string {
	return "Kai"
}
func (this *Kai) Classclone() spx3.Handler {
	_xgo_ret := *this
	return &_xgo_ret
}
func main() {
	new(Game).Main()
}
