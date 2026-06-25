package main

func log(fn func() error) {
	fn()
}
func retry(times int, fn func() error) {
	for i := 0; i < times; i++ {
		if fn() == nil {
			return
		}
	}
}
func _xgodeco_fetch(urls ...string) (results []string, err error) {
	results = urls
	return
}
func fetch(urls ...string) (results []string, err error) {
	log(func() error {
		retry(3, func() error {
			results, err = _xgodeco_fetch(urls...)
			return err
		})
		return err
	})
	return
}
