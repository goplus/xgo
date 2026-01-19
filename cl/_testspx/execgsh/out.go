package main

import (
	"fmt"
	"github.com/qiniu/x/gsh"
)

type demo struct {
	gsh.App
	score int
}

func (this *demo) MainEntry() {
	this.XGo_Init()
	fmt.Println(this.score)
	this.XGo_Exec("xgo", "run", "./foo")
	this.Exec__1("ls $HOME")
	this.XGo_Exec("ls", this.XGo_Env("HOME"))
}
func (this *demo) Main() {
	gsh.XGot_App_Main(this)
}
func (this *demo) XGo_Init() *demo {
	this.score = 100
	return this
}
func main() {
	new(demo).Main()
}
