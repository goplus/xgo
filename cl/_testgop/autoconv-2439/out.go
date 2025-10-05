package main

type BasePtr struct {
}
type Base struct {
	*BasePtr
}
type T struct {
	Base
}

func Walk(p *Base) {
}
func WalkPtr(p *BasePtr) {
}
func f() *T {
	return &T{}
}
func main() {
	Walk(&new(T).Base)
	WalkPtr(f().BasePtr)
}
