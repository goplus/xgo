package main

import (
	"fmt"
	"reflect"
)

func accept(...reflect.Type) {
}
func main() {
	fmt.Println(reflect.New(reflect.TypeFor[int]()).Elem())
	accept(reflect.TypeFor[*int](), reflect.TypeFor[bool]())
}
