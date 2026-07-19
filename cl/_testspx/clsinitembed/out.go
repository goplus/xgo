package main

import "github.com/goplus/xgo/cl/internal/mcp"

type Bar struct {
	mcp.Prompt
	*Game
}
type Foo struct {
	mcp.Prompt
	*Game
	others []*Bar
}
type Game struct {
	mcp.Game
	Bar *Bar
	Foo *Foo
}

func (this *Game) MainEntry() {
	this.Server("protos")
}
func (this *Game) Main() {
	_xgo_obj0 := &Bar{Game: this}
	this.Bar = _xgo_obj0
	_xgo_obj1 := &Foo{Game: this}
	this.Foo = _xgo_obj1
	_xgo_lst2 := []mcp.PromptProto{_xgo_obj0, _xgo_obj1}
	mcp.Gopt_Game_Main(this, nil, nil, _xgo_lst2)
}
func (this *Bar) Main(_xgo_arg0 *mcp.Tool) string {
	this.Prompt.Main(_xgo_arg0)
	return "Bar"
}
func (this *Foo) Main(_xgo_arg0 *mcp.Tool) string {
	this.XGo_Init()
	this.Prompt.Main(_xgo_arg0)
	return "Foo"
}
func (this *Foo) XGo_Init() *Foo {
	this.others = []*Bar{this.Bar}
	return this
}
func main() {
	new(Game).Main()
}
