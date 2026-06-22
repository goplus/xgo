package main

import "fmt"

type RepeatMode string

const (
	None               = ""
	Once    RepeatMode = "Once"
	Daily   RepeatMode = "Daily"
	Weekly  RepeatMode = "Weekly"
	Monthly RepeatMode = "Monthly"
	Yearly  RepeatMode = "Yearly"
)

type MonthlyRepeatOnDay string

var r RepeatMode = None
var m MonthlyRepeatOnDay = None

func main() {
	fmt.Println(r, m)
}
