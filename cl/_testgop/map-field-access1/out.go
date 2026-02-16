package main

import "fmt"

func main() {
	v := map[string]map[string]interface{}{"a": map[string]interface{}{"b": 1}}
	c, ok := v["b"]["c"].(int)
	fmt.Println(c, ok)
	c, ok = v["b"]["c"].(int)
	fmt.Println(c, ok)
}
