package main

type foo int

const XGos_foo_Name = "xgo"

func Name() string {
	return XGos_foo_Name
}
func Inc() {
	XGos_foo_Count++
}

var XGos_foo_Count int = 100
