package main

import (
	"fmt"
	"os"
)

type Base struct {
	Value int
}
type Alias = Base
type Chain = Alias
type Reference = *Base
type Anonymous = struct {
	Count int
}
type Outer struct {
	Chain
	*Anonymous
	os.FileInfo
}
type Distinct struct {
	Alias
	Chain
}
type PointerOuter struct {
	Reference
}

func (b Base) Read() int {
	return b.Value
}
func main() {
	outer := Outer{Chain: Chain{Value: 3}, Anonymous: &Anonymous{Count: 5}}
	outer.Value++
	outer.Count += 2
	fmt.Println(outer.Chain.Read(), outer.Anonymous.Count)
	distinct := Distinct{Alias: Alias{Value: 13}, Chain: Chain{Value: 17}}
	fmt.Println(distinct.Alias.Value, distinct.Chain.Value)
	pointer := PointerOuter{Reference: new(Base)}
	pointer.Value = 19
	fmt.Println(pointer.Reference.Value)
}
