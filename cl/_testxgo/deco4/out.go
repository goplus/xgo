package main

func timeIt(fn func()) {
	fn()
}
func _xgodeco_compute(x int) (result int) {
	result = x * 2
	return
}
func compute(x int) (result int) {
	timeIt(func() {
		result = _xgodeco_compute(x)
	})
	return
}
func main() {
	compute(21)
}
