package main

import "fmt"

func foo(yield func() bool) {
	yield()
}
func bar(yield func(v string) bool) {
	yield("World")
}
func weekdays(yield func(k string, v int) bool) {
	yield("Mon", 1)
}
func main() {
	for range foo {
		fmt.Println("Hi")
	}
	for v := range bar {
		fmt.Println(v)
	}
	for k, v := range weekdays {
		fmt.Println(k, v)
	}
}
