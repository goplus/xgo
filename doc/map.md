# Map Type

XGo provides a concise syntax for working with maps. Maps are key-value data structures that allow you to store and retrieve values using keys.

## Map Literal Syntax

### Creating Maps

In XGo, you can create maps using curly braces `{}`:

```go
a := {"Hello": 1, "xsw": 3}     // map[string]int
b := {"Hello": 1, "xsw": 3.4}   // map[string]float64
c := {"Hello": 1, "xsw": "XGo"} // map[string]any
d := {}                         // map[string]any
```

### Automatic Type Inference

XGo automatically infers the complete map type `map[KeyType]ValueType` based on the literal syntax and values provided.

#### Type Inference Rules

Both `KeyType` and `ValueType` follow the same inference rules:

1. **All elements have the same type**: The type is that common type. Note that untyped literals (such as untyped integers and untyped floats) follow Go's conversion rules - untyped values can be implicitly converted to a common type when compatible (e.g., untyped integer literals like `1` can be converted to `float64` when mixed with untyped float literals like `3.4`)
2. **Not all elements have the same type**: The type is `any` (which can hold any value)

#### Empty Map Special Case

The empty map literal `{}` is a special case:
- `KeyType` is inferred as `string`
- `ValueType` is inferred as `any`

This provides the most flexible type for an empty map, allowing you to add any string-keyed values dynamically.

## Map Operations

### Getting Map Length

You can get the number of elements in a map using the `len` function:

```go
a := {"a": 1, "b": 2, "c": 3}
echo len(a)  // Output: 3
```

### Accessing Elements

You can access map elements using the `[]` operator with a key:

```go
a := {"name": "Alice", "age": 25}
echo a["name"]  // Output: Alice
```

### Checking Key Existence

To check if a key exists in the map, use the two-value assignment form. The second value is a boolean indicating whether the key exists:

```go
a := {"a": 1, "b": 0}

// Method 1: Using two return values
v, ok := a["c"]
echo v, ok  // Output: 0 false (0 is the default value, false means key doesn't exist)

v, ok = a["b"]
echo v, ok  // Output: 0 true (0 is the actual value, true means key exists)

// Method 2: Direct conditional check
if v, ok := a["c"]; ok {
    echo "Found:", v
} else {
    echo "Not found"
}
// Output: Not found
```

### Adding and Updating Elements

You can add new elements or update existing ones using the `[]` operator:

```go
a := {"a": 1, "b": 0}
a["c"] = 100  // Add new element
a["b"] = 200  // Update existing element
echo a  // Output: map[a:1 b:200 c:100]
```

### Deleting Elements

Use the `delete` function to remove elements from a map:

```go
a := {"a": 1, "b": 0, "c": 100}
delete(a, "b")
echo a  // Output: map[a:1 c:100]
```

### Iterating Over Maps

XGo provides two forms of `for in` loop for iterating over maps:

#### Iterate Over Keys and Values

```go
m := {"x": 10, "y": 20, "z": 30}
for key, value in m {
    echo key, value
}
```

#### Iterate Over Values Only

```go
m := {"x": 10, "y": 20, "z": 30}
for value in m {
    echo value
}
```

## Common Patterns

### Configuration Maps

```go
config := {
    "host": "localhost",
    "port": 8080,
    "debug": true,
}
```

### Counting Occurrences

```go
counts := {}
for item in items {
    counts[item] = counts[item] + 1
}
```

### Lookup Tables

```go
statusCodes := {
    "ok": 200,
    "not_found": 404,
    "error": 500,
}
```

## Best Practices

1. Use descriptive key names for better code readability
2. Check for key existence before accessing values when the key might not exist
3. Initialize empty maps with `{}` when you plan to add elements dynamically
4. Use consistent value types when possible for type safety
