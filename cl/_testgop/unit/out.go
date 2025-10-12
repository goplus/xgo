package main

import (
	"github.com/goplus/xgo/cl/internal/unit"
	"time"
)

func Wait(time.Duration) {
}
func Step(unit.Distance) {
}
func main() {
	Wait(500)
	Wait(60000000000)
	Step(1000)
}
