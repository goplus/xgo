package main

import "fmt"

type foo struct {
}

const XGoo_foo_mul = ".mulInt,.mulFoo"

func (a *foo) mulInt(b int) *foo {
	fmt.Println("mulInt")
	return a
}
func (a *foo) mulFoo(b *foo) *foo {
	fmt.Println("mulFoo")
	return a
}

var a, b foo
var c = a.mulInt(100)
var d = a.mulFoo(c)
