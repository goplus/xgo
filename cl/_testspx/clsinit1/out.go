package main

type Rect struct {
	w      int
	h      int
	line   int
	color  int
	border float64
}

func (this *Rect) XGo_Init() *Rect {
	this.w, this.h = 10, 20
	this.line = 3
	this.border = 1.2
	return this
}
