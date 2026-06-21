package main

func main() {
	a := []float64{1, 3.4, 5}
	b := func() (_xgo_ret []float64) {
		for _, x := range a {
			_xgo_ret = append(_xgo_ret, x*x)
		}
		return
	}()
}
