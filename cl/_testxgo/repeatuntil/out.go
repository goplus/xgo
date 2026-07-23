package main

import "fmt"

func RepeatUntil(__xgo_autoclosure_cond func() bool, body func()) {
	for !__xgo_autoclosure_cond() {
		body()
	}
}
func main() {
	x := 0
	RepeatUntil(func() bool {
		return x >= 3
	}, func() {
		fmt.Println(x)
		x++
	})
	RepeatUntil(func() bool {
		return false
	}, func() {
	})
}
