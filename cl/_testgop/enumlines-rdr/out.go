package main

import (
	"fmt"
	"github.com/qiniu/x/osx"
	"io"
)

var r io.Reader

func main() {
	for line := range osx.Lines(r) {
		fmt.Println(line)
	}
}
