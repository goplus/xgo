package main

func wrap(fn func()) {
	fn()
}
func retry(times int, fn func() error) {
	for i := 0; i < times; i++ {
		if fn() == nil {
			return
		}
	}
}
func _xgodeco_process() {
}
func process() {
	wrap(func() {
		_xgodeco_process()
		return
	})
	return
}
func _xgodeco_fetchData(url string) (data string, err error) {
	data = url
	return
}
func fetchData(url string) (data string, err error) {
	retry(3, func() error {
		data, err = _xgodeco_fetchData(url)
		return err
	})
	return
}
