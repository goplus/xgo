# Map Type

XGo provides a concise syntax for working with maps. Maps are key-value data structures that allow you to store and retrieve values using keys.

## Creating Maps

XGo provides two ways to create maps: using map literals for quick initialization with data, and using the `make` function for more control over map types and capacity.

### Map Literal Syntax

In XGo, you can create maps using curly braces `{}`:

```go
a := {"Hello": 1, "xsw": 3}     // map[string]int
b := {"Hello": 1, "xsw": 3.4}   // map[string]float64
c := {"Hello": 1, "xsw": "XGo"} // map[string]any
e := {1: "one", 2: "two"}       // map[int]string
d := {}                         // map[string]any
```

#### Automatic Type Inference

When using the `:=` syntax without explicit type declaration, XGo automatically infers the complete map type `map[KeyType]ValueType` based on the literal syntax and values provided.

**Type Inference Rules**

Both `KeyType` and `ValueType` follow the same inference rules:

1. **Uniform Types**: If all elements have the same type, that type is used.
2. **Mixed Types**: If elements have different types, the type is inferred as `any`.
3. **Empty Map** `{}`: Inferred as `map[string]any` by default for maximum flexibility.

You can also explicitly specify the map type to override automatic type inference:

```go
var a map[string]float64 = {"Hello": 1, "xsw": 3}  // Values converted to float64
var c map[string]any = {"x": 1, "y": "text"}       // Explicit any type
```

When a type is explicitly declared, the literal values are converted to match the declared type.

### Creating Maps with `make`

Use `make` when you need an empty map or want to optimize performance by pre-allocating capacity.

#### Basic `make` Syntax

```go
// Basic creation
m := make(map[string]int)

m["count"] = 42
echo m  // Output: map[count:42]

// Create a map with complex key types
type Point (x, y int)
positions := make(map[Point]string)
positions[(0, 0)] = "origin"
```

#### Pre-allocating Capacity

For performance optimization, you can specify an initial capacity hint:

```go
// Create a map with initial capacity for ~100 elements
// Pre-allocating capacity (helps performance for large maps)
largeMap := make(map[string]int, 100)

// This doesn't limit the map size, but helps reduce allocations
for i := 0; i < 150; i++ {
    largeMap["key${i}"] = i
}
```

The capacity hint doesn't limit the map's size but helps the runtime allocate memory more efficiently when you know approximately how many elements you'll add.

#### When to Use `make` vs Literals

**Use map literals** (`{}`) when:
- You have initial data to populate
- You want automatic type inference for convenience
- You prefer concise, readable code

**Use map literals with explicit type** (`var m map[K]V = {}`) when:
- You have initial data with a specific type requirement
- You need type safety while keeping syntax concise
- You want to ensure value types are converted correctly

**Use `make`** when:
- You're creating an empty map and plan to add elements later
- You want to pre-allocate capacity for performance
- You prefer the traditional Go style
- Working with codebases that consistently use `make`

## Map Operations

Before manipulating maps, it is important to understand that XGo supports two notations for referencing keys:

- **Bracket Notation** (`m["key"]`): The universal syntax. It works for all key types and allows using variables as keys.
- **Field Access Notation** (`m.key`): A convenient shorthand for string-keyed maps when the key is a valid identifier (no spaces or special characters).

Both notations are used for both **assigning** values and **retrieving** them.

### Adding and Updating Elements

To add a new key-value pair or update an existing one, assign a value to a key using either notation. If the key exists, its value is updated; otherwise, a new entry is created.

```go
a := {"a": 1, "b": 0}

// Using bracket notation
a["c"] = 100

// Using field notation
a.d = 200

echo a  // Output: map[a:1 b:0 c:100 d:200]

// Works with maps created by make too
m := make(map[string]int)
m["x"] = 10
m.y = 20
echo m  // Output: map[x:10 y:20]
```

### Deleting Elements

Use the `delete` function to remove elements from a map:

```go
a := {"a": 1, "b": 0, "c": 100}
delete(a, "b")
echo a  // Output: map[a:1 c:100]

// Works with any key type
m := make(map[int]string)
m[1] = "one"
m[2] = "two"
delete(m, 1)
echo m  // Output: map[2:two]
```

### Getting Map Length

You can get the number of elements in a map using the `len` function:

```go
a := {"a": 1, "b": 2, "c": 3}
echo len(a)  // Output: 3

b := make(map[string]int)
b["x"] = 10
echo len(b)  // Output: 1
```

### Accessing Elements

XGo provides flexible ways to retrieve map values, including safety checks for missing keys.

#### Bracket Notation

The traditional way to access map elements is using the `[]` operator with a key:

```go
a := {"name": "Alice", "age": 25}
echo a["name"]  // Output: Alice

// Works with any key type
m := make(map[int]string)
m[42] = "answer"
echo m[42]  // Output: answer
```

#### Field Access Notation

For string-keyed maps, XGo allows you to use dot notation when the key is a valid identifier:

```go
config := {"host": "localhost", "port": 8080}
echo config.host  // Output: localhost
echo config.port  // Output: 8080

// Equivalent to:
echo config["host"]
echo config["port"]
```

**Field access is pure syntax sugar** - `m.field` and `m["field"]` behave identically in all contexts.

##### When to Use Each Style

**Use field notation** (`m.field`) when:
- Keys are known at development time
- Keys are valid identifiers (letters, digits, underscores only)
- You want more readable code

**Use bracket notation** (`m["key"]`) when:
- Keys are computed at runtime
- Keys contain special characters, spaces, or start with digits
- You need explicit compatibility with standard Go
- Working with non-string key types

```go
// Field notation - clean and readable
user := {"name": "Alice", "age": 30}
echo user.name
echo user.age

// Bracket notation - necessary for dynamic or special keys
keyName := "name"
echo user[keyName]           // Dynamic key
echo user["first-name"]      // Key with hyphen
echo user["2024-score"]      // Key starting with digit

// Bracket notation - required for non-string keys
scores := make(map[int]float64)
scores[1] = 95.5
echo scores[1]  // Must use bracket notation
```

##### Nested Access

Field notation works seamlessly with nested maps:

```go
data := {
    "user": {
        "profile": {
            "name": "Alice",
            "age": 30,
        },
    },
}

// Clean nested access
echo data.user.profile.name  // Output: Alice

// Equivalent to:
echo data["user"]["profile"]["name"]
```

##### Working with `any` Type

Field notation also works with variables of type `any`, automatically treating them as `map[string]any`:

```go
var response any = {"status": "ok", "code": 200}
echo response.status  // Output: ok
echo response.code    // Output: 200
```

#### Safe Access with Comma-ok

When accessing uncertain data (such as from JSON or external APIs), use the comma-ok form to safely check if a path exists. The comma-ok form returns two values:
- The value itself (or zero value if path doesn't exist)
- A boolean indicating whether the access succeeded

```go
a := {"a": 1, "b": 0}

// Check if key exists
v, ok := a["c"]
echo v, ok  // Output: 0 false (key doesn't exist)

v, ok = a["b"]
echo v, ok  // Output: 0 true (key exists with value 0)

// Works with field notation too
v, ok = a.c
echo v, ok  // Output: 0 false

// Direct conditional check
if v, ok := a["c"]; ok {
    echo "Found:", v
} else {
    echo "Not found"  // Output: Not found
}
```

**With comma-ok, accessing non-existent paths never panics** - it simply returns `false`:

```go
data := {"user": {"name": "Alice"}}

// Safe single-level access
name, ok := data.user
if ok {
    echo "User found:", name
}

// Safe nested access
profile, ok := data.user.profile
if !ok {
    echo "Profile not found"  // This will print
}

// Safe access with type assertion
var response any = {"status": "ok", "code": 200}
code, ok := response.code.(int)
if ok {
    echo "Status code:", code
}
```

This is especially useful when working with dynamic data:

```go
var data any = {"user": "Alice"}

// Without comma-ok - may panic if structure is wrong
// name := data.user.profile.name  // Would panic!

// With comma-ok - safe, never panics
name, ok := data.user.profile.name
if !ok {
    echo "Path does not exist"  // Output: Path does not exist
    name = "Unknown"
}

// Processing API response
var apiResponse any = fetchFromAPI()

// Safely extract nested values
if userID, ok := apiResponse.data.user.id.(string); ok {
    processUser(userID)
} else {
    echo "Invalid response structure"
}

// With fallback values
city := "Unknown"
if c, ok := apiResponse.user.address.city.(string); ok {
    city = c
}
echo "City:", city
```

### Iterating Over Maps

XGo provides two forms of `for in` loop for iterating over maps:

#### Iterate Over Keys and Values

```go
m := {"x": 10, "y": 20, "z": 30}
for key, value in m {
    echo key, value
}

// Works with any map type
ages := make(map[string]int)
ages["Alice"] = 30
ages["Bob"] = 25
for name, age in ages {
    echo name, "is", age, "years old"
}
```

#### Iterate Over Keys Only

To iterate over just the keys, you can use the blank identifier `_` for the value part.

```go
m := {"x": 10, "y": 20, "z": 30}
for key, _ in m {
    echo key
}
```

#### Iterate Over Values Only

```go
m := {"x": 10, "y": 20, "z": 30}
for value in m {
    echo value
}
```

## Map Comprehensions

Map comprehensions provide a concise and expressive way to create new maps by transforming or filtering existing sequences. They follow a syntax similar to Python's dictionary comprehensions.

### Basic Syntax

The general form of a map comprehension is:

```go
{keyExpr: valueExpr for vars in iterable}
```

This creates a new map where each element from the `iterable` is transformed into a key-value pair.

#### Creating Maps from Slices

```go
// Map slice values to their indices
numbers := [10, 20, 30, 40, 50]
valueToIndex := {v: i for i, v in numbers}
echo valueToIndex  // Output: map[10:0 20:1 30:2 40:3 50:4]
```

#### Creating Maps from Strings

```go
// Character positions in a string
word := "hello"
charPositions := {char: i for i, char in word}
echo charPositions  // Output: map[h:0 e:1 l:3 o:4]
// Note: 'l' appears twice, so the last occurrence (index 3) is kept
```

### Comprehensions with Conditions

Add an `if` clause to filter elements:

```go
{keyExpr: valueExpr for vars in iterable if condition}
```

#### Filtering Even/Odd Elements

```go
numbers := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

// Only even numbers
evenSquares := {v: v * v for v in numbers if v%2 == 0}
echo evenSquares  // Output: map[2:4 4:16 6:36 8:64 10:100]

// Only odd indices
oddIndexValues := {i: v for i, v in numbers if i%2 == 1}
echo oddIndexValues  // Output: map[1:2 3:4 5:6 7:8 9:10]
```

### Best Practices for Comprehensions

1. **Use comprehensions for simple transformations**: They're most readable when the logic is straightforward
2. **Consider traditional loops for complex logic**: If you need multiple statements or complex conditions, a regular loop may be clearer
3. **Watch for duplicate keys**: In comprehensions, later values overwrite earlier ones for the same key
4. **Keep conditions simple**: Complex filtering logic is often better in a traditional loop

## Common Patterns

### Configuration Maps

```go
// Using literals for initial configuration
config := {
    "host": "localhost",
    "port": 8080,
    "debug": true,
}

// Access with field notation
echo "Connecting to", config.host, "on port", config.port

// Using explicit type for type safety
var settings map[string]int = {
    "maxConnections": 100,
    "timeout": 30,
}

// Using make for type-safe configuration
options := make(map[string]int)
options["maxConnections"] = 100
options["timeout"] = 30
```

### Processing JSON Responses

```go
var response any = parseJSON(apiData)

// Safe extraction with defaults
userID, ok := response.user.id.(string)
if !ok {
    userID = "guest"
}

userName, ok := response.user.name.(string)
if !ok {
    userName = "Anonymous"
}

echo "User:", userName, "(", userID, ")"
```

### Counting Occurrences

```go
// Using make with pre-allocated capacity
wordCounts := make(map[string]int, 1000)
for word in words {
    wordCounts[word]++
}

// Using comprehension to initialize
words := ["apple", "banana", "apple", "orange", "banana", "apple"]
uniqueWords := {w: 0 for w in words}  // Initialize all to 0
for word in words {
    uniqueWords[word]++
}
```

### Lookup Tables

```go
// Simple lookup table with literals
statusCodes := {
    "ok": 200,
    "not_found": 404,
    "error": 500,
}

echo statusCodes.ok  // Output: 200

// Using comprehension to reverse the mapping
codeToStatus := {code: status for status, code in statusCodes}
echo codeToStatus[200]  // Output: ok
```

### Caching

```go
// Cache with pre-allocated capacity for performance
cache := make(map[string][]byte, 10000)

func getCachedData(key string) []byte {
    if data, ok := cache[key]; ok {
        return data
    }

    data := fetchData(key)
    cache[key] = data
    return data
}
```

### Grouping Data

```go
// Group items by category
groups := make(map[string][]string)

for item in items {
    category := getCategory(item)
    groups[category] = append(groups[category], item)
}

// Access grouped data
for category, items in groups {
    echo "Category:", category
    for item in items {
        echo "  -", item
    }
}
```

## Best Practices

1. **Use field notation for readability** when keys are known and are valid identifiers
2. **Use bracket notation** when keys are dynamic, contain special characters, or are non-string types
3. **Use comma-ok form** when working with uncertain data structures (APIs, JSON, dynamic data)
4. **Use map literals** for quick initialization with known data
5. **Use explicit type declaration** with literals when you need type safety or specific conversions
6. **Use `make`** when you need specific types, non-string keys, or want to pre-allocate capacity
7. **Use map comprehensions** for simple transformations and filtering of sequences
8. Check for key existence before accessing values when the key might not exist
9. Pre-allocate capacity with `make` for large maps when the approximate size is known
10. Use consistent value types when possible for type safety
11. Consider using `make` with explicit types for better code documentation and type safety in larger projects

## Performance Tips

1. **Pre-allocate capacity**: When you know the approximate size, use `make(map[K]V, size)` to reduce allocations
2. **Avoid frequent reallocations**: Maps grow dynamically, but pre-allocation prevents repeated internal resizing
3. **Use appropriate key types**: Simple types (int, string) as keys are more efficient than complex structs
4. **Consider zero values**: Accessing non-existent keys returns zero values, which can be useful for counters
5. **Comprehensions vs loops**: For large datasets or complex transformations, traditional loops with pre-allocation may be more efficient than comprehensions
