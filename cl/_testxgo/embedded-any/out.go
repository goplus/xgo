package main

type Payload struct {
	any `json:"value"`
}

func unwrap(p Payload) interface{} {
	return p.any
}
func shadow() int {
	type any struct {
		Value int
	}
	type Holder struct {
		any
	}
	return Holder{any{3}}.Value
}

var _ = Payload{any: "text"}
