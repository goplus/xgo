package main

import (
	"fmt"
	html1 "github.com/goplus/xgo/dql/html"
	"github.com/goplus/xgo/encoding/html"
	"github.com/qiniu/x/errors"
)

func main() {
	doc := func() (_xgo_ret html.Object) {
		var _xgo_err error
		_xgo_ret, _xgo_err = html.New(`<html><body>
<p>Links:</p>
<ul>
	<li class="abc"><a href="foo">Foo</a>
	<li><a href="/bar/baz">BarBaz</a>
</ul>
</body></html>
`)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "html`<html><body>\n<p>Links:</p>\n<ul>\n\t<li class=\"abc\"><a href=\"foo\">Foo</a>\n\t<li><a href=\"/bar/baz\">BarBaz</a>\n</ul>\n</body></html>\n`", "cl/_testgop/dql5/in.xgo", 1, "main.main")
			panic(_xgo_err)
		}
		return
	}()
	fmt.Println(html1.NodeSet_Cast(func(_xgo_yield func(*html1.Node) bool) {
		doc.XGo_Any("li").XGo_Enum()(func(self html1.NodeSet) bool {
			if self.IsClass("abc") {
				if _xgo_val, _xgo_err := self.XGo_first(); _xgo_err == nil {
					if !_xgo_yield(_xgo_val) {
						return false
					}
				}
			}
			return true
		})
	}).Text__0())
	for a := range doc.XGo_Elem("body").XGo_Any("a").Dump().XGo_Enum() {
		fmt.Println(a.XGo_Attr__0("href"), a.Text__0())
	}
}
