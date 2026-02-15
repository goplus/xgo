package main

import (
	"fmt"
	"github.com/goplus/xgo/encoding/json"
	"github.com/goplus/xgo/encoding/yaml"
)

func main() {
	fmt.Println(json.New(`{"a":1, "b":2}`))
	fmt.Println(yaml.New(`
a: 1
b: c
`))
}
