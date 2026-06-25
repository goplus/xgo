package main

func wrap(fn func()) {
	fn()
}
func _xgodeco_process() {
}
func process() {
	wrap(func() {
		_xgodeco_process()
	})
}
func main() {
	process()
}
