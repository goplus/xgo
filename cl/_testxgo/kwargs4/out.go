package main

import "github.com/goplus/xgo/cl/internal/testutil"

func PlaySound(path string, options *testutil.Options) {
}
func main() {
	PlaySound("1.mp3", &testutil.Options{Loop: true})
	PlaySound("2.mp3", &testutil.Options{Loop: false, Async: true})
}
