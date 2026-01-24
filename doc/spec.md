The XGo Full Specification
=====

TODO

## Comments

See [Comments](spec-mini.md#comments).


## Literals

See [Literals](spec-mini.md#literals).

### String literals

#### C style string literals

TODO

```go
c"Hello, world!\n"
```

#### Python string literals

TODO

```go
py"Hello, world!\n"
```


## Types

### Boolean types

See [Boolean types](spec-mini.md#boolean-types).

### Numeric types

See [Numeric types](spec-mini.md#numeric-types).

### String types

See [String types](spec-mini.md#string-types).

#### C style string types

```go
import "c"

*c.Char  // alias for *int8
```

#### Python string types

```go
import "py"

*py.Object  // TODO: *py.String?
```

### Array types

See [Array types](spec-mini.md#array-types).

An array type T may not have an element of type T, or of a type containing T as a component, directly or indirectly, if those containing types are only array or struct types.

```go
// invalid array types
type (
	T1 [10]T1                 // element type of T1 is T1
	T2 [10]struct{ f T2 }     // T2 contains T2 as component of a struct
	T3 [10]T4                 // T3 contains T3 as component of a struct in T4
	T4 struct{ f T3 }         // T4 contains T4 as component of array T3 in a struct
)

// valid array types
type (
	T5 [10]*T5                // T5 contains T5 as component of a pointer
	T6 [10]func() T6          // T6 contains T6 as component of a function type
	T7 [10]struct{ f []T7 }   // T7 contains T7 as component of a slice in a struct
)
```

### Pointer types

See [Pointer types](spec-mini.md#pointer-types).

### Slice types

See [Slice types](spec-mini.md#slice-types).

### Map types

See [Map types](spec-mini.md#map-types).

### Struct types

A _struct_ is a sequence of named elements, called fields, each of which has a name and a type. Field names may be specified explicitly (IdentifierList) or implicitly (EmbeddedField). Within a struct, non-[blank]() field names must be [unique]().

```go
// An empty struct.
struct {}

// A struct with 6 fields.
struct {
	x, y int
	u float32
	_ float32  // padding
	A *[]int
	F func()
}
```

A field declared with a type but no explicit field name is called an _embedded field_. An embedded field must be specified as a type name T or as a pointer to a non-interface type name *T, and T itself may not be a pointer type. The unqualified type name acts as the field name.

```go
// A struct with four embedded fields of types T1, *T2, P.T3 and *P.T4
struct {
	T1        // field name is T1
	*T2       // field name is T2
	P.T3      // field name is T3
	*P.T4     // field name is T4
	x, y int  // field names are x and y
}
```

The following declaration is illegal because field names must be unique in a struct type:

```go
struct {
	T     // conflicts with embedded field *T and *P.T
	*T    // conflicts with embedded field T and *P.T
	*P.T  // conflicts with embedded field T and *T
}
```

A field or [method]() f of an embedded field in a struct x is called _promoted_ if x.f is a legal [selector]() that denotes that field or method f.

Promoted fields act like ordinary fields of a struct except that they cannot be used as field names in [composite literals]() of the struct.

Given a struct type S and a [named type]() T, promoted methods are included in the method set of the struct as follows:

* If S contains an embedded field T, the [method sets]() of S and *S both include promoted methods with receiver T. The method set of *S also includes promoted methods with receiver *T.
* If S contains an embedded field *T, the method sets of S and *S both include promoted methods with receiver T or *T.

A field declaration may be followed by an optional string literal tag, which becomes an attribute for all the fields in the corresponding field declaration. An empty tag string is equivalent to an absent tag. The tags are made visible through a [reflection interface]() and take part in [type identity]() for structs but are otherwise ignored.

```go
struct {
	x, y float64 ""  // an empty tag string is like an absent tag
	name string  "any string is permitted as a tag"
	_    [4]byte "ceci n'est pas un champ de structure"
}

// A struct corresponding to a TimeStamp protocol buffer.
// The tag strings define the protocol buffer field numbers;
// they follow the convention outlined by the reflect package.
struct {
	microsec  uint64 `protobuf:"1"`
	serverIP6 uint64 `protobuf:"2"`
}
```

A struct type T may not contain a field of type T, or of a type containing T as a component, directly or indirectly, if those containing types are only array or struct types.

```go
// invalid struct types
type (
	T1 struct{ T1 }            // T1 contains a field of T1
	T2 struct{ f [10]T2 }      // T2 contains T2 as component of an array
	T3 struct{ T4 }            // T3 contains T3 as component of an array in struct T4
	T4 struct{ f [10]T3 }      // T4 contains T4 as component of struct T3 in an array
)

// valid struct types
type (
	T5 struct{ f *T5 }         // T5 contains T5 as component of a pointer
	T6 struct{ f func() T6 }   // T6 contains T6 as component of a function type
	T7 struct{ f [10][]T7 }    // T7 contains T7 as component of a slice in an array
)
```

### Tuple types

See [Tuple types](spec-mini.md#tuple-types).

Tuple types in the FullSpec support all MiniSpec features plus additional advanced capabilities for compatibility with Go and migration scenarios.

#### Brace-Style Construction (FullSpec Only)

In addition to function-style construction, the FullSpec supports brace-based initialization using `:` for field assignment:

```go
type Point (x int, y int)

// Function-style (available in both MiniSpec and FullSpec)
p1 := Point(10, 20)
p1 := Point(x = 10, y = 20)

// Brace-style (FullSpec only)
p2 := Point{x: 10, y: 20}
p2 := Point{10, 20}
```

#### Anonymous Tuple Literals with Braces

The FullSpec allows using tuple literals within brace-based composite literals:

```go
// Using tuples in struct fields
type Record struct {
	coords (int, int)
	data   (string, bool)
}

r := Record{
	coords: (10, 20),
	data:   ("test", true),
}
```

#### Type Compatibility and Reflection

At runtime, tuples are implemented as structs with ordinal field names `_0`, `_1`, `_2`, etc.:

```go
type Point (x int, y int)

// At runtime, Point is equivalent to:
// struct {
//     _0 int  // accessible as .x at compile time, ._0 or .0 at runtime
//     _1 int  // accessible as .y at compile time, ._1 or .1 at runtime
// }
```

Tuple types with the same element types (in the same order) have identical underlying structures but are distinct named types:

```go
type Point (x int, y int)
type Coord (a int, b int)

// Point and Coord have identical underlying types but are different types
// Conversion is required: c := Coord(p)
```

### Function types

See [Function types](spec-mini.md#function-types).

### Interface types

TODO

#### Builtin interfaces

See [Builtin interfaces](spec-mini.md#builtin-interfaces).

##### The comparable interface

The predeclared interface type `comparable` denotes the set of all non-interface types that are strictly comparable. A type is strictly comparable if values of that type can be compared using the `==` and `!=` operators.

The `comparable` interface is primarily used as a type constraint in generic code and cannot be used as the type of a variable or struct field:

```go
// Example: using comparable as a type constraint
func Find[T comparable](slice []T, value T) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}
```

Types that are strictly comparable include:
- Boolean, numeric, and string types
- Pointer types
- Channel types
- Array types (if their element type is strictly comparable)
- Struct types (if all their field types are strictly comparable)

Slice, map, and function types are not comparable and cannot be used with `comparable`.

### Channel types

TODO


## Expressions

### Commands and calls

See [Commands and calls](spec-mini.md#commands-and-calls).

### Operators

See [Operators](spec-mini.md#operators).

#### Operator precedence

See [Operator precedence](spec-mini.md#operator-precedence).

#### Arithmetic operators

See [Arithmetic operators](spec-mini.md#arithmetic-operators).

#### Comparison operators

See [Comparison operators](spec-mini.md#comparison-operators).

The equality operators == and != apply to operands of comparable types. The ordering operators <, <=, >, and >= apply to operands of ordered types. These terms and the result of the comparisons are defined as follows:

* Channel types are comparable. Two channel values are equal if they were created by the same call to [make]() or if both have value `nil`.
* Struct types are comparable if all their field types are comparable. Two struct values are equal if their corresponding non-[blank]() field values are equal. The fields are compared in source order, and comparison stops as soon as two field values differ (or all fields have been compared).
* Type parameters are comparable if they are strictly comparable (see below).

```go
const c = 3 < 4            // c is the untyped boolean constant true

type MyBool bool
var x, y int
var (
	// The result of a comparison is an untyped boolean.
	// The usual assignment rules apply.
	b3        = x == y // b3 has type bool
	b4 bool   = x == y // b4 has type bool
	b5 MyBool = x == y // b5 has type MyBool
)
```

A type is _strictly comparable_ if it is comparable and not an interface type nor composed of interface types. Specifically:

* Boolean, numeric, string, pointer, and channel types are strictly comparable.
* Struct types are strictly comparable if all their field types are strictly comparable.
* Array types are strictly comparable if their array element types are strictly comparable.
* Type parameters are strictly comparable if all types in their type set are strictly comparable.

#### Logical operators

See [Logical operators](spec-mini.md#logical-operators).

### Address operators

See [Address operators](spec-mini.md#address-operators).

### Send/Receive operator

TODO

### Conversions

See [Conversions](spec-mini.md#conversions).

TODO


## Statements

TODO

## Built-in functions

### Appending to and copying slices

See [Appending to and copying slices](spec-mini.md#appending-to-and-copying-slices).

### Clear

TODO

### Close

TODO

### Manipulating complex numbers
