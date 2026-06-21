package main

import "fmt"

type foo struct {
}

const XGoo__foo__XGo_Mul = ".mulInt,.mulFoo,intMulFoo"

func (a foo) mulInt(b int) (ret foo) {
	return
}
func (a foo) mulFoo(b foo) (ret foo) {
	return
}
func intMulFoo(a int, b foo) (ret foo) {
	return
}

var a, b foo

func main() {
	fmt.Println((foo).mulInt(a, 10))
	fmt.Println((foo).mulFoo(a, b))
	fmt.Println(intMulFoo(10, a))
}
