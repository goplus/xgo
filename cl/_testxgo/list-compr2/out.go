package main

import "fmt"

func main() {
	arr := []float64{1, 2, 3, 4.1, 5, 6}
	x := func() (_xgo_ret [][]float64) {
		for _, b := range arr {
			if b > 2 {
				for _, a := range arr {
					if a < b {
						_xgo_ret = append(_xgo_ret, []float64{a, b})
					}
				}
			}
		}
		return
	}()
	fmt.Println("x:", x)
}
