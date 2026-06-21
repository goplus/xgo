package main

import "fmt"

const XGoo_mul = "mulInt,mulFloat"

func mulInt(a int, b int) int {
	return a * b
}
func mulFloat(a float64, b float64) float64 {
	return a * b
}
func main() {
	fmt.Println(mulInt(100, 7))
	fmt.Println(mulFloat(1.2, 3.14))
}
