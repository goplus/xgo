package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/flat"
)

type App struct {
	flat.App
}

func (this *App) MainEntry() {
	fmt.Println("main entry")
}
func (this *App) _xgo_workfrag_0() {
	fmt.Println("task a")
}
func (this *App) _xgo_workfrag_1() {
	fmt.Println("util b")
}
func (this *App) _xgo_WorkMain() {
	this._xgo_workfrag_0()
	this._xgo_workfrag_1()
}
func (this *App) Main() {
	flat.XGot_App_Main(&this.App, this._xgo_WorkMain)
}
func main() {
	new(App).Main()
}
