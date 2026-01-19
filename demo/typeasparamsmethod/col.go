package main

import (
	"fmt"
	"reflect"
)

type basetype interface {
	int | string
}

type Table struct {
}

func XGot_Table_XGox_Col__0[T basetype](p *Table, name string) {
	fmt.Printf("XGot_Table_XGox_Col__0 %v: %s\n", reflect.TypeOf((*T)(nil)).Elem(), name)
}

func XGot_Table_XGox_Col__1[Array any](p *Table, name string) {
	fmt.Printf("XGot_Table_XGox_Col__1 %v: %s\n", reflect.TypeOf((*Array)(nil)).Elem(), name)
}
