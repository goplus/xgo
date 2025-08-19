package main

import (
	"fmt"
	"github.com/qiniu/x/xgo"
)

func main() {
	fmt.Println(func() (_xgo_ret []int) {
		for _xgo_it := xgo.NewRange__0(0, 3, 1).Gop_Enum(); ; {
			var _xgo_ok bool
			x, _xgo_ok := _xgo_it.Next()
			if !_xgo_ok {
				break
			}
			_xgo_ret = append(_xgo_ret, x)
		}
		return
	}())
}
