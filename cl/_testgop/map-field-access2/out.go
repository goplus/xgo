package main

import "fmt"

var v interface{}

func main() {
	_autoGo_1, _ := v.(map[string]any)
	_autoGo_2, _ := _autoGo_1["b"].(map[string]any)
	c, ok := _autoGo_2["c"].(int)
	fmt.Println(c, ok)
	_autoGo_3, _ := v.(map[string]any)
	_autoGo_4, _ := _autoGo_3["b"].(map[string]any)
	d, ok := _autoGo_4["c"]
	fmt.Println(d, ok)
}
