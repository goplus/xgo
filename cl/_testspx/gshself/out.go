package main

import (
	"github.com/goplus/xgo/cl/internal/dql"
	"github.com/qiniu/x/gsh"
)

type demo struct {
	gsh.App
}

func (this *demo) MainEntry() {
	self := dql.Node{}
	this.XGo_Exec("ls", this.XGo_Env("HOME"))
}
func (this *demo) Main() {
	gsh.XGot_App_Main(this)
}
func main() {
	new(demo).Main()
}
