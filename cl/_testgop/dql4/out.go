package main

import (
	"fmt"
	"github.com/goplus/xgo/dql/xml"
	xml1 "github.com/goplus/xgo/encoding/xml"
	"github.com/qiniu/x/errors"
)

func main() {
	doc := func() (_xgo_ret *xml.Node) {
		var _xgo_err error
		_xgo_ret, _xgo_err = xml1.New(`<doc><animals>
  <animal class="gopher">Line 1</animal>
  <animal class="armadillo">Line 2</animal>
  <animal class="zebra">Line 3</animal>
  <animal class="unknown">Line 4</animal>
  <animal class="gopher">Line 5</animal>
  <animal class="bee">Line 6</animal>
  <animal class="gopher">Line 7</animal>
  <animal class="zebra">Line 8</animal>
</animals></doc>
`)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "xml`<doc><animals>\n  <animal class=\"gopher\">Line 1</animal>\n  <animal class=\"armadillo\">Line 2</animal>\n  <animal class=\"zebra\">Line 3</animal>\n  <animal class=\"unknown\">Line 4</animal>\n  <animal class=\"gopher\">Line 5</animal>\n  <animal class=\"bee\">Line 6</animal>\n  <animal class=\"gopher\">Line 7</animal>\n  <animal class=\"zebra\">Line 8</animal>\n</animals></doc>\n`", "cl/_testgop/dql4/in.xgo", 1, "main.main")
			panic(_xgo_err)
		}
		return
	}()
	fmt.Println(xml.NodeSet_Cast(func(_xgo_yield func(*xml.Node) bool) {
		doc.XGo_Elem("animals").XGo_Child().XGo_Enum()(func(self xml.NodeSet) bool {
			if self.XGo_Attr__0("class") == "zebra" {
				if _xgo_val, _xgo_err := self.XGo_first(); _xgo_err == nil {
					if !_xgo_yield(_xgo_val) {
						return false
					}
				}
			}
			return true
		})
	}).XGo_text__0())
}
