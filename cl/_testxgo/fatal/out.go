package main

import (
	"github.com/qiniu/x/osx"
	"github.com/qiniu/x/stringutil"
	"os"
)

func main() {
	f, err := os.Open("hello.txt")
	if err != nil {
		osx.Errorln("[WARN] an error")
		osx.Fatal(stringutil.Concat("open file failed: ", err.Error()))
	}
	f.Close()
}
