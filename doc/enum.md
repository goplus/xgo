# Enum Type

In Go, defining an enumeration requires two separate declarations: first a
named type with an explicit underlying type, and then a `const` block whose
values are typed as that named type:

```go
type Color int

const (
    ColorRed   Color = iota
    ColorGreen
    ColorBlue
)
```

This pattern is verbose for a very common need, and it forces the developer
to make an early, often arbitrary, decision about the underlying type
(`int`, `int8`, `string`, `bool`, etc.) before any values have even been
written down. In practice the underlying type is usually *derived* from the
values themselves — if all values are integers, the underlying type is some
integer type; if all values are strings, it's `string`; and so on.

XGo aims to reduce this boilerplate while remaining fully compatible with
Go's type system, AST, and toolchain at the semantic level.

## Proposal

Introduce a new declaration form:

```go
type XXX const (
    XXXEnum1 = expr1
    XXXEnum2 = expr2
    ...
)
```

This declares `XXX` as a new named type whose underlying type is **inferred**
from the constant expressions inside the block, and simultaneously declares
`XXXEnum1`, `XXXEnum2`, ... as named constants of type `XXX`.

This is semantically equivalent to writing, in Go:

```go
type XXX <inferred underlying type>

const (
    XXXEnum1 XXX = expr1
    XXXEnum2 XXX = expr2
    ...
)
```

but without requiring the author to spell out `<inferred underlying type>`
or repeat `XXX` on every line.

## Syntax

```go
type Identifier const (
    Name1 [= Expr1]
    Name2 [= Expr2]
    ...
)
```

- `Identifier` becomes a new named type.
- Each `Name` becomes a constant of type `Identifier`.
- `Expr` follows the same rules as ordinary `const` declarations, including
  `iota` support and the "repeat previous expression" shorthand:

```go
type Weekday const (
    Sunday = iota
    Monday
    Tuesday
    Wednesday
    Thursday
    Friday
    Saturday
)
```

This expands to underlying type `int` (the default type of `iota`) and is
equivalent to:

```go
type Weekday int

const (
    Sunday Weekday = iota
    Monday
    Tuesday
    Wednesday
    Thursday
    Friday
    Saturday
)
```

## Underlying Type Inference

The underlying type of `XXX` is determined by applying Go's normal constant
type-inference rules to the expressions in the block, then taking the
*default type* of the resulting untyped constant:

| Constant expressions                  | Inferred underlying type |
|----------------------------------------|---------------------------|
| Untyped integer constants (`0`, `iota`) | `int`                      |
| Untyped string constants (`"a"`)        | `string`                   |
| Untyped boolean constants (`true`)      | `bool`                     |
| Untyped float constants (`1.5`)         | `float64`                  |
| Untyped rune constants (`'a'`)          | `rune` (`int32`)           |

### Rules

1. **All expressions must have the same default type.** Mixing, e.g., string
   and integer constants in the same block is a compile-time error:

   ```go
   type Bad const (
       A = 1
       B = "x" // error: inconsistent constant types in enum declaration
   )
   ```

2. **`iota` defaults to `int`**, matching Go's existing behavior.

3. **Explicit type conversions are allowed** for cases where a different
   underlying type is desired, e.g.:

   ```go
   type SmallEnum const (
       A = int8(1)
       B = int8(2)
   )
   // underlying type: int8
   ```

   In this case all expressions must convert to (or default to) the same
   type, following normal Go conversion/typing rules.

4. **The first expression establishes the type** if there is any ambiguity;
   subsequent expressions are checked for consistency against it. This
   mirrors how `iota`-based blocks already behave when only the first entry
   carries an explicit type in Go.

## Examples

### Integer enum (most common case)

```go
type Color const (
    Red   = iota
    Green
    Blue
)
```

Equivalent Go:

```go
type Color int

const (
    Red   Color = iota
    Green
    Blue
)
```

### String enum

```go
type LogLevel const (
    Debug = "debug"
    Info  = "info"
    Warn  = "warn"
    Error = "error"
)
```

Equivalent Go:

```go
type LogLevel string

const (
    Debug LogLevel = "debug"
    Info  LogLevel = "info"
    Warn  LogLevel = "warn"
    Error LogLevel = "error"
)
```

### Boolean-backed enum

```go
type Toggle const (
    Off = false
    On  = true
)
```

Equivalent Go:

```go
type Toggle bool

const (
    Off Toggle = false
    On  Toggle = true
)
```

### Bit-flag enum with explicit expressions

```go
type Permission const (
    Read    = 1 << iota
    Write
    Execute
)
```

Equivalent Go:

```go
type Permission int

const (
    Read    Permission = 1 << iota
    Write
    Execute
)
```

## Compatibility

- This is purely additive syntax sugar at the source level. After XGo's
  compiler expands `type XXX const (...)` into the equivalent Go
  `type` + `const` declarations, the resulting Go AST is indistinguishable
  from hand-written Go code.
- All downstream tooling (type-checking, `go vet`, reflection, JSON
  marshaling, `String()`/`Stringer` codegen via `go generate`, etc.) works
  unchanged, since the generated type and constants are ordinary Go
  declarations.
- Existing Go code using the traditional two-block form remains fully valid
  and is unaffected.
