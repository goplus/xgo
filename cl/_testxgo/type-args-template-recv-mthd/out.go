package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/typeargs"
)

type MyApp struct {
	typeargs.App
}
type Start struct {
}

var app MyApp

func main() {
	typeargs.XGot_App_XGox_OnCall[Start, *MyApp](&app, func(args *Start) {
		fmt.Println("onCall")
	})
}
