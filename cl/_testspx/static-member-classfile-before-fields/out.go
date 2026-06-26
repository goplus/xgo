package main

type Rect struct {
	width int
}

var XGos_Rect_Count int = 100

func (this *Rect) Get() int {
	return this.width
}
func (this *Rect) XGo_Init() *Rect {
	this.width = 10
	return this
}
