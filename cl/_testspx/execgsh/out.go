package main

import "github.com/qiniu/x/gsh"

type demo struct {
	gsh.App
}

func (this *demo) MainEntry() {
	this.XGo_Exec("xgo", "run", "./foo")
	this.Exec__1("ls $HOME")
	this.XGo_Exec("ls", this.XGo_Env("HOME"))
}
func (this *demo) Main() {
	gsh.XGot_App_Main(this)
}
func main() {
	new(demo).Main()
}
