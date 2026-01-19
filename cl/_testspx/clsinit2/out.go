package main

import "fmt"

type Rect struct {
	w      int
	h      int
	color  int
	border float64
}

func (this *Rect) Main() {
	this.XGo_Init()
	fmt.Println(*this)
}
func (this *Rect) XGo_Init() *Rect {
	this.w, this.h = 10, 20
	this.border = 1.2
	return this
}
func main() {
	new(Rect).Main()
}
