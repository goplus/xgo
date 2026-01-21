package main

import "fmt"

func main() {
	ken := struct {
		X_0 string
		X_1 string
		X_2 int
	}{"Ken", "ken@abc.com", 7}
	fmt.Println(ken)
}
