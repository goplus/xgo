package main

import (
	"fmt"
	"github.com/qiniu/x/osx"
	"io"
)

var r io.Reader

func main() {
	for _xgo_it := osx.Lines(r).Gop_Enum(); ; {
		var _xgo_ok bool
		line, _xgo_ok := _xgo_it.Next()
		if !_xgo_ok {
			break
		}
		fmt.Println(line)
	}
}
