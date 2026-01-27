package main

import "fmt"

var v interface{}

func main() {
	c, ok := v.(map[string]any)["b"].(map[string]any)["c"].(int)
	fmt.Println(c, ok)
	d, ok := v.(map[string]any)["b"].(map[string]any)["c"]
	fmt.Println(d, ok)
}
