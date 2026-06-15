package main

import "fmt"

type Permission int

const (
	Read Permission = 1 << iota
	Write
	Execute
)

func main() {
	fmt.Println(Read, Write, Execute)
}
