package main

import "fmt"

type RepeatMode string

const (
	_None_1 RepeatMode = ""
	Once    RepeatMode = "Once"
	Daily   RepeatMode = "Daily"
	Weekly  RepeatMode = "Weekly"
	Monthly RepeatMode = "Monthly"
	Yearly  RepeatMode = "Yearly"
)

type MonthlyRepeatOnDay string

const (
	_None_2    MonthlyRepeatOnDay = ""
	Sunday     MonthlyRepeatOnDay = "Sunday"
	Monday     MonthlyRepeatOnDay = "Monday"
	Tuesday    MonthlyRepeatOnDay = "Tuesday"
	Wednesday  MonthlyRepeatOnDay = "Wednesday"
	Thursday   MonthlyRepeatOnDay = "Thursday"
	Friday     MonthlyRepeatOnDay = "Friday"
	Saturday   MonthlyRepeatOnDay = "Saturday"
	Day        MonthlyRepeatOnDay = "day"
	Weekday    MonthlyRepeatOnDay = "weekday"
	WeekendDay MonthlyRepeatOnDay = "weekend day"
)
const None = ""

type Third string

const _None_3 Third = ""

var r RepeatMode = None
var m MonthlyRepeatOnDay = None
var t Third = None

func main() {
	fmt.Println(r, m, t)
}
