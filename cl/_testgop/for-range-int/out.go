package main

import "fmt"

func main() {
	for i := range 10 {
		fmt.Println(i)
	}
	for range 5 {
		fmt.Println("Hi")
	}
}
