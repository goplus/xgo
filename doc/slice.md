# Slice Type

A slice is a dynamically-sized, flexible view into the elements of an array. Slices are one of the most commonly used data structures in XGo, providing efficient and convenient ways to work with sequences of elements.

**Note**: In XGo, the terms **slice** and **list** are identical and refer to the same data structure. The term "slice" comes from Go's terminology, while "list" aligns with Python's naming convention.

## Understanding Slices

A slice consists of three components:

1. **Pointer** - Points to the first element of the slice in the underlying array
2. **Length** - The number of elements in the slice
3. **Capacity** - The number of elements from the beginning of the slice to the end of the underlying array

Unlike arrays which have a fixed size, slices can grow and shrink dynamically, making them ideal for most collection use cases.

## Creating Slices

XGo provides multiple ways to create slices: using slice literals for quick initialization with data, using the `make` function for more control over slice types and capacity, and slicing existing arrays or slices.

### Slice Literal Syntax

In XGo, you can create slices using square brackets `[]`:

```go
a := [1, 2, 3]    // []int
b := [1, 2, 3.4]  // []float64
c := ["Hi"]       // []string
d := ["Hi", 10]   // []any - mixed types
e := []           // []any - empty slice
```

#### Automatic Type Inference

When using the `:=` syntax without explicit type declaration, XGo automatically infers the complete slice type `[]ElementType` based on the literal values provided.

**Type Inference Rules**

1. **Uniform Types**: If all elements have the same type, that type is used.
2. **Mixed Types**: If elements have incompatible types, the type is inferred as `any`.
3. **Empty Slice** `[]`: Inferred as `[]any` by default for maximum flexibility.

You can also explicitly specify the slice type to override automatic type inference:

```go
// Explicit type declaration
var a []float64 = [1, 2, 3]     // Values converted to float64
var c []any = ["x", 1, true]    // Explicit any type
```

When a type is explicitly declared, the literal values are converted to match the declared type.

### Creating Slices with `make`

Use `make` when you need an empty slice or want to optimize performance by pre-allocating capacity.

#### Basic `make` Syntax

```go
// Create slice with specified length (initialized to zero values)
s1 := make([]int, 5)        // [0, 0, 0, 0, 0]
s2 := make([]string, 3)     // ["", "", ""]

// Access and modify
s1[0] = 100
s1[2] = 300
echo s1  // Output: [100 0 300 0 0]
```

#### Pre-allocating Capacity

For performance optimization, you can specify both length and capacity:

```go
// Create slice with length 0 and capacity 100
s := make([]int, 0, 100)

// This doesn't limit the slice size, but helps reduce allocations
for i := 0; i < 150; i++ {
    s <- i
}

echo len(s)  // Output: 150
echo cap(s)  // Output: likely > 150
```

The capacity hint doesn't limit the slice's size but helps the runtime allocate memory more efficiently when you know approximately how many elements you'll add.

#### When to Use `make` vs Literals

**Use slice literals** (`[]`) when:
- You have initial data to populate
- You want automatic type inference for convenience
- You prefer concise, readable code

**Use slice literals with explicit type** (`var s []T = []`) when:
- You have initial data with a specific type requirement
- You need type safety while keeping syntax concise
- You want to ensure value types are converted correctly

**Use `make`** when:
- You need a slice initialized with zero values
- You want to pre-allocate capacity for performance
- You're creating an empty slice and plan to add elements later
- Working with codebases that consistently use `make`

### Creating Slices from Arrays or Slices

You can create new slices by slicing existing arrays or slices using the range syntax `[start:end]`:

```go
arr := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

// Basic slicing
slice1 := arr[2:5]   // [3, 4, 5] - from index 2 to 5 (exclusive)
slice2 := arr[:3]    // [1, 2, 3] - from start to index 3
slice3 := arr[5:]    // [6, 7, 8, 9, 10] - from index 5 to end
slice4 := arr[:]     // [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] - full slice (shallow copy)
```

**Important**: Slices created this way share the same underlying array. Modifying one slice may affect others:

```go
arr := [1, 2, 3, 4, 5]
slice1 := arr[1:4]  // [2, 3, 4]
slice2 := arr[2:5]  // [3, 4, 5]

slice1[1] = 100     // Modifies the shared underlying array
echo slice1         // Output: [2 100 4]
echo slice2         // Output: [100 4 5] - also affected!
echo arr            // Output: [1 2 100 4 5] - original array modified
```

## Slice Operations

### Modifying Elements

You can directly modify elements at specific indexes:

```go
nums := [1, 2, 3, 4, 5]
nums[0] = 100
nums[2] = 300

echo nums  // Output: [100 2 300 4 5]
```

### Appending Elements

XGo provides two ways to append elements to slices: the `<-` operator and the `append` built-in function.

#### Using the `<-` Operator

The `<-` operator provides an intuitive way to append elements:

```go
nums := [1, 2, 3]
nums <- 4           // Append single element
nums <- 5, 6, 7     // Append multiple elements

more := [8, 9, 10]
nums <- more...     // Append another slice

echo nums  // Output: [1 2 3 4 5 6 7 8 9 10]
```

#### Using the `append` Function

The `append` function returns a new slice with elements added or removed:

```go
// Adding elements
nums := [1, 2, 3]
nums = append(nums, 4)        // Append single element
nums = append(nums, 5, 6, 7)  // Append multiple elements

more := [8, 9, 10]
nums = append(nums, more...)  // Append another slice

echo nums  // Output: [1 2 3 4 5 6 7 8 9 10]
```

**Important**: The `append` function returns a new slice, so you must assign the result back to a variable.

#### Removing Elements with `append`

The `append` function can also remove consecutive elements by concatenating slices before and after the range to remove:

```go
nums := [1, 2, 3, 4, 5]

// Remove element at index 2 (value 3)
nums = append(nums[:2], nums[3:]...)
echo nums  // Output: [1 2 4 5]

// Remove multiple consecutive elements (indices 1-2)
nums = [1, 2, 3, 4, 5]
nums = append(nums[:1], nums[3:]...)
echo nums  // Output: [1 4 5]
```

This pattern uses slice notation to select everything before the removal range (`nums[:start]`) and everything after it (`nums[end:]`), then concatenates them together. This effectively removes the elements in the slice `nums[start:end]`.

### Accessing Elements

Indexes start from `0`. Valid indexes range from `0` to `len(slice) - 1`:

```go
nums := [10, 20, 30, 40, 50]

echo nums[0]   // 10 - first element
echo nums[1]   // 20 - second element
echo nums[4]   // 50 - last element
```

**Note**: Negative indexing is not supported. Using an index outside the valid range will cause a runtime error.

### Getting Slice Length and Capacity

You can get the length and capacity of a slice using the `len` and `cap` functions:

```go
nums := [1, 2, 3, 4, 5]

echo len(nums)  // 5 - number of elements
echo cap(nums)  // 5 - capacity (may be larger)
```

### Extracting Sub-slices

You can extract portions of a slice using the range syntax:

```go
nums := [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]

echo nums[2:5]   // [2 3 4] - from index 2 to 5 (exclusive)
echo nums[:3]    // [0 1 2] - from start to index 3
echo nums[5:]    // [5 6 7 8 9] - from index 5 to end
echo nums[:]     // [0 1 2 3 4 5 6 7 8 9] - full slice (shallow copy)
```

**Remember**: Sub-slices share the underlying array with the original slice. See "Creating Slices from Arrays or Slices" for details on this behavior.

### Iterating Over Slices

XGo provides multiple ways to iterate over slices using `for` loops:

#### Iterate Over Index and Value

```go
nums := [10, 20, 30, 40, 50]

for i, v in nums {
    echo "Index:", i, "Value:", v
}
```

#### Iterate Over Values Only

```go
nums := [10, 20, 30, 40, 50]

for v in nums {
    echo v
}
```

#### Iterate Over Indexes Only

```go
nums := [10, 20, 30, 40, 50]

for i, _ in nums {
    echo "Index:", i
}
```

## List Comprehensions

List comprehensions provide a concise and expressive way to create new lists by transforming or filtering existing sequences. They follow a syntax similar to Python's list comprehensions.

### Basic Syntax

The general form of a list comprehension is:

```go
[expression for vars in iterable]
```

This creates a new list where each element from the `iterable` is transformed by the `expression`.

#### Transforming Elements

```go
// Square all numbers
numbers := [1, 2, 3, 4, 5]
squares := [v * v for v in numbers]
echo squares  // Output: [1 4 9 16 25]

// Convert to strings
words := ["hello", "world"]
upper := [v.toUpper for v in words]
echo upper  // Output: ["HELLO" "WORLD"]

// Extract from index-value pairs
doubled := [v * 2 for i, v in numbers]
echo doubled  // Output: [2 4 6 8 10]
```

#### Creating Lists from Ranges

```go
// Generate sequence
nums := [i for i in 1:11]
echo nums  // Output: [1 2 3 4 5 6 7 8 9 10]

// With transformation
evens := [i * 2 for i in :5]
echo evens  // Output: [0 2 4 6 8]

// With step
odds := [i for i in 1:10:2]
echo odds  // Output: [1 3 5 7 9]
```

### Comprehensions with Conditions

Add an `if` clause to filter elements:

```go
[expression for vars in iterable if condition]
```

#### Filtering Elements

```go
numbers := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

// Only even numbers
evens := [v for v in numbers if v % 2 == 0]
echo evens  // Output: [2 4 6 8 10]

// Only numbers greater than 5
large := [v for v in numbers if v > 5]
echo large  // Output: [6 7 8 9 10]

// Filter and transform
evenSquares := [v * v for v in numbers if v % 2 == 0]
echo evenSquares  // Output: [4 16 36 64 100]
```

#### Filtering with Index

```go
numbers := [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]

// Elements at even indices
evenIndexValues := [v for i, v in numbers if i % 2 == 0]
echo evenIndexValues  // Output: [10 30 50 70 90]
```

### Nested Comprehensions

List comprehensions can be nested to work with multi-dimensional data:

```go
// Flatten a 2D list
matrix := [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
flattened := [num for row in matrix for num in row]
echo flattened  // Output: [1 2 3 4 5 6 7 8 9]

// Create multiplication table
table := [[i * j for j in 1:6] for i in 1:6]
echo table
// Output: [[1 2 3 4 5] [2 4 6 8 10] [3 6 9 12 15] [4 8 12 16 20] [5 10 15 20 25]]

// Extract diagonal elements
diagonal := [matrix[i][i] for i in :len(matrix)]
echo diagonal  // Output: [1 5 9]
```

### Best Practices for Comprehensions

1. **Use comprehensions for simple transformations**: They're most readable when the logic is straightforward
2. **Consider traditional loops for complex logic**: If you need multiple statements or complex conditions, a regular loop may be clearer
3. **Avoid excessive nesting**: More than two levels of nesting can be hard to read
4. **Keep expressions concise**: Long or complex expressions reduce readability
5. **Use meaningful variable names**: Even in short comprehensions, clarity matters

### When to Use Comprehensions vs Loops

**Use list comprehensions** when:
- You need a simple transformation of each element
- You're filtering based on a clear condition
- The logic fits naturally in a single expression
- You want concise, functional-style code

**Use traditional loops** when:
- You need multiple statements per iteration
- You have complex conditional logic
- You need to break or continue based on conditions
- You're modifying external state or have side effects
- Readability would suffer from cramming logic into a comprehension

```go
// Good use of comprehension
squares := [x * x for x in :10]

// Better as a traditional loop (side effects, complex logic)
results := []
for x in :10 {
    result := complexCalculation(x)
    if result > threshold {
        results <- result
        updateGlobalState(result)
    }
}
```

## Common Patterns

### Filtering Slices

```go
nums := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
evens := []

for v in nums {
    if v % 2 == 0 {
        evens <- v
    }
}

echo evens  // Output: [2 4 6 8 10]
```

### Transforming Slices (Map Operation)

```go
nums := [1, 2, 3, 4, 5]
squared := []

for v in nums {
    squared <- v * v
}

echo squared  // Output: [1 4 9 16 25]
```

### Finding Elements

```go
nums := [10, 20, 30, 40, 50]
target := 30
found := false
index := -1

for i, v in nums {
    if v == target {
        found = true
        index = i
        break
    }
}

if found {
    echo "Found", target, "at index", index
} else {
    echo target, "not found"
}
```

### Reversing a Slice

```go
nums := [1, 2, 3, 4, 5]
reversed := []

for i := len(nums) - 1; i >= 0; i-- {
    reversed <- nums[i]
}

echo reversed  // Output: [5 4 3 2 1]
```

### Removing Duplicates

```go
nums := [1, 2, 2, 3, 3, 3, 4, 5, 5]
unique := []
seen := {}

for v in nums {
    if !seen[v] {
        unique <- v
        seen[v] = true
    }
}

echo unique  // Output: [1 2 3 4 5]
```

### Merging Multiple Slices

```go
a := [1, 2, 3]
b := [4, 5, 6]
c := [7, 8, 9]

merged := []
merged <- a...
merged <- b...
merged <- c...

echo merged  // Output: [1 2 3 4 5 6 7 8 9]
```

### Summing Elements

```go
nums := [1, 2, 3, 4, 5]
sum := 0

for v in nums {
    sum += v
}

echo sum  // Output: 15
```

### Finding Maximum and Minimum

```go
nums := [34, 12, 67, 23, 89, 45]

max := nums[0]
min := nums[0]

for v in nums {
    if v > max {
        max = v
    }
    if v < min {
        min = v
    }
}

echo "Max:", max  // Output: 89
echo "Min:", min  // Output: 12
```

### Checking if Slice Contains Element

```go
nums := [10, 20, 30, 40, 50]
target := 30
contains := false

for v in nums {
    if v == target {
        contains = true
        break
    }
}

echo contains  // Output: true
```

### Partitioning a Slice

```go
nums := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
evens := []
odds := []

for v in nums {
    if v % 2 == 0 {
        evens <- v
    } else {
        odds <- v
    }
}

echo evens  // Output: [2 4 6 8 10]
echo odds   // Output: [1 3 5 7 9]
```

### Flattening Nested Slices

```go
nested := [[1, 2], [3, 4], [5, 6]]
flat := []

for subslice in nested {
    flat <- subslice...
}

echo flat  // Output: [1 2 3 4 5 6]
```

### Using Slices as Stacks

```go
stack := []

// Push elements
stack <- 1
stack <- 2
stack <- 3

echo stack  // Output: [1 2 3]

// Pop element
if len(stack) > 0 {
    top := stack[len(stack) - 1]
    stack = stack[:len(stack) - 1]
    echo "Popped:", top  // Output: Popped: 3
    echo stack           // Output: [1 2]
}
```

### Using Slices as Queues

```go
queue := []

// Enqueue elements
queue <- 1
queue <- 2
queue <- 3

echo queue  // Output: [1 2 3]

// Dequeue element
if len(queue) > 0 {
    front := queue[0]
    queue = queue[1:]
    echo "Dequeued:", front  // Output: Dequeued: 1
    echo queue               // Output: [2 3]
}
```

### Sliding Window Pattern

```go
nums := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
windowSize := 3

for i := 0; i <= len(nums) - windowSize; i++ {
    window := nums[i:i + windowSize]
    echo "Window:", window
}
// Output:
// Window: [1 2 3]
// Window: [2 3 4]
// Window: [3 4 5]
// ...
```

### Grouping Data

```go
// Group items by category
items := ["apple", "banana", "carrot", "date", "eggplant"]
groups := make(map[string][]string)

for item in items {
    firstLetter := item[0:1]
    groups[firstLetter] <- item
}

// Access grouped data
for category, itemList in groups {
    echo "Category:", category
    for item in itemList {
        echo "  -", item
    }
}
```

## Best Practices

1. **Pre-allocate capacity when size is known**: Use `make([]T, 0, size)` to avoid multiple reallocations
2. **Use `len(slice)` and `cap(slice)`**: These are the recommended ways to get length and capacity
3. **Check bounds before accessing**: Ensure indexes are within valid range `[0, len(slice)-1]`
4. **Be aware of slice sharing**: Slices created by slicing share the same underlying array
5. **Use the `<-` operator for appending**: It's more concise and idiomatic in XGo
6. **Use meaningful variable names**: Make code self-documenting
7. **Avoid modifying slices during iteration**: Create a new slice instead
8. **Document slice modifications**: Make it clear whether functions modify input slices
9. **Use deep copies when independence is needed**: Use `copy` or manual copying
10. **Consider slice capacity for performance**: Pre-allocating can significantly improve performance for large slices

## Performance Considerations

### Slice Growth

When a slice's capacity is exceeded during append operations, XGo allocates a new underlying array with increased capacity:

```go
s := []
echo len(s), cap(s)  // Output: 0 0

s <- 1
echo len(s), cap(s)  // Output: 1 1

s <- 2
echo len(s), cap(s)  // Output: 2 2

s <- 3
echo len(s), cap(s)  // Output: 3 4 (capacity doubled)

s <- 4, 5
echo len(s), cap(s)  // Output: 5 8 (capacity doubled again)
```

The exact growth strategy may vary, but typically capacity doubles when exceeded.

### Memory Efficiency

Pre-allocating capacity avoids multiple reallocations:

```go
// Inefficient - multiple reallocations
inefficient := []
for i := 0; i < 1000; i++ {
    inefficient <- i
}

// Efficient - single allocation
efficient := make([]int, 0, 1000)
for i := 0; i < 1000; i++ {
    efficient <- i
}
```

### Avoiding Memory Leaks

When creating a small slice from a large slice, the underlying array is still retained:

```go
// May cause memory leak
func getFirstThree(data []int) []int {
    return data[:3]  // Still references the entire underlying array
}

// Better approach - create independent slice
func getFirstThree(data []int) []int {
    result := make([]int, 3)
    copy(result, data[:3])
    return result
}
```

## Common Pitfalls

### 1. Index Out of Bounds

```go
nums := [1, 2, 3]

// This will cause a runtime error
// echo nums[10]  // Error: index out of range

// Always check bounds
index := 10
if index >= 0 && index < len(nums) {
    echo nums[index]
} else {
    echo "Index out of bounds"
}
```

### 2. Negative Indexing Not Supported

```go
nums := [1, 2, 3, 4, 5]

// This is NOT valid in XGo
// echo nums[-1]  // Error: invalid slice index

// To access last element, use:
echo nums[len(nums) - 1]  // Output: 5
```

### 3. Unintended Sharing

```go
a := [1, 2, 3]
b := a       // b references same underlying array
b[0] = 100

echo a  // Output: [100 2 3] - a is also modified!

// To avoid this, make a copy
c := make([]int, len(a))
copy(c, a)
c[0] = 200
echo a  // Output: [100 2 3] - a is not affected
```

### 4. Slice of Slices Sharing

```go
// Careful with slice of slices
matrix := []
row := [1, 2, 3]

matrix <- row
matrix <- row  // Both rows reference the same underlying array!

row[0] = 100
echo matrix  // Output: [[100 2 3] [100 2 3]] - both rows are modified!

// Better approach - create independent rows
matrix := []
matrix <- [1, 2, 3]
matrix <- [1, 2, 3]  // Each row is independent
```

### 5. Modifying During Iteration

```go
// Avoid this - may cause unexpected behavior
nums := [1, 2, 3, 4, 5]
for i, v in nums {
    if v % 2 == 0 {
        nums <- v * 2  // Modifying during iteration - risky!
    }
}

// Better approach - create new slice
result := []
for v in nums {
    if v % 2 == 0 {
        result <- v * 2
    } else {
        result <- v
    }
}
```

## Summary

XGo's slices provide a powerful and flexible way to work with sequences of elements. Key features include:

1. **Simple Literal Syntax**: Use `[]` for concise slice creation
2. **Automatic Type Inference**: No need for explicit type specification in most cases
3. **Intuitive Append Operations**: Use the `<-` operator or `append` function for adding elements
4. **Flexible Slicing**: Create sub-slices with simple range syntax
5. **Multiple Iteration Styles**: Choose the iteration pattern that fits your needs

By understanding these features and following best practices, you can write efficient and maintainable code that leverages the full power of XGo's slice type.
