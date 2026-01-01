package main

import "fmt"

type Rect struct {
	Width  float64
	Height float64
}

func (this *Rect) Gop_Init() {
	this.Width = 100.0
	this.Height = 200.0
}
func (this *Rect) Area() float64 {
	return this.Width * this.Height
}
func main() {
	r := &Rect{Width: 10, Height: 20}
	r.Gop_Init()
	fmt.Println(r.Area())
}
