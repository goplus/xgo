package main

type M = map[string]any

type basetype interface {
	string | int | bool | float64
}

type Var__0[T basetype] struct {
	Val T
}

type Var__1[T map[string]any] struct {
	Val T
}

func XGox_Var_Cast__0[T basetype]() *Var__0[T] {
	return new(Var__0[T])
}

func XGox_Var_Cast__1[T map[string]any]() *Var__1[T] {
	return new(Var__1[T])
}
