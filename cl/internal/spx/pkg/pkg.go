package pkg

const (
	XGoPackage = true
)

type Vector struct {
	X int
	Y int
}

func NewVector(x, y int) *Vector {
	return &Vector{x, y}
}

func (v *Vector) Add__0(x int, y int) {
	v.X += x
	v.Y += y
}

func (v *Vector) Add__1(o *Vector) {
	v.Add__0(o.X, o.Y)
}

func (v *Vector) Self() *Vector {
	return v
}
