package main

import "fmt"

func dump(a struct {
	X_0 int16
	X_1 float32
}, _ ...bool) {
	fmt.Println(a)
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
}
