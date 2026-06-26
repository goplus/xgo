package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/flat"
)

type App struct {
	flat.App
}

func (this *App) MainEntry() {
	fmt.Println("hello from main")
}
func (this *App) Main() {
	flat.XGot_App_Main(&this.App, nil)
}
func main() {
	new(App).Main()
}
