package main

type Rect struct {
	w      int
	h      int
	color  int
	border float64
}

func (_xgo_this *Rect) XGo_Init() *Rect {
	_xgo_this.w, _xgo_this.h = 10, 20
	_xgo_this.border = 1.2
	return _xgo_this
}
