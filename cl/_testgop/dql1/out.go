package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/dql"
)

func main() {
	doc := dql.New()
	fmt.Println(doc.XGo_Elem("foo").XGo_Any("users").XGo_Child().XGo_Attr__0("age"))
	fmt.Println(doc.XGo_Elem("foo-name").XGo_Any("elem-name").XGo_Child().XGo_Attr__0("attr-name"))
	fmt.Println(doc.XGo_Any("").XGo_Attr__0("name"))
}
