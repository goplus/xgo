package main

import "fmt"

type Point struct {
	_0 int
	_1 int
}
type Int int

func main() {
	pt := Point{2, 3}
	fmt.Println(pt._0, pt._1)
	fmt.Println(Int(100))
}
