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
func (this *App) _xgofrag_1() {
	fmt.Println("task a")
}
func (this *App) _xgofrag_2() {
	fmt.Println("util b")
}
func (this *App) Main() {
	flat.XGot_App_Main(&this.App, this._xgo_WorkMain)
}
func (this *App) _xgo_WorkMain() {
	this._xgofrag_1()
	this._xgofrag_2()
}
func main() {
	new(App).Main()
}
