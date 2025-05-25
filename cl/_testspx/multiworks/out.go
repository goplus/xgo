package main

import "github.com/goplus/xgo/cl/internal/mcp"

type foo struct {
	mcp.Prompt
	*Game
}
type Tool_hello struct {
	mcp.Tool
	*Game
}
type Game struct {
	mcp.Game
	foo *foo
}

func (this *Game) MainEntry() {
	this.Server("protos")
}
func (this *Game) Main() {
	_xgo_obj0 := &Tool_hello{Game: this}
	_xgo_lst1 := []mcp.ToolProto{_xgo_obj0}
	_xgo_obj1 := &foo{Game: this}
	this.foo = _xgo_obj1
	_xgo_lst2 := []mcp.PromptProto{_xgo_obj1}
	mcp.Gopt_Game_Main(this, nil, _xgo_lst1, _xgo_lst2)
}
func (this *foo) Main(_xgo_arg0 *mcp.Tool) string {
	this.Prompt.Main(_xgo_arg0)
	return "Hi"
}
func (this *Tool_hello) Main(_xgo_arg0 string) int {
	this.Tool.Main(_xgo_arg0)
	return -1
}
func main() {
	new(Game).Main()
}
