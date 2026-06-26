package main

type Rect struct {
	width int
}

var XGos_Rect_Total int = 200

func (this *Rect) Get() int {
	return this.width + XGos_Rect_Total
}
func (this *Rect) XGo_Init() *Rect {
	this.width = 10
	return this
}
