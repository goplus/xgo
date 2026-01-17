package foo

const XGoPackage = true

const XGoo__T__XGo_Add = ".AddInt,AddString"

type T int

// AddInt doc
func (p T) AddInt(b int) *T {
	return nil
}

// AddString doc
func AddString(this *T, b string) *T {
	return nil
}
