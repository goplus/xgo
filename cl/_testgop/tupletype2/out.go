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
	pt2 := Point{100, 200}
	fmt.Println(pt2)
	pt3 := Point(Point{X_1: 5, X_0: 3})
	fmt.Println(pt3.X_0, pt3.X_1)
}
