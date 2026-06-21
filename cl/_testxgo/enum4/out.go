package main

import "fmt"

type Weekday byte

const (
	Sunday Weekday = Weekday(byte(iota))
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

var v Weekday

func main() {
	v = Friday
	fmt.Println(v)
}
