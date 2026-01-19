package bar

const GopPackage = true

type M = map[string]any

type basetype interface {
	string | int | bool | float64
}

type Var__0[T basetype] struct {
	val T
}

func (p *Var__0[T]) Value() T {
	return p.val
}

type Var__1[T map[string]any] struct {
	val T
}

func (p *Var__1[T]) Value() T {
	return p.val
}

func XGox_Var_Cast__0[T basetype]() *Var__0[T] {
	return new(Var__0[T])
}

func XGox_Var_Cast__1[T map[string]any]() *Var__1[T] {
	return new(Var__1[T])
}

type Player struct {
}

func XGot_Player_XGox_OnCmd__0[T any](p *Player, handler func(cmd T) error) {
	var t T
	handler(t)
}

func XGot_Player_XGox_OnCmd__1[T1 ~int, T2 any](p *Player, n T1, handler func(n T1, cmd T2) error) {
	var t T2
	handler(n, t)
}
