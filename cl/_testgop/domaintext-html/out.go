package main

import (
	"fmt"
	"github.com/goplus/xgo/encoding/html"
)

func main() {
	fmt.Println(html.New(`<html><body><h1>hello</h1></body></html>`))
}
