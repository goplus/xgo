package main

import (
	"fmt"
	"github.com/qiniu/x/stringutil"
	"io"
	"strconv"
)
// Empty tuple
type Empty struct {
}
// Anonymous tuple types
type Pair struct {
	X_0 int
	X_1 string
}
type Triple struct {
	X_0 int
	X_1 string
	X_2 bool
}
// Named tuple types
type Point struct {
	X_0 int
	X_1 int
}
type Person struct {
	X_0 string
	X_1 int
}
// Type shorthand syntax
type Point3D struct {
	X_0 int
	X_1 int
	X_2 int
}
type Mixed struct {
	X_0 string
	X_1 string
	X_2 int
}
// Tuple with array types (covers token.LBRACK case)
type WithArray struct {
	X_0 []int
	X_1 [5]string
}
// Tuple with pointer types (covers token.MUL case)
type WithPointers struct {
	X_0 *int
	X_1 *string
}
// Tuple with function types (covers token.FUNC case)
type WithFunc struct {
	X_0 func(int) string
	X_1 func()
}
// Tuple with channel types (covers token.CHAN case)
type WithChan struct {
	X_0 chan int
	X_1 <-chan string
}
// Tuple with map types (covers token.MAP case)
type WithMap struct {
	X_0 map[string]int
	X_1 map[int]bool
}
// Tuple with struct types (covers token.STRUCT case)
type WithStruct struct {
	X_0 struct {
		x int
	}
	X_1 struct {
		name string
	}
}
// Tuple with interface types (covers token.INTERFACE case)
type WithInterface struct {
	X_0 interface{}
	X_1 interface {
		Read([]byte) int
	}
}
// Tuple with parenthesized types (covers token.LPAREN case)
type WithParen struct {
	X_0 int
	X_1 string
}
// Tuple with qualified type names (covers token.PERIOD case)
type WithQualified struct {
	X_0 io.Reader
	X_1 io.Writer
}
// Tuple with mixed named and array types
type MixedArray struct {
	X_0 []int
	X_1 int
}
// Single named field tuple
type SingleNamed int
// Tuple as channel element type
var ch chan struct {
	X_0 int
	X_1 error
}
// Tuple as map value type
var cache map[string]struct {
	X_0 int
	X_1 bool
}
// Tuple as slice element type
var pairs []struct {
	X_0 string
	X_1 int
}
var ken Person

func main() {
	ken.X_0, ken.X_1 = "Ken", 18
	ken.X_1++
	fmt.Println(stringutil.Concat("name: ", ken.X_0, ", age: ", strconv.Itoa(ken.X_1)))
}
