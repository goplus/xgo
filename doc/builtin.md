# Built‐in Functions

XGo extends Go's standard built-in functions with additional capabilities for common operations. This document provides a comprehensive reference for all built-in functions available in XGo.

## Standard Go Built-in Functions

XGo supports all standard Go built-in functions:

### Memory and Data Structure Operations

**`append(slice []Type, elems ...Type) []Type`**

Appends elements to the end of a slice. Returns the updated slice.

```go
numbers := []int{1, 2, 3}
numbers = append(numbers, 4, 5)
echo numbers  // Output: [1 2 3 4 5]
```

**`len(v Type) int`**

Returns the length of arrays, slices, maps, strings, or channels.

```go
echo len("Hello")      // Output: 5
echo len([]int{1,2,3}) // Output: 3
```

**`cap(v Type) int`**

Returns the capacity of arrays, slices, or channels.

```go
s := make([]int, 3, 5)
echo len(s)  // Output: 3
echo cap(s)  // Output: 5
```

**`make(t Type, size ...IntegerType) Type`**

Allocates and initializes slices, maps, or channels.

```go
slice := make([]int, 5, 10)  // length 5, capacity 10
m := make(map[string]int)     // empty map
ch := make(chan int, 10)      // buffered channel
```

**`copy(dst, src []Type) int`**

Copies elements from source to destination slice. Returns the number of elements copied.

```go
src := []int{1, 2, 3}
dst := make([]int, 3)
n := copy(dst, src)
echo n    // Output: 3
echo dst  // Output: [1 2 3]
```

**`delete(m map[Type]Type1, key Type)`**

Deletes a key from a map.

```go
m := map[string]int{"a": 1, "b": 2}
delete(m, "a")
echo m  // Output: map[b:2]
```

**`clear[T ~[]Type | ~map[Type]Type1](t T)`**

Clears all entries in maps or sets slice elements to zero values.

```go
m := map[string]int{"a": 1, "b": 2}
clear(m)
echo len(m)  // Output: 0
```

**`new(Type) *Type`**

Allocates memory and returns a pointer to zero value.

```go
p := new(int)
echo *p  // Output: 0
```

### Numeric Operations

**`max[T cmp.Ordered](x T, y ...T) T`**

Returns the largest value among arguments.

```go
echo max(1, 5, 3, 9, 2)  // Output: 9
echo max(3.14, 2.71)     // Output: 3.14
```

**`min[T cmp.Ordered](x T, y ...T) T`**

Returns the smallest value among arguments.

```go
echo min(1, 5, 3, 9, 2)  // Output: 1
echo min(3.14, 2.71)     // Output: 2.71
```

**`complex(r, i FloatType) ComplexType`**

Constructs a complex number from real and imaginary parts.

```go
c := complex(3.0, 4.0)
echo c  // Output: (3+4i)
```

**`real(c ComplexType) FloatType`**

Returns the real part of a complex number.

```go
c := 3 + 4i
echo real(c)  // Output: 3
```

**`imag(c ComplexType) FloatType`**

Returns the imaginary part of a complex number.

```go
c := 3 + 4i
echo imag(c)  // Output: 4
```

### Control Flow

**`panic(v any)`**

Stops normal execution and begins panicking.

```go
if value < 0 {
    panic("negative value not allowed")
}
```

**`recover() any`**

Recovers from a panic in deferred functions.

```go
defer func() {
    if r := recover(); r != nil {
        println("Recovered from:", r)
    }
}()
```

### Channel Operations

**`close(c chan<- Type)`**

Closes a channel.

```go
ch := make(chan int)
close(ch)
```

## I/O and Formatting Functions

XGo provides enhanced I/O functions with a cleaner API:

### Standard Output

**`Echo(a ...any) (n int, err error)`**

Formats and writes to standard output with spaces between operands and a newline appended.

```go
echo "Hello", "World"  // Output: Hello World
echo 42, true, 3.14    // Output: 42 true 3.14
```

**`Print(a ...any) (n int, err error)`**

Writes to standard output, adding spaces only when neither operand is a string.

```go
print("Hello", "World")  // Output: HelloWorld
print(1, 2, 3)           // Output: 1 2 3
```

**`Println(a ...any) (n int, err error)`**

Writes to standard output with spaces between operands and a newline appended.

```go
println("Hello", "World")  // Output: Hello World
```

**`Printf(format string, a ...any) (n int, err error)`**

Formats according to format specifier and writes to standard output.

```go
printf("Name: %s, Age: %d\n", "Alice", 30)
// Output: Name: Alice, Age: 30
```

### String Formatting

**`Sprint(a ...any) string`**

Returns formatted string, adding spaces when neither operand is a string.

```go
s := sprint("Hello", "World")
echo s  // Output: HelloWorld
```

**`Sprintln(a ...any) string`**

Returns formatted string with spaces between operands and a newline.

```go
s := sprintln("Hello", "World")
echo s  // Output: Hello World
```

**`Sprintf(format string, a ...any) string`**

Returns string formatted according to format specifier.

```go
s := sprintf("Age: %d", 25)
echo s  // Output: Age: 25
```

### Writer Operations

**`Fprint(w io.Writer, a ...any) (n int, err error)`**

Writes to specified writer.

```go
file, _ := create("output.txt")
fprint(file, "Hello", "World")
```

**`Fprintln(w io.Writer, a ...any) (n int, err error)`**

Writes to specified writer with newline.

```go
file, _ := create("output.txt")
fprintln(file, "Hello", "World")
```

**`Fprintf(w io.Writer, format string, a ...any) (n int, err error)`**

Writes formatted output to specified writer.

```go
file, _ := create("output.txt")
fprintf(file, "Count: %d\n", 42)
```

### Error Operations

**`Errorf(format string, a ...any) error`**

Creates formatted error. Supports `%w` verb for error wrapping.

```go
err := errorf("failed to process: %w", originalError)
```

**`Errorln(args ...any)`**

Formats and prints to standard error.

```go
errorln("Warning:", "Something went wrong")
```

**`Fatal(args ...any)`**

Formats and prints to standard error (typically exits program).

```go
fatal("Critical error occurred")
```

## File Operations

**`Open(name string) (*os.File, error)`**

Opens file for reading with O_RDONLY mode.

```go
file, err := open("data.txt")
if err != nil {
    errorln(err)
}
defer file.close
```

**`Create(name string) (*os.File, error)`**

Creates or truncates file with mode 0o666.

```go
file, err := create("output.txt")
if err != nil {
    errorln(err)
}
defer file.close
```

## Type Reflection

**`Type(i any) reflect.Type`**

Returns the reflection Type representing the dynamic type of i.

```go
t := type(42)
echo t.name  // Output: int

s := "hello"
echo type(s).name  // Output: string
```

## Line Reading

XGo provides convenient line reading utilities:

**`Lines(r io.Reader) osx.LineReader`**

Returns a LineReader for reading lines.

```go
file, _ := open("data.txt")
for line in lines(file) {
    echo line
}
```

**`Blines(r io.Reader) osx.BLineReader`**

Returns a BLineReader for reading lines as byte slices.

```go
file, _ := open("data.txt")
for line in blines(file) {
    echo string(line)
}
```

**`(r io.Reader).XGo_Enum() osx.LineIter`**

Returns a LineIter for iterating over lines (supports `for in` syntax).

```go
file, _ := open("data.txt")
for line in file {
    echo line
}
```

## String Methods

When working with strings, XGo provides convenient method syntax for common operations.

### Length and Counting

**`(s string).Len() int`**

Returns the number of bytes in the string.

```go
echo "Hello".len  // Output: 5
```

**`(s string).Count(substr string) int`**

Counts non-overlapping instances of substring.

```go
echo "hello world".count("l")  // Output: 3
echo "aaaa".count("aa")        // Output: 2
```

### Case Conversion

**`(s string).ToUpper() string`**

Converts all letters to uppercase.

```go
echo "Hello".toUpper  // Output: HELLO
```

**`(s string).ToLower() string`**

Converts all letters to lowercase.

```go
echo "Hello".toLower  // Output: hello
```

**`(s string).ToTitle() string`**

Converts all letters to title case.

```go
echo "hello world".toTitle  // Output: HELLO WORLD
```

**`(s string).Capitalize() string`**

Capitalizes the first letter only.

```go
echo "hello world".capitalize  // Output: Hello world
```

### String Manipulation

**`(s string).Repeat(count int) string`**

Returns string repeated count times.

```go
echo "Ha".repeat(3)  // Output: HaHaHa
```

**`(s string).ReplaceAll(old, new string) string`**

Replaces all non-overlapping instances of old with new.

```go
echo "Hello".replaceAll("l", "L")  // Output: HeLLo
```

**`(s string).Replace(old, new string, n int) string`**

Returns a copy of the string with the first `n` non-overlapping instances of `old` replaced by `new`. If `n < 0`, there is no limit on the number of replacements.

```go
s := "hello world"
result := s.replace("world", "XGo", -1)
echo result  // Output: hello XGo
```

### Trimming

**`(s string).Trim(cutset string) string`**

Removes leading and trailing characters from cutset.

```go
echo "  hello  ".trim(" ")  // Output: hello
echo "!!hello!!".trim("!")  // Output: hello
```

**`(s string).TrimSpace() string`**

Removes leading and trailing whitespace.

```go
echo "  hello  ".trimSpace  // Output: hello
```

**`(s string).TrimLeft(cutset string) string`**

Removes leading characters from cutset.

```go
echo "###hello".trimLeft("#")  // Output: hello
```

**`(s string).TrimRight(cutset string) string`**

Removes trailing characters from cutset.

```go
echo "hello###".trimRight("#")  // Output: hello
```

**`(s string).TrimPrefix(prefix string) string`**

Removes leading prefix if present.

```go
echo "Hello World".trimPrefix("Hello ")  // Output: World
```

**`(s string).TrimSuffix(suffix string) string`**

Removes trailing suffix if present.

```go
echo "file.txt".trimSuffix(".txt")  // Output: file
```

### Splitting

**`(s string).Fields() []string`**

Splits string around whitespace.

```go
echo "hello world xgo".fields  // Output: [hello world xgo]
```

**`(s string).Split(sep string) []string`**

Splits string around separator.

```go
echo "a,b,c".split(",")  // Output: [a b c]
```

**`(s string).SplitN(sep string, n int) []string`**

Splits string with count limit.

```go
echo "a-b-c-d".splitN("-", 2)  // Output: [a b-c-d]
```

**`(s string).SplitAfter(sep string) []string`**

Splits after each separator.

```go
echo "a,b,c".splitAfter(",")  // Output: [a, b, c]
```

**`(s string).SplitAfterN(sep string, n int) []string`**

Splits after separator with count limit.

```go
echo "a,b,c,d".splitAfterN(",", 2)  // Output: [a, b,c,d]
```

### Searching

**`(s string).Index(substr string) int`**

Returns index of first instance of substring, or -1 if not found.

```go
echo "hello".index("ll")  // Output: 2
echo "hello".index("x")   // Output: -1
```

**`(s string).IndexByte(c byte) int`**

Returns index of first instance of byte.

```go
echo "hello".indexByte('l')  // Output: 2
```

**`(s string).IndexRune(r rune) int`**

Returns index of first instance of rune.

```go
echo "hello".indexRune('o')  // Output: 4
```

**`(s string).IndexAny(chars string) int`**

Returns index of first instance of any character from chars.

```go
echo "hello".indexAny("aeiou")  // Output: 1 (finds 'e')
```

**`(s string).LastIndex(substr string) int`**

Returns index of last instance of substring.

```go
echo "hello".lastIndex("l")  // Output: 3
```

**`(s string).LastIndexByte(c byte) int`**

Returns index of last instance of byte.

```go
echo "hello".lastIndexByte('l')  // Output: 3
```

**`(s string).LastIndexAny(chars string) int`**

Returns index of last instance of any character.

```go
echo "hello".lastIndexAny("aeiou")  // Output: 4 (finds 'o')
```

### Testing

**`(s string).Contains(substr string) bool`**

Reports whether substring is present.

```go
echo "hello".contains("ll")   // Output: true
echo "hello".contains("xyz")  // Output: false
```

**`(s string).ContainsAny(chars string) bool`**

Reports whether any character from chars is present.

```go
echo "hello".containsAny("aeiou")  // Output: true
```

**`(s string).ContainsRune(r rune) bool`**

Reports whether rune is present.

```go
echo "hello".containsRune('e')  // Output: true
```

**`(s string).HasPrefix(prefix string) bool`**

Reports whether string begins with prefix.

```go
echo "hello".hasPrefix("hel")  // Output: true
echo "hello".hasPrefix("bye")  // Output: false
```

**`(s string).HasSuffix(suffix string) bool`**

Reports whether string ends with suffix.

```go
echo "hello.txt".hasSuffix(".txt")  // Output: true
```

**`(s string).EqualFold(t string) bool`**

Reports case-insensitive equality.

```go
echo "Hello".equalFold("hello")  // Output: true
```

### Comparison

**`(s string).Compare(b string) int`**

Returns 0 if equal, -1 if less, +1 if greater.

```go
echo "abc".compare("abc")  // Output: 0
echo "abc".compare("xyz")  // Output: -1
echo "xyz".compare("abc")  // Output: 1
```

### Quoting

**`(s string).Quote() string`**

Returns double-quoted Go string literal.

```go
echo "hello\nworld".quote  // Output: "hello\nworld"
```

**`(s string).Unquote() (string, error)`**

Interprets string as a quoted Go string literal.

```go
s, err := `"hello\nworld"`.unquote
echo s  // Output: hello
        //         world
```

### Type Conversion

**`(s string).Int() (int, error)`**

Parses string as base-10 integer.

```go
n, err := "42".int
if err == nil {
    echo n  // Output: 42
}
```

**`(s string).Int64() (int64, error)`**

Parses string as 64-bit signed integer.

```go
n, err := "9223372036854775807".int64
if err == nil {
    echo n  // Output: 9223372036854775807
}
```

**`(s string).Uint64() (uint64, error)`**

Parses string as 64-bit unsigned integer.

```go
n, err := "18446744073709551615".uint64
if err == nil {
    echo n  // Output: 18446744073709551615
}
```

**`(s string).Float() (float64, error)`**

Parses string as 64-bit floating-point number.

```go
f, err := "3.14159".float
if err == nil {
    echo f  // Output: 3.14159
}
```

## Numeric Type String Methods

XGo provides String() methods for numeric types to easily convert numbers to strings.

**`(i int).String() string`**

Converts int to base-10 string.

```go
echo (42).string  // Output: 42
```

**`(i int64).String() string`**

Converts int64 to base-10 string.

```go
n := int64(123456789)
echo n.string  // Output: 123456789
```

**`(u uint64).String() string`**

Converts uint64 to base-10 string.

```go
n := uint64(18446744073709551615)
echo n.string  // Output: 18446744073709551615
```

**`(f float64).String() string`**

Converts float64 to string using format 'g' with precision -1.

```go
echo 3.14159.string  // Output: 3.14159
```

## String Slice Methods

XGo provides method syntax for operations on string slices, making batch operations more convenient.

### Information

**`(v []string).Len() int`**

Returns the number of elements.

```go
words := []string{"hello", "world"}
echo words.len  // Output: 2
```

**`(v []string).Cap() int`**

Returns the capacity.

```go
words := make([]string, 2, 5)
echo words.cap  // Output: 5
```

### Joining

**`(v []string).Join(sep string) string`**

Concatenates elements with separator.

```go
words := []string{"hello", "world", "xgo"}
echo words.join(" ")   // Output: hello world xgo
echo words.join(", ")  // Output: hello, world, xgo
```

### Batch Case Conversions

**`(v []string).Capitalize() []string`**

Capitalizes first letter of each string.

```go
words := []string{"hello", "world"}
echo words.capitalize  // Output: [Hello World]
```

**`(v []string).ToTitle() []string`**

Title-cases all strings.

```go
words := []string{"hello", "world"}
echo words.toTitle  // Output: [HELLO WORLD]
```

**`(v []string).ToUpper() []string`**

Upper-cases all strings.

```go
words := []string{"hello", "world"}
echo words.toUpper  // Output: [HELLO WORLD]
```

**`(v []string).ToLower() []string`**

Lower-cases all strings.

```go
words := []string{"HELLO", "WORLD"}
echo words.toLower  // Output: [hello world]
```

### Batch Manipulation

**`(v []string).Repeat(count int) []string`**

Repeats each string count times.

```go
words := []string{"ha", "ho"}
echo words.repeat(3)  // Output: [hahaha hohoho]
```

**`(v []string).Replace(old, new string, n int) []string`**

Replaces occurrences in each string.

```go
words := []string{"hello", "yellow"}
echo words.replace("ll", "LL", -1)  // Output: [heLLo yeLLow]
```

**`(v []string).ReplaceAll(old, new string) []string`**

Replaces all occurrences in each string.

```go
words := []string{"hello", "yellow"}
echo words.replaceAll("l", "L")  // Output: [heLLo yeLLow]
```

### Batch Trimming

**`(v []string).Trim(cutset string) []string`**

Trims each string.

```go
words := []string{"  hello  ", "  world  "}
echo words.trim(" ")  // Output: [hello world]
```

**`(v []string).TrimSpace() []string`**

Removes whitespace from each string.

```go
words := []string{"  hello  ", "  world  "}
echo words.trimSpace  // Output: [hello world]
```

**`(v []string).TrimLeft(cutset string) []string`**

Removes leading characters from each string.

```go
words := []string{"###hello", "###world"}
echo words.trimLeft("#")  // Output: [hello world]
```

**`(v []string).TrimRight(cutset string) []string`**

Removes trailing characters from each string.

```go
words := []string{"hello###", "world###"}
echo words.trimRight("#")  // Output: [hello world]
```

**`(v []string).TrimPrefix(prefix string) []string`**

Removes prefix from each string.

```go
words := []string{"Mr. John", "Mr. Smith"}
echo words.trimPrefix("Mr. ")  // Output: [John Smith]
```

**`(v []string).TrimSuffix(suffix string) []string`**

Removes suffix from each string.

```go
files := []string{"file1.txt", "file2.txt"}
echo files.trimSuffix(".txt")  // Output: [file1 file2]
```

## Notes

### Method Syntax

XGo allows calling many standard library functions as methods on their first argument. This provides a more fluent API while maintaining compatibility with Go's standard library. For example:

```go
// Traditional Go style
s := strings.ToUpper("hello")

// XGo method style
s := "hello".toUpper
```

Both styles work in XGo, giving you flexibility in how you write your code.

### Performance Considerations

When performing batch operations on string slices, the method syntax creates a new slice with transformed values:

```go
// Creates a new slice with all strings uppercased
upper := words.toUpper

// Original slice unchanged
echo words  // Original values
echo upper  // Uppercased values
```

### For In Loops

XGo uses `for in` syntax for iterating over collections, which is more intuitive than Go's traditional `for range`:

```go
// Iterate over slice
words := []string{"hello", "world", "xgo"}
for word in words {
    echo word
}

// Iterate over file lines
file, _ := open("data.txt")
for line in file {
    echo line
}
```
