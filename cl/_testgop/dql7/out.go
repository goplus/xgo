package main

import (
	"github.com/goplus/xgo/dql/maps"
	"github.com/goplus/xgo/encoding/json"
	"github.com/qiniu/x/errors"
)

func main() {
	doc := func() (_xgo_ret json.Object) {
		var _xgo_err error
		_xgo_ret, _xgo_err = json.New(`{
	"animals": [
		{"class": "gopher", "at": "Line 1"},
		{"class": "armadillo", "at": "Line 2"},
		{"class": "zebra", "at": Line 3"},
		{"class": "unknown", "at": Line 4"},
		{"class": "gopher", "at": Line 5"},
		{"class": "bee", "at": Line 6"},
		{"class": "gopher", "at": Line 7"},
		{"class": "zebra", "at": Line 8"}
	]
}
`)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "json`{\n\t\"animals\": [\n\t\t{\"class\": \"gopher\", \"at\": \"Line 1\"},\n\t\t{\"class\": \"armadillo\", \"at\": \"Line 2\"},\n\t\t{\"class\": \"zebra\", \"at\": Line 3\"},\n\t\t{\"class\": \"unknown\", \"at\": Line 4\"},\n\t\t{\"class\": \"gopher\", \"at\": Line 5\"},\n\t\t{\"class\": \"bee\", \"at\": Line 6\"},\n\t\t{\"class\": \"gopher\", \"at\": Line 7\"},\n\t\t{\"class\": \"zebra\", \"at\": Line 8\"}\n\t]\n}\n`", "cl/_testgop/dql7/in.xgo", 1, "main.main")
			panic(_xgo_err)
		}
		return
	}()
	for animal := range maps.NodeSet_Cast(func(_xgo_yield func(maps.Node) bool) {
		maps.New(_autoGo_1["animals"]).XGo_Child().XGo_Enum()(func(self maps.NodeSet) bool {
			if self.XGo_Attr__0("class") == "zebra" {
				if _xgo_val, _xgo_err := self.XGo_first(); _xgo_err == nil {
					if !_xgo_yield(_xgo_val) {
						return false
					}
				}
			}
			return true
		})
	}).XGo_Enum() {
		_autoGo_1, _ := doc.(map[string]any)
		animal.XGo_Attr__0("at")
	}
}
