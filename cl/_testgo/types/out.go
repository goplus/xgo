package main

import (
	"fmt"
	"math/cmplx"
)

var ToBe bool = false
var MaxInt uint64 = 1<<64 - 1
var z complex128 = cmplx.Sqrt(-5 + 12i)

func main() {
	fmt.Printf("Type: %T Value: %v\n", ToBe, ToBe)
	fmt.Printf("Type: %T Value: %v\n", MaxInt, MaxInt)
	fmt.Printf("Type: %T Value: %v\n", z, z)
}
