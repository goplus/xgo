package main

import "fmt"

type Point struct {
	X_0 int
	X_1 int
}
type Int int

func main() {
	pt := Point{X_0: 2, X_1: 3}
	fmt.Println(pt.X_0, pt.X_1)
	fmt.Println(Int(100))
	x, y := pt.X_0, pt.X_1
	fmt.Println(x, y)
	var pt2 Point
	pt2.X_0, pt2.X_1 = pt.X_0, pt.X_1
	fmt.Println(pt2)
}
