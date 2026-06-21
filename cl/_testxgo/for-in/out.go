package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/dql"
)

func main() {
	doc := dql.New()
	for user := range doc.XGo_Elem("users").XGo_Enum() {
		fmt.Println(user)
	}
}
