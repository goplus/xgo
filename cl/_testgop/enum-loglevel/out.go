package main

import "fmt"

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func main() {
	fmt.Println(Debug, Info, Warn, Error)
}
