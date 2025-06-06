package main

import (
	"fmt"
	"os"

	"github.com/qiniu/x/osx"
)

func main() {
	for _xgo_it := osx.EnumLines(os.Stdin); ; {
		var _xgo_ok bool
		line, _xgo_ok := _xgo_it.Next()
		if !_xgo_ok {
			break
		}
		fmt.Println(line)
	}
}
