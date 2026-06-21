package main

func foo() (int, bool) {
	return func() (_xgo_ret int, _xgo_ok bool) {
		for i, x := range []string{"1", "3", "5", "7", "11"} {
			if x == "5" {
				return i, true
			}
		}
		return
	}()
}
