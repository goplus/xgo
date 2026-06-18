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

---

## Under the Hood: How XGo Encodes Overloading in Go

XGo treats Go as its compilation target — the same relationship TypeScript has with JavaScript. Every XGo construct must be representable as valid, standard Go code. Overloading is one of the clearest windows into how this works.

### Overload funcs: the two encoding strategies

XGo uses two different Go encodings depending on how an overloaded function is defined.

**Inline literal style → `__N` numeric suffixes**

When variants are written as anonymous function literals inside the overload declaration, the compiler assigns each variant an index and emits a separate top-level Go function with a `__0`, `__1`, `__2`, … suffix:

```go
// XGo source
func add = (
    func(a, b int) int       { return a + b }
    func(a, b string) string { return a + b }
)
```

```go
// Generated Go
func add__0(a, b int) int {
    return a + b
}

func add__1(a, b string) string {
    return a + b
}
```

Each emitted function is an ordinary Go function. `go vet`, `go test`, and every static analysis tool can process them without any knowledge of XGo.

**Ident style → `XGoo_` constant**

When variants are written as references to already-named functions, the functions themselves are emitted unchanged and a string constant records the overload group membership:

```go
// XGo source
func mul = (
    mulInt
    mulFloat
)
```

```go
// Generated Go
func mulInt(a, b int) int            { return a * b }
func mulFloat(a, b float64) float64  { return a * b }

const XGoo_mul = "mulInt,mulFloat"
```

The `XGoo_` prefix is a protocol: any XGo-aware tool reading this package knows that `mul` is an overloaded symbol and that its variants are `mulInt` and `mulFloat`. A plain Go tool ignores the constant without errors.

### Overload methods: `.`-prefixed references

Method overloading uses the same two encoding strategies as functions. The key difference is in the `XGoo_` constant: method references are written with a leading `.` to distinguish them from global functions. A name like `.mulInt` means `foo.mulInt` (a method on the receiver type), while a bare name like `intMulFoo` means a package-level function.

```go
// XGo source
func (foo).mul = (
    (foo).mulInt
    (foo).mulFoo
)
```

```go
// Generated Go
func (a *foo) mulInt(b int) *foo  { ... }
func (a *foo) mulFoo(b *foo) *foo { ... }

const XGoo_foo_mul = ".mulInt,.mulFoo"
```

The constant name follows the pattern `XGoo_<TypeName>_<MethodName>`, scoping the overload declaration to its receiver type. The `.` prefix on each variant signals "this is a method on `foo`", not a global function.

> **Escaping rule:** if either `<TypeName>` or `<MethodName>` contains an underscore, the single-underscore separators are promoted to double underscores throughout the entire constant name, i.e. `XGoo__<TypeName>__<MethodName>`. This prevents ambiguity when parsing the name back into its components.

### Overload operators: `XGo_` names and the full operator table

Go does not have operator overloading, so XGo maps each operator symbol to a canonical `XGo_`-prefixed name. Because operator symbols are not legal Go identifiers, they cannot appear directly as function names; the `XGo_` name is the Go-legal stand-in.

**Binary operator name table**

| Operator | `XGo_` name       | Operator | `XGo_` name          |
|----------|-------------------|----------|----------------------|
| `+`      | `XGo_Add`         | `+=`     | `XGo_AddAssign`      |
| `-`      | `XGo_Sub`         | `-=`     | `XGo_SubAssign`      |
| `*`      | `XGo_Mul`         | `*=`     | `XGo_MulAssign`      |
| `/`      | `XGo_Quo`         | `/=`     | `XGo_QuoAssign`      |
| `%`      | `XGo_Rem`         | `%=`     | `XGo_RemAssign`      |
| `&`      | `XGo_And`         | `&=`     | `XGo_AndAssign`      |
| `\|`     | `XGo_Or`          | `\|=`    | `XGo_OrAssign`       |
| `^`      | `XGo_Xor`         | `^=`     | `XGo_XorAssign`      |
| `<<`     | `XGo_Lsh`         | `<<=`    | `XGo_LshAssign`      |
| `>>`     | `XGo_Rsh`         | `>>=`    | `XGo_RshAssign`      |
| `&^`     | `XGo_AndNot`      | `&^=`    | `XGo_AndNotAssign`   |
| `==`     | `XGo_EQ`          | `!=`     | `XGo_NE`             |
| `<`      | `XGo_LT`          | `<=`     | `XGo_LE`             |
| `>`      | `XGo_GT`          | `>=`     | `XGo_GE`             |
| `&&`     | `XGo_LAnd`        | `\|\|`   | `XGo_LOr`            |
| `<-`     | `XGo_Send`        | `->`     | `XGo_PointTo`        |
| `<>`     | `XGo_PointBi`     |          |                      |

**Unary operator name table**

| Operator | `XGo_` name  |
|----------|--------------|
| `++`     | `XGo_Inc`    |
| `--`     | `XGo_Dec`    |
| `-`      | `XGo_Neg`    |
| `+`      | `XGo_Dup`    |
| `^`      | `XGo_Not`    |
| `!`      | `XGo_LNot`   |
| `<-`     | `XGo_Recv`   |

**Simple operator overloading**

A single-type binary operator becomes a method named after the operator's `XGo_` name:

```go
// XGo source
func (a foo) * (b foo) (ret foo) { ... }
func (a foo) != (b foo) bool     { ... }
```

```go
// Generated Go
func (a foo) XGo_Mul(b foo) (ret foo) { ... }
func (a foo) XGo_NE(b foo) bool       { ... }
```

A unary operator similarly becomes a no-argument method:

```go
// XGo source
func -(a foo) (ret foo) { ... }
func ++(a foo)          { ... }
```

```go
// Generated Go
func (a foo) XGo_Neg() (ret foo) { ... }
func (a foo) XGo_Inc()           { ... }
```

**Multi-type binary operators**

When an operator must work across multiple type combinations (e.g. `foo * int`, `foo * foo`, `int * foo`), the ident-style overload encoding applies. The `XGoo_` constant name uses the operator's `XGo_` name as the method component, and each variant follows the same `.method` vs bare-function convention:

```go
// XGo source
func (foo).* = (
    (foo).mulInt
    (foo).mulFoo
    intMulFoo
)
```

```go
// Generated Go
func (a foo) mulInt(b int) (ret foo)  { ... }  // foo * int
func (a foo) mulFoo(b foo) (ret foo)  { ... }  // foo * foo
func intMulFoo(a int, b foo) (ret foo){ ... }  // int * foo

const XGoo__foo__XGo_Mul = ".mulInt,.mulFoo,intMulFoo"
```

Reading `XGoo__foo__XGo_Mul`: this declares the overloaded `*` operator (`XGo_Mul`) on type `foo`. Because `XGo_Mul` contains an underscore, the separators are doubled throughout. The variants `.mulInt` and `.mulFoo` are methods on `foo`; `intMulFoo` is a package-level function handling the `int * foo` case.

### Summary: the two encoding primitives

All of XGo's overloading — functions, methods, and operators — is built from exactly two primitives in the generated Go code:

| Primitive | When used | Example |
|---|---|---|
| `__N` numeric suffix | Inline literal variants | `add__0`, `add__1` |
| `XGoo_` string constant | Named variant groups (funcs, methods, operators) | `XGoo_mul`, `XGoo_foo_mul`, `XGoo__foo__XGo_Mul` |

In `XGoo_` constant values, `.name` denotes a method on the receiver type; a bare `name` denotes a package-level function.

Neither primitive requires changes to the Go compiler or runtime. XGo-aware tools (the compiler, `xgofmt`, language server) read the constants to reconstruct overload groups. Plain Go tools see ordinary functions and an ordinary string constant, and work without modification.
