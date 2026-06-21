package main

import "github.com/qiniu/x/errors"

func F() (a int8, b int16, err error) {
	a = 1
	return
}
func F2() (err error) {
	var _autoGo_1 int8
	var _autoGo_2 int16
	{
		var _xgo_err error
		_autoGo_1, _autoGo_2, _xgo_err = F()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "F()", "cl/_testgop/errwrap2/in.xgo", 7, "main.F2")
			return _xgo_err
		}
		goto _autoGo_3
	_autoGo_3:
	}
	c, d := _autoGo_1, _autoGo_2
	_ = c
	_ = d
	return
}
func main() {
	F2()
}
