package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/typeargs"
)

func main() {
	fmt.Println(typeargs.XGox_As[int]("100"))
	fmt.Println(typeargs.XGox_Convert[string, int](100))
}
