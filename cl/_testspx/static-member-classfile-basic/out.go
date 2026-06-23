package main

const XGos_Rect_Name = "rect"

type Rect struct {
}

var XGos_Rect_Count int = 100
var XGos_Rect_Total int = 200

func (this *Rect) Get() string {
	return XGos_Rect_Name
}
func (this *Rect) Inc() {
	XGos_Rect_Count++
	XGos_Rect_Total++
}
