package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/dql"
)

func main() {
	doc := dql.New()
	fmt.Println(doc.XGo_Any().XGo_Node("users").XGo_Child().XGo_Attr("age").XGo_0())
}
