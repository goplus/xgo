package foo

const XGoPackage = true

type Mesher interface {
	Name() string
}

type N struct {
}

func (m *N) OnKey__0(a string, fn func()) {
}

func (m *N) OnKey__1(a string, fn func(key string)) {
}

func (m *N) OnKey__2(a []string, fn func()) {
}

func (m *N) OnKey__3(a []string, fn func(key string)) {
}

func (m *N) OnKey__4(a []Mesher, fn func()) {
}

func (m *N) OnKey__5(a []Mesher, fn func(key Mesher)) {
}

func (m *N) OnKey__6(a []string, b []string, fn func(key string)) {
}

func (m *N) OnKey__7(a []string, b []Mesher, fn func(key string)) {
}

func (m *N) OnKey__8(x int, y int) {
}

func OnKey__0(a string, fn func()) {
}

func OnKey__1(a string, fn func(key string)) {
}

func OnKey__2(a []string, fn func()) {
}

func OnKey__3(a []string, fn func(key string)) {
}

func OnKey__4(a []Mesher, fn func()) {
}

func OnKey__5(a []Mesher, fn func(key Mesher)) {
}

func OnKey__6(a []string, b []string, fn func(key string)) {
}

func OnKey__7(a []string, b []Mesher, fn func(key string)) {
}

func OnKey__8(x int, y int) {
}

func Test__0() {
}

func Test__1[N any](n N) {
}

func Test__2[N1, N2 any](n1 N1, n2 N2) {
}
