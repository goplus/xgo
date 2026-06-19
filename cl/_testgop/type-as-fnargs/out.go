package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/typeargs"
)

func main() {
	fmt.Println(typeargs.XGox_Convert[string, int](100))
}
