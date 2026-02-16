package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
)

const XGoo_Bar = "BarOne,BarTwo"

func BarOne() int {
	return 0
}
func BarTwo() (int, error) {
	return 0, nil
}
func main() {
	fmt.Println(func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = BarTwo()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "bar", "cl/_testgop/errwrap3/in.xgo", 14, "main.main")
			panic(_xgo_err)
		}
		return
	}())
}
