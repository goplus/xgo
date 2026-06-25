package main

func log(fn func()) {
	fn()
}
func retry(times int, fn func() error) {
	for i := 0; i < times; i++ {
		if fn() == nil {
			return
		}
	}
}
func _xgodeco_fetch(url string) (result string, err error) {
	result = url
	return
}
func fetch(url string) (result string, err error) {
	log(func() {
		retry(3, func() error {
			result, err = _xgodeco_fetch(url)
			return err
		})
	})
	return
}
func main() {
	fetch("http://example.com")
}
