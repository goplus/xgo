# Functions and Closures

## Basic Function Definition

Functions in XGo are defined with clear type specifications for parameters and return values:

```go
func add(x int, y int) int {
    return x + y
}

echo add(2, 3) // 5
```

### Parameter List Rules

When multiple consecutive parameters share the same type, you can combine them for more concise syntax:

```go
// Verbose form: each parameter has its own type
func add(x int, y int) int {
    return x + y
}

// Concise form: parameters of the same type can be grouped
func add(x, y int) int {
    return x + y
}

// Mixed types require separate declarations
func greet(firstName, lastName string, age int) {
    echo firstName, lastName, "is", age, "years old"
}
```

The same grouping rule applies to return values:

```go
// Multiple return values of the same type can be grouped
func minMax(a, b int) (min, max int) {
    if a < b {
        return a, b
    }
    return b, a
}

// Mixed return types require separate declarations
func divide(a, b float64) (result float64, err error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

**Key points:**
- Parameters or return values of the same type can be grouped: `x, y int` instead of `x int, y int`
- Different types must be declared separately
- This rule applies to both parameter lists and return value lists
- Grouping improves readability without changing functionality

### Functions Without Return Values

Functions don't always need to return values. When a function performs an action without producing a result, you can omit the return type:

```go
func greet(name string) {
    echo "Hello,", name
}

greet("Alice") // Hello, Alice
```

You can also explicitly return early from such functions using a bare `return` statement:

```go
func printPositive(x int) {
    if x <= 0 {
        return // exit early if condition not met
    }
    echo "Positive number:", x
}

printPositive(5)  // Positive number: 5
printPositive(-3) // (prints nothing)
```

### Multiple Return Values

Functions can return multiple values simultaneously:

```go
func foo() (int, int) {
    return 2, 3
}

a, b := foo()
echo a // 2
echo b // 3
c, _ := foo() // ignore values using `_`
```

### Named Return Values

Return values can be named to simplify return statements:

```go
func sum(a ...int) (total int) {
    for x in a {
        total += x
    }
    return // no need to explicitly return named values that are already assigned
}

echo sum(2, 3, 5) // 10
```

## Optional Parameters

XGo supports optional parameters using the `T?` syntax. Optional parameters default to their type's zero value:

```go
func greet(name string, count int?) {
    if count == 0 {
        count = 1
    }
    for i := 0; i < count; i++ {
        echo "Hello,", name
    }
}

greet "Alice", 3  // prints "Hello, Alice" three times
greet "Bob"       // prints "Hello, Bob" once (uses default value)
```

Optional parameters are denoted by adding `?` after the parameter type. The default value is always the zero value of that type (e.g., `0` for integers, `""` for strings, `false` for booleans).

```go
func connect(host string, port int?, secure bool?) {
    if port == 0 {
        port = 80
    }
    echo "Connecting to", host, "on port", port, "secure:", secure
}

connect "example.com", 443, true  // Connecting to example.com on port 443 secure: true
connect "example.com"             // Connecting to example.com on port 80 secure: false
```

## Variadic Parameters

Variadic parameters allow functions to accept a variable number of arguments of the same type using the `...` syntax:

```go
func sum(a ...int) int {
    total := 0
    for x in a {
        total += x
    }
    return total
}

echo sum(2, 3, 5)       // 10
echo sum(1, 2, 3, 4, 5) // 15
echo sum()              // 0
```

Inside the function, the variadic parameter behaves as a slice of the specified type.

### Parameter Positioning Rules

**Important:** The variadic parameter must be the last parameter in the function signature, appearing after all regular parameters and optional parameters:

```go
// ✓ Correct: variadic parameter is last
func log(level string, verbose bool?, messages ...string) {
    // ...
}

// ✗ Wrong: variadic parameter must be last
func log(messages ...string, level string) {  // Error!
    // ...
}

// ✗ Wrong: variadic parameter must be after optional parameters
func log(messages ...string, verbose bool?) {  // Error!
    // ...
}
```

### Combining Regular and Variadic Parameters

```go
func log(level string, messages ...string) {
    echo "[" + level + "]", strings.Join(messages, " ")
}

log("INFO", "Server", "started", "successfully")
// Output: [INFO] Server started successfully
```

### Passing Slices to Variadic Parameters

Use the `...` suffix to pass an existing slice to a variadic parameter:

```go
func max(nums ...int) int {
    if len(nums) == 0 {
        return 0
    }
    maxVal := nums[0]
    for _, n in nums[1:] {
        if n > maxVal {
            maxVal = n
        }
    }
    return maxVal
}

numbers := []int{3, 7, 2, 9, 1}
echo max(numbers...)  // 9
echo max(5, 8, 3)     // 8
```

### Practical Examples

```go
// String formatting
func format(template string, args ...any) string {
    result := template
    for i, arg in args {
        result = strings.Replace(result, "{${i}}", fmt.Sprint(arg), 1)
    }
    return result
}

echo format("Hello {0}, you have {1} messages", "Alice", 5)
// Output: Hello Alice, you have 5 messages
```

## Keyword Arguments

XGo supports keyword arguments syntax for improved code readability and expressiveness. When calling functions with many parameters, you can use `key = value` syntax.

### Using Maps for Keyword Arguments

```go
func process(opts map[string]any?, args ...any) {
    if name, ok := opts["name"]; ok {
        echo "Name:", name
    }
    if age, ok := opts["age"]; ok {
        echo "Age:", age
    }
    echo "Args:", args
}

process name = "Ken", age = 17              // keyword parameters only
process "extra", 1, name = "Ken", age = 17  // variadic parameters first, then keyword parameters
process                                     // all parameters optional
```

**Best for:** Dynamic or extensible parameter sets, diverse parameter types, runtime parameter checking.

### Using Tuples for Keyword Arguments

```go
type Config (timeout, maxRetries int, debug bool)

func run(task int, cfg Config?) {
    if cfg.timeout == 0 {
        cfg.timeout = 30
    }
    if cfg.maxRetries == 0 {
        cfg.maxRetries = 3
    }
    echo "timeout:", cfg.timeout, "maxRetries:", cfg.maxRetries, "debug:", cfg.debug
    echo "task:", task
}

run 100, timeout = 60, maxRetries = 5
run 200
```

**Best for:** Fixed parameter sets with known types, compile-time validation, optimal performance. **Recommended for most use cases.**

**Note:** Tuple field names must match exactly as defined - no automatic case conversion is performed.

### Using Structs for Keyword Arguments

Structs provide type safety and full runtime reflection support for keyword parameters:

```go
type Config struct {
    Timeout    int
    MaxRetries int
    Debug      bool
}

func run(cfg *Config?) {
    timeout := 30
    maxRetries := 3
    debug := false
    if cfg != nil {
        if cfg.Timeout > 0 {
            timeout = cfg.Timeout
        }
        if cfg.MaxRetries > 0 {
            maxRetries = cfg.MaxRetries
        }
        debug = cfg.Debug
    }
    echo "Timeout:", timeout, "MaxRetries:", maxRetries, "Debug:", debug
}

run timeout = 60, maxRetries = 5           // lowercase field names work
run Timeout = 10, Debug = true             // uppercase field names work too
run                                        // uses default values
```

**Best for:** Go codebase compatibility, struct features (methods, embedding, tags), runtime reflection needs.

### Rules and Best Practices

**Syntax Rules:**

1. **Parameter Position Requirements**
   - The keyword parameter must be an optional parameter (marked with `?`)
   - The keyword parameter must be the last parameter (if no variadic parameters), or second-to-last when variadic parameters are present
   
2. **Call Order Requirements**
   - When calling a function, keyword arguments must be placed after all normal parameters (including variadic parameters)

```go
// ✓ Correct call order
process "value", key1 = "a", key2 = "b"
process "v1", "v2", key = "x"

// ✗ Wrong call order
process key = "x", "value"  // keyword arguments must come last
```

**Type Selection:**

- **Use Map** when you need dynamic parameters or runtime flexibility
- **Use Tuple** for most cases - lightweight, compile-time validated, optimal performance (recommended)
- **Use Struct** when you need Go compatibility or struct-specific features

## Higher-Order Functions

Functions can be passed as parameters to other functions:

```go
func square(x float64) float64 {
    return x*x
}

func abs(x float64) float64 {
    if x < 0 {
        return -x
    }
    return x
}

func transform(a []float64, f func(float64) float64) []float64 {
    return [f(x) for x in a]
}

y := transform([1, 2, 3], square)
echo y // [1 4 9]

z := transform([-3, 1, -5], abs)
echo z // [3 1 5]
```

## Lambda Expressions

Lambda expressions provide a concise way to define anonymous functions inline using the `=>` operator.

### Basic Syntax

```go
// Single parameter (no parentheses needed)
x => x * x

// Multiple parameters (parentheses required)
(x, y) => x + y

// No parameters (no parentheses needed, just start with =>)
=> someValue

// Multi-line body
x => {
    result := x * 2
    return result
}
```

### Common Use Cases

**Transformations:**

```go
func transform(a []float64, f func(float64) float64) []float64 {
    return [f(x) for x in a]
}

y := transform([1, 2, 3], x => x*x)           // [1 4 9]
z := transform([-3, 1, -5], x => {
    if x < 0 {
        return -x
    }
    return x
})                                            // [3 1 5]
```

**Combining values:**

```go
func combine(a, b []int, f func(int, int) int) (result []int) {
    for i := 0; i < len(a) && i < len(b); i++ {
        result = append(result, f(a[i], b[i]))
    }
    return result
}

sums := combine([1, 2, 3], [4, 5, 6], (x, y) => x + y)  // [5 7 9]
```

**Closures (capturing variables):**

```go
func multiplier(factor int) func(int) int {
    return x => x * factor
}

func counter(start int) func() int {
    count := start
    return => {
        count++
        return count
    }
}

double := multiplier(2)
echo double(5)  // 10

c := counter(0)
echo c()  // 1
echo c()  // 2
```

**Sorting and filtering:**

```go
numbers := [1, 2, 3, 4, 5, 6]
evens := filter(numbers, x => x % 2 == 0)  // [2 4 6]

sort.Slice(products, (i, j) => products[i].Price < products[j].Price)
```

**Event handling:**

```go
// Event registration functions
func OnStart(onStart func())
func OnMsg(msg string, onMsg func())

// Register event handlers with lambdas
// With one argument (the lambda), no comma is needed.
onStart => {
    echo "Game started!"
    initializeGame()
}
// With multiple arguments, use a comma to separate them.
onMsg "game over", => {
    echo "Game over!"
    cleanup()
}
```

**Note:** If a function with the lowercase name doesn't exist, XGo will automatically look for the capitalized version (e.g., if `onStart` is not defined, it tries `OnStart`). This allows for more flexible and natural function calling syntax.

### Type Inference

Parameter and return types are automatically inferred from context. XGo lambdas do not support explicit type annotations:

```go
// Types are inferred from the function signature
transform([1, 2, 3], x => x * 2)

// The lambda parameter type is inferred from transform's function parameter type
// which expects func(float64) float64
```

### When to Use

**Use lambdas for:**
- One-off functions used inline
- Simple transformations and filters
- Callbacks and event handlers
- Building processing pipelines

**Use named functions for:**
- Reusable logic
- Complex operations needing documentation
- Public APIs

## Summary

XGo's function system combines powerful features:
- Standard function definitions with multiple return values
- Functions without return values for action-oriented operations
- Optional parameters for flexible function calls
- Comprehensive variadic parameters for variable-length argument lists
- Keyword arguments with maps, tuples, or structs for improved readability
- Higher-order functions for functional programming patterns
- Elegant and expressive lambda expressions for inline function definitions
- Closures for capturing and maintaining state

These features work together to create a versatile and expressive function system that supports both traditional imperative programming and modern functional programming paradigms.
