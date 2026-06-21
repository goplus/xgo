package main

import "fmt"

type foo struct {
	a []int
}

func main() {
	a := []int{1, 2, 3}
	a = append(a, 4)
	fmt.Println(a)
	f := foo{a: []int{1, 2, 3}}
	f.a = append(f.a, 4)
	fmt.Println(f)
	f2 := [2]chan int{}
	f2[0] <- 4
}
