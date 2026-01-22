package main

import "fmt"

type T struct {
	x struct {
		X_0 int16
		X_1 float32
	}
}

func dump(a struct {
	X_0 int16
	X_1 float32
}, _ ...bool) {
	t := T{x: struct {
		X_0 int16
		X_1 float32
	}{1, 3.14}}
	fmt.Println(a, t)
}
func demo(a int16, b float32) {
	fmt.Println(a, b)
}
func main() {
	ken := struct {
		X_0 string
		X_1 string
		X_2 int
	}{"Ken", "ken@abc.com", 7}
	fmt.Println(ken)
	dump(struct {
		X_0 int16
		X_1 float32
	}{1, 3.14}, true)
	dump(struct {
		X_0 int16
		X_1 float32
	}{1, 3.14})
	demo(1, 3.14)
}
