package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/dql"
)

func main() {
	doc := dql.New()
	name := func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = dql.NodeSet_Cast(func(_xgo_yield func(*dql.Node) bool) {
			doc.XGo_Elem("users").XGo_Enum()(func(self dql.NodeSet) bool {
				if func() (_xgo_ret int) {
					var _xgo_err error
					_xgo_ret, _xgo_err = self.XGo_Attr__1("age")
					if _xgo_err != nil {
						return 100
					}
					return
				}() < 18 {
					if _xgo_val, _xgo_err := self.XGo_first(); _xgo_err == nil {
						if !_xgo_yield(_xgo_val) {
							return false
						}
					}
				}
				return true
			})
		}).XGo_Attr__1("name")
		if _xgo_err != nil {
			return 0
		}
		return
	}()
	fmt.Println(name)
}
