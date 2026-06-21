package main

import "github.com/goplus/xgo/tpl"

func main() {
	cl, err := tpl.NewEx(`expr = INT % ("+" | "-")`, "cl/_testgop/domaintext-tpl/in.xgo", 1, 15)
	cl.ParseExpr("1+2", nil)
	_ = err
}
