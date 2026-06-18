Function/Method/Operator/Type Overloading
=====

### Overload Funcs

Define `overload func` in `inline func literal` style (see [overloadfunc1/add.xgo](../demo/fullspec/overloadfunc1/add.xgo)):

```go
func add = (
	func(a, b int) int {
		return a + b
	}
	func(a, b string) string {
		return a + b
	}
)

echo add(100, 7)
echo add("Hello", "World")
```

Define `overload func` in `ident` style (see [overloadfunc2/mul.xgo](../demo/fullspec/overloadfunc2/mul.xgo)):

```go
func mulInt(a, b int) int {
	return a * b
}

func mulFloat(a, b float64) float64 {
	return a * b
}

func mul = (
	mulInt
	mulFloat
)

echo mul(100, 7)
echo mul(1.2, 3.14)
```

### Overload Methods

Define `overload method` (see [overloadmethod/method.xgo](../demo/fullspec/overloadmethod/method.xgo)):

```go
type foo struct {
}

func (a *foo) mulInt(b int) *foo {
	echo "mulInt"
	return a
}

func (a *foo) mulFoo(b *foo) *foo {
	echo "mulFoo"
	return a
}

func (foo).mul = (
	(foo).mulInt
	(foo).mulFoo
)

var a, b *foo
var c = a.mul(100)
var d = a.mul(c)
```

### Overload Unary Operators

Define `overload unary operator` (see [overloadop1/overloadop.xgo](../demo/fullspec/overloadop1/overloadop.xgo)):

```go
type foo struct {
}

func -(a foo) (ret foo) {
	echo "-a"
	return
}

func ++(a foo) {
	echo "a++"
}

var a foo
var b = -a
a++
```

### Overload Binary Operators

Define `overload binary operator` (see [overloadop1/overloadop.xgo](../demo/fullspec/overloadop1/overloadop.xgo)):

```go
type foo struct {
}

func (a foo) * (b foo) (ret foo) {
	echo "a * b"
	return
}

func (a foo) != (b foo) bool {
	echo "a != b"
	return true
}

var a, b foo
var c = a * b
var d = a != b
```

However, `binary operator` usually need to support interoperability between multiple types. In this case it becomes more complex (see [overloadop2/overloadop.xgo](../demo/fullspec/overloadop2/overloadop.xgo)):

```go
type foo struct {
}

func (a foo) mulInt(b int) (ret foo) {
	echo "a * int"
	return
}

func (a foo) mulFoo(b foo) (ret foo) {
	echo "a * b"
	return
}

func intMulFoo(a int, b foo) (ret foo) {
	echo "int * b"
	return
}

func (foo).* = (
	(foo).mulInt
	(foo).mulFoo
	intMulFoo
)

var a, b foo
var c = a * 10
var d = a * b
var e = 10 * a
```

### Overload Types

TODO

### Overload Typecast

TODO
