package main

import (
	"fmt"
	"github.com/goplus/xgo/encoding/html"
	"github.com/qiniu/x/errors"
)

func main() {
	doc := func() (_xgo_ret *html.Object) {
		var _xgo_err error
		_xgo_ret, _xgo_err = html.New(`<html><body>
<p>Links:</p>
<ul>
	<li><a href="foo">Foo</a>
	<li><a href="/bar/baz">BarBaz</a>
</ul>
</body></html>
`)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "html`<html><body>\n<p>Links:</p>\n<ul>\n\t<li><a href=\"foo\">Foo</a>\n\t<li><a href=\"/bar/baz\">BarBaz</a>\n</ul>\n</body></html>\n`", "cl/_testgop/dql5/in.xgo", 1, "main.main")
			panic(_xgo_err)
		}
		return
	}()
	for a := range doc.XGo_Elem("body").XGo_Any("a").Dump().XGo_Enum() {
		fmt.Println(a.XGo_Attr__0("href"), a.Text__0())
	}
}
