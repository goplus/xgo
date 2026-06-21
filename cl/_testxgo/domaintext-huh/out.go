package main

import "github.com/goplus/xgo/cl/internal/huh"

func main() {
	form := huh.New("<form id=\"test\">\n</form>\n", "1", 2)
	form.Run()
}
