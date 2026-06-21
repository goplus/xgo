package main

import "fmt"

const XGoo_add = ",addInt,addFloat"

func add__0(a string, b string) string {
	return a + b
}
func addInt(a int, b int) int {
	return a + b
}
func addFloat(a float64, b float64) float64 {
	return a + b
}
func main() {
	fmt.Println(addInt(100, 7))
	fmt.Println(addFloat(1.2, 3.14))
}
