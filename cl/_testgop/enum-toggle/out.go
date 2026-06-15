package main

import "fmt"

type Toggle bool

const (
	Off Toggle = false
	On  Toggle = true
)

func main() {
	fmt.Println(Off, On)
}
