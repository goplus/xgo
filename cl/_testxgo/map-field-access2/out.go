package main

import "fmt"

var v interface{}

func main() {
	c, ok := _autoGo_1(_autoGo_1(v)["b"])["c"].(int)
	fmt.Println(c, ok)
	d, ok := _autoGo_1(_autoGo_1(v)["b"])["c"]
	fmt.Println(d, ok)
}
func _autoGo_1(value interface{}) map[string]interface{} {
	m, _ := value.(map[string]interface{})
	return m
}
