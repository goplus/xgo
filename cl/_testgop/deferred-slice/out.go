package main

import "fmt"

func main() {
	x := []int{}
	x = append(x, 100)
	x = append(x, 200)
	fmt.Println(x[0] + x[1])
}
