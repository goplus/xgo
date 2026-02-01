package main

func main() {
	z := func() (_xgo_ret map[int]string) {
		_xgo_ret = map[int]string{}
		for k, v := range map[string]int{"Hello": 1, "Hi": 3, "xsw": 5, "XGo": 7} {
			if v > 3 {
				_xgo_ret[v] = k
			}
		}
		return
	}()
}
