package main

import "fmt"

func f() {
	type Permission int
	const (
		Read Permission = 1 << iota
		Write
		Execute
	)
	var a Permission
	a = Write
	fmt.Println(a)
}
