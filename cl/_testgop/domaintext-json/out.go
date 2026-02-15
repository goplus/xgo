package main

import (
	"fmt"
	"github.com/goplus/xgo/encoding/json"
)

func main() {
	fmt.Println(json.New(`{"a":1, "b":2}`))
}
