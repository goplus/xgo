package main

import "fmt"

func returnValue(__xgo_optional_x int) int {
	return __xgo_optional_x
}
func useInExpression(__xgo_optional_a int, __xgo_optional_b int) int {
	result := __xgo_optional_a + __xgo_optional_b
	return result * 2
}
func simpleNested(__xgo_optional_outer int) {
	f := func() {
		fmt.Println(__xgo_optional_outer)
	}
	f()
}
func main() {
	returnValue(42)
	useInExpression(10, 5)
	simpleNested(100)
}
