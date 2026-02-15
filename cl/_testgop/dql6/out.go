package main

import (
	"fmt"
	"github.com/goplus/xgo/dql/golang"
	"github.com/goplus/xgo/dql/reflects"
	golang1 "github.com/goplus/xgo/encoding/golang"
	"github.com/qiniu/x/errors"
)

func main() {
	doc := func() (_xgo_ret *golang.File) {
		var _xgo_err error
		_xgo_ret, _xgo_err = golang1.New(`package main

var (
    a, b string
    c    int
)

func add(a, b int) int {
    return a + b
}

func mul(a, b float64) float64 {
    return a * b
}
`)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "golang`package main\n\nvar (\n    a, b string\n    c    int\n)\n\nfunc add(a, b int) int {\n    return a + b\n}\n\nfunc mul(a, b float64) float64 {\n    return a * b\n}\n`", "cl/_testgop/dql6/in.xgo", 1, "main.main")
			panic(_xgo_err)
		}
		return
	}()
	for fn := range golang.NodeSet_Cast(func(_xgo_yield func(reflects.Node) bool) {
		doc.XGo_Elem("decls").XGo_Child().XGo_Enum()(func(self golang.NodeSet) bool {
			if self.Class() == "FuncDecl" {
				if _xgo_val, _xgo_err := self.XGo_first(); _xgo_err == nil {
					if !_xgo_yield(_xgo_val) {
						return false
					}
				}
			}
			return true
		})
	}).XGo_Enum() {
		fmt.Println(fn.XGo_Attr__0("name"))
	}
}
