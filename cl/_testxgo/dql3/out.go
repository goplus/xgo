package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/dql"
)

func main() {
	doc := dql.New()
	fmt.Println(doc.XGo_Child().XGo_Select("users").XGo_Select("users").XGo_Attr__0("name"))
}
