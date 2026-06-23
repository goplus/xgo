package main

import "fmt"

type Direction int

const (
	_None_1 Direction = 0
	North   Direction = 1
	South   Direction = 2
	East    Direction = 3
	West    Direction = 4
)

type Priority int

const (
	_None_2 Priority = 0
	Low     Priority = 1
	High    Priority = 2
)
const None = 0

var d Direction = None
var p Priority = None

func main() {
	fmt.Println(d, p)
}
