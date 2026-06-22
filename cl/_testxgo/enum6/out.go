package main

import "fmt"

type Direction int

const (
	None            = 0
	North Direction = 1
	South Direction = 2
	East  Direction = 3
	West  Direction = 4
)

type Priority int

var d Direction = None
var p Priority = None

func main() {
	fmt.Println(d, p)
}
