package main

import "fmt"

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

type Permission int

const (
	Read Permission = 1 << iota
	Write
	Execute
)

func main() {
	fmt.Println(Sunday, Monday, Saturday)
	fmt.Println(Read, Write, Execute)
}
