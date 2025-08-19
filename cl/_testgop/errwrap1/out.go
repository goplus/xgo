package main

import "github.com/qiniu/x/errors"

func F() (a int8, b int16, err error) {
	a = 1
	return
}
func F2() (err error) {
	c, d := func() (_xgo_ret int8, _xgo_ret2 int16) {
		var _xgo_err error
		_xgo_ret, _xgo_ret2, _xgo_err = F()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "F()", "cl/_testgop/errwrap1/in.xgo", 7, "main.F2")
			panic(_xgo_err)
		}
		return
	}()
	_ = c
	_ = d
	return
}
func main() {
	F2()
}
