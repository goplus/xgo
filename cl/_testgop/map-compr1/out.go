package main

func main() {
	y := func() (_xgo_ret map[string]int) {
		_xgo_ret = map[string]int{}
		for i, x := range []string{"1", "3", "5", "7", "11"} {
			_xgo_ret[x] = i
		}
		return
	}()
}
