package main

import "github.com/goplus/xgo/cl/internal/overload/intf"

func f(sp intf.Sprite) {
	sp.GlideToXYpos(10, 20, 0.01)
}
