package main

import "fmt"

type foo struct {
}

func (a *foo) XGo_Add(b *foo) *foo {
	fmt.Println("a + b")
	return &foo{}
}
func (a foo) XGo_Sub(b foo) foo {
	fmt.Println("a - b")
	return foo{}
}
func (a foo) XGo_NE(b foo) bool {
	fmt.Println("a!=b")
	return true
}
func (a foo) XGo_Neg() {
	fmt.Println("-a")
}
func (a foo) XGo_Inc() {
	fmt.Println("a++")
}

var a, b foo
var c = (foo).XGo_Sub(a, b)
var d = a.XGo_Neg()
var e = (foo).XGo_NE(a, b)
