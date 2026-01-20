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
	_0 int
	_1 string
}
type Triple struct {
	_0 int
	_1 string
	_2 bool
}
// Named tuple types
type Point struct {
	_0 int
	_1 int
}
type Person struct {
	_0 string
	_1 int
}
// Type shorthand syntax
type Point3D struct {
	_0 int
	_1 int
	_2 int
}
type Mixed struct {
	_0 string
	_1 string
	_2 int
}
// Tuple with array types (covers token.LBRACK case)
type WithArray struct {
	_0 []int
	_1 [5]string
}
// Tuple with pointer types (covers token.MUL case)
type WithPointers struct {
	_0 *int
	_1 *string
}
// Tuple with function types (covers token.FUNC case)
type WithFunc struct {
	_0 func(int) string
	_1 func()
}
// Tuple with channel types (covers token.CHAN case)
type WithChan struct {
	_0 chan int
	_1 <-chan string
}
// Tuple with map types (covers token.MAP case)
type WithMap struct {
	_0 map[string]int
	_1 map[int]bool
}
// Tuple with struct types (covers token.STRUCT case)
type WithStruct struct {
	_0 struct {
		x int
	}
	_1 struct {
		name string
	}
}
// Tuple with interface types (covers token.INTERFACE case)
type WithInterface struct {
	_0 interface{}
	_1 interface {
		Read([]byte) int
	}
}
// Tuple with parenthesized types (covers token.LPAREN case)
type WithParen struct {
	_0 int
	_1 string
}
// Tuple with qualified type names (covers token.PERIOD case)
type WithQualified struct {
	_0 io.Reader
	_1 io.Writer
}
// Tuple with mixed named and array types
type MixedArray struct {
	_0 []int
	_1 int
}
// Single named field tuple
type SingleNamed struct {
	_0 int
}
// Tuple as channel element type
var ch chan struct {
	_0 int
	_1 error
}
// Tuple as map value type
var cache map[string]struct {
	_0 int
	_1 bool
}
// Tuple as slice element type
var pairs []struct {
	_0 string
	_1 int
}
var ken Person

func main() {
	ken._0, ken._1 = "Ken", 18
	ken._1++
	fmt.Println(stringutil.Concat("name: ", ken._0, ", age: ", strconv.Itoa(ken._1)))
}
