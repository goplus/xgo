package main

import "fmt"

func RepeatUntil(cond func() bool, body func()) {
	for !cond() {
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
