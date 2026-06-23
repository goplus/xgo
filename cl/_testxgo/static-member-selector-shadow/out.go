package main

type foo struct {
	name  string
	count int
}

const XGos_foo_Name = "static"

var XGos_foo_Count int = 100

func main() {
	foo := foo{name: "local"}
	a := foo.name
	foo.count++
}
