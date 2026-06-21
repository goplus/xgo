package main

import (
	"fmt"
	"github.com/qiniu/x/stringslice"
)

func main() {
	a := []string{"hello", "world", "123"}
	fmt.Println(stringslice.Capitalize(a))
}
