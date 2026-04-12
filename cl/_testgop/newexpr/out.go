package main

import "fmt"

func main() {
	a := new(42)
	b := new("hello")
	fmt.Println(*a, *b)
}
