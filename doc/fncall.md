# Commands and Function Calls

In XGo, there's a fundamental unity beneath seemingly different syntactic forms: **commands, function calls, and operators are all essentially function invocations**. This unified model makes the language both intuitive for beginners and consistent for experienced programmers.

## The Unified Function Model

Consider these three ways of invoking functions:

```go
echo "Hello"           // Command style
echo("Hello")          // Function call style
3 + 4                  // Operator style
```

While they look different, all three represent function invocations. The different syntaxes simply provide flexibility in how you express intent.

### Command Style: Natural and Intuitive

Commands look like natural language instructions:

```go
echo "Hello world"
println "Temperature:", 25.5
time.sleep 2*time.Second
```

**Key characteristic:** Parentheses are optional. Arguments follow the command naturally, making code read like sentences.

### Function Call Style: Explicit and Familiar

Function calls use traditional syntax with mandatory parentheses:

```go
echo("Hello world")
println("Temperature:", 25.5)
time.sleep(2*time.Second)
```

**Key characteristic:** Explicit parentheses make nesting and composition clearer in complex expressions.

### Operators: Mathematical Notation for Functions

Operators use familiar mathematical notation:

```go
3 + 4        // Addition
x * y        // Multiplication
a == b       // Equality comparison
```

While operators look like special symbols, they're actually function calls in disguise. The `+` operator calls an addition function, `*` calls a multiplication function, and so on. This syntax matches mathematical conventions, making numeric code natural to read and write.

**Note:** XGo allows you to define your own operators (covered in advanced topics), reinforcing that operators are truly functions at their core.

## Categories of Functions

Functions in XGo come from three sources, each accessed slightly differently:

### 1. Built-in Functions

Built-in functions are always available without any imports. They're part of the language core.

#### Input/Output Functions

```go
echo "Hello", "World"     // Output with spaces and newline
print "Hello", "World"    // Output without spaces, no newline
```

Both forms work identically:
```go
echo "Result:", 42         // Command style
echo("Result:", 42)        // Function call style
```

**Difference between `echo` and `print`:**
- `echo` adds spaces between arguments and ends with a newline
- `print` concatenates arguments directly without spaces or newline

```go
echo "A", "B", "C"    // Output: A B C\n
print "A", "B", "C"   // Output: ABC
```

#### Error Handling

```go
panic "Something went wrong!"
panic("Fatal error: division by zero")
```

The `panic` function stops program execution immediately—use it for unrecoverable errors.

#### Operators as Built-in Functions

Arithmetic and comparison operators are also built-in functions:

```go
sum := 3 + 4           // Addition operator
product := 5 * 6       // Multiplication operator
equal := (x == y)      // Equality operator
```

The operator syntax is designed to match mathematical notation, but conceptually these are function invocations. This is why you can define custom operators in XGo—they're not special language primitives, just functions with infix notation.

### 2. Package Functions

Functions from packages are accessed through import and qualified names.

#### Importing Packages

Place all imports at the beginning of your file:

```go
import "math"
import "time"
```

Or use the grouped form:

```go
import (
    "math"
    "time"
)
```

#### Using Package Functions

Access package functions with dot notation: `packageName.functionName`

```go
import "math"

echo math.sqrt(16)        // Square root: 4
echo math.pow(2, 3)       // Power: 8
echo math.abs(-5)         // Absolute value: 5
```

**Lowercase calling convention:** XGo provides a convenient feature—exported functions (which start with uppercase letters in Go convention) can be called with lowercase names:

```go
// In the math package, the actual function is Sqrt (uppercase)
math.sqrt(16)    // ✓ Lowercase call (recommended in XGo)
math.Sqrt(16)    // ✓ Original uppercase name (also works)

// But you cannot call a lowercase function with uppercase
// somePackage.DoSomething()  // ✗ Won't work if function is actually doSomething
```

This feature is specifically designed to make code more readable while maintaining compatibility with Go's export rules. The convention is:
- Exported functions start with uppercase (Go requirement)
- You can call them with lowercase for convenience (XGo feature)
- The reverse is not true—lowercase functions must be called with lowercase

**Omitting parentheses:** For zero-parameter functions, parentheses are optional when using lowercase names:

```go
import "time"

echo time.now        // Current time (no parentheses needed)
echo time.now()      // Same thing with explicit call
echo time.Now()      // Also works with uppercase
```

#### Common Package Examples

**Math operations:**
```go
import "math"

math.sqrt(16)              // 4
math.pow(2, 8)             // 256
math.max(10, 20)           // 20
math.min(10, 20)           // 10
math.Pi                    // 3.141592653589793 (constant, not a function)
```

**Time operations:**
```go
import "time"

time.now                   // Current timestamp
time.sleep 2*time.Second   // Pause for 2 seconds
```

**Note on constants:** Packages also provide constants like `math.Pi` and `time.Second`. These are accessed the same way as functions but represent fixed values rather than executable code.

### 3. Methods: Functions Belonging to Objects

Methods are functions that operate on specific objects. They're called using dot notation: `object.method()`

Think of methods as actions an object can perform. A string can be converted to uppercase, a time can tell you what day of the week it is.

#### String Methods

Strings have built-in methods for common operations:

```go
"Hello".len              // 5 (length of string)
"Hello".toUpper          // "HELLO"
"Hello".toLower          // "hello"
"Go".repeat(3)           // "GoGoGo"
"Hello".replaceAll("l", "L")  // "HeLLo"
```

**Zero-parameter methods:** Like package functions, methods without parameters can omit parentheses:

```go
"Hello".len        // Parentheses optional
"Hello".len()      // Explicit call—same result
```

**Methods with parameters:** Require parentheses:

```go
"Go".repeat(3)                    // Must use parentheses
"Hello".replaceAll("l", "L")      // Must use parentheses
```

#### Time Methods

Time objects returned from `time.now` have methods to extract components:

```go
import "time"

now := time.now

echo now.weekday      // e.g., "Wednesday"
echo now.year         // e.g., 2025
echo now.month        // e.g., "February"
echo now.day          // e.g., 15
echo now.hour         // e.g., 14 (24-hour format, UTC)
echo now.minute       // e.g., 30
echo now.second       // e.g., 45
```

All these methods work without parentheses since they take no parameters.

**Chaining method calls:**

```go
import "time"

echo time.now.weekday         // Current day of week
echo time.now.year            // Current year
```

## Understanding the Dot Notation

The dot (`.`) connects an object to its method or a package to its function:

```go
// Package.function
math.sqrt(16)
time.now

// Object.method
"Hello".toUpper
time.now.weekday
```

Both follow the same pattern, making the language consistent and predictable.

## Practical Examples

### Combining Different Function Types

```go
import "math"
import "time"

// Built-in + Package function
echo "Square root of 25 is", math.sqrt(25)

// Package function + Method
echo "Today is", time.now.weekday

// Operator + Built-in + Method
result := 3 + 4
message := "Result: " + result.string
echo message.toUpper
```

### Command vs. Function Call Style

Use command style for simple, top-level statements:

```go
echo "Starting calculation..."
result := math.sqrt(144)
echo "Result:", result
```

Use function call style for nested expressions:

```go
// Clear nesting with explicit parentheses
echo math.sqrt(math.pow(3, 2) + math.pow(4, 2))  // Pythagorean theorem

// String method in expression
name := "alice"
echo "Hello, " + name.toUpper()
```

### Real-World Example

```go
import "time"
import "math"

// Get current time details
now := time.now
echo "Current time:", now
echo "Day:", now.weekday
echo "Date:", now.year, "-", now.month, "-", now.day

// Wait a bit
time.sleep 2*time.Second

// Do some calculations
value := math.pow(2, 10)
echo "2^10 =", value
echo "Square root:", math.sqrt(value)

// String manipulation
message := "processing complete"
echo message.toUpper()
```

## Key Takeaways

1. **Everything is a function call** at the conceptual level—commands, function calls, and operators all invoke functions
2. **Operators use mathematical notation** but represent function calls underneath, allowing for custom operator definitions
3. **Built-in functions** like `echo`, `print`, and operators are always available
4. **Package functions** require imports and use `package.function` syntax
5. **Methods** are functions that belong to objects, using `object.method` syntax
6. **Parentheses are optional** for commands, and for zero-parameter functions and methods when using lowercase names
7. **Lowercase calling convention:** Uppercase-exported functions can be called with lowercase (e.g., `math.sqrt` for `math.Sqrt`), but not vice versa

This unified model means once you understand one form, you understand them all. Whether you write `echo "Hello"` or `echo("Hello")`, whether you use `3 + 4` or call `time.now`, you're invoking functions—just with different syntactic styles suited to different situations.
