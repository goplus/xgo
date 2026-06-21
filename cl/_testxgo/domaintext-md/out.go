package main

import (
	"github.com/xushiwei/markdown"
	"os"
)

func main() {
	md := markdown.New(`
# Title

Hello world
`)
	md.Convert(os.Stdout)
}
