package main

import "fmt"

type Direction uint8

const (
	_None_1 Direction = Direction(uint8(iota))
	North
	South
	East
	West
)

type Priority int

const (
	_None_2 Priority = iota
	Low
	High
)
const None = 0

var d Direction = None
var p Priority = None

func main() {
	fmt.Println(d, p)
}
