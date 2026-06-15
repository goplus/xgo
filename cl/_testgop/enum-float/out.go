package main

import "fmt"

type Scale float64

const (
	Tiny   Scale = 0.25
	Small  Scale = 0.5
	Normal Scale = 1.0
	Large  Scale = 2.0
)

func main() {
	fmt.Println(Tiny, Small, Normal, Large)
}
