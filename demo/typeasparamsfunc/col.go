package main

import (
	"fmt"
	"reflect"
)

func XGox_Col[T any](name string) {
	fmt.Printf("%v: %s\n", reflect.TypeOf((*T)(nil)).Elem(), name)
}
