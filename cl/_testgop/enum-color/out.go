package main

import "fmt"

type Color int

const (
	Red Color = iota
	Green
	Blue
)

func main() {
	fmt.Println(Red, Green, Blue)
}
