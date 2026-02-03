# String Type

XGo provides powerful and flexible string handling capabilities. Strings are sequences of characters used to represent text, and they are one of the most commonly used data types in programming.

## String Literals

XGo provides multiple ways to represent strings, from simple literals to complex Unicode characters.

### Basic String Literals

In XGo, you can create strings using double quotes:

```go
name := "Bob"
message := "Hello, World!"
empty := ""
```

### Escape Sequences

XGo supports various escape sequences for special characters:

```go
// Common escape sequences
newline := "Line 1\nLine 2"     // Newline
tab := "Column1\tColumn2"       // Tab
quote := "She said \"Hello\""   // Double quote
backslash := "Path: C:\\files"  // Backslash

// Octal escape notation \### where # is an octal digit
octalChar := "\141a"            // aa

// Unicode can be specified as \u#### where # is a hex digit
// It will be converted internally to its UTF-8 representation
star := "\u2605"                // ★
heart := "\u2665"               // ♥
```

#### Common Escape Sequences

| Sequence | Description | Example |
|----------|-------------|---------|
| `\n` | Newline | `"Line 1\nLine 2"` |
| `\t` | Tab | `"Name:\tAlice"` |
| `\\` | Backslash | `"C:\\path"` |
| `\"` | Double quote | `"He said \"Hi\""` |
| `\###` | Octal character | `"\141"` (a) |
| `\u####` | Unicode character | `"\u2605"` (★) |

## String Immutability

String values are immutable in XGo. Once created, you cannot modify individual characters:

```go failcompile
s := "hello 🌎"
s[0] = `H` // Error: not allowed
```

To modify a string, you must create a new one:

```go
s := "hello"
s = "Hello"  // OK: assigning a new string
s = s + " world"  // OK: creating a new concatenated string
```

## String Operations

### Indexing and Slicing

#### Indexing

Indexing a string returns a `byte` value (not a `rune` or another `string`):

```go
name := "Bob"
echo name[0]   // 66 (ASCII value of 'B')
echo name[1]   // 111 (ASCII value of 'o')
echo name[2]   // 98 (ASCII value of 'b')
```

**Warning**: Indexing into multi-byte characters (like Chinese characters or emojis) will return individual bytes, which may not represent a complete character.

#### Slicing

You can extract substrings using slice notation:

```go
name := "Bob"
echo name[1:3]  // ob (from index 1 to 3, exclusive)
echo name[:2]   // Bo (from start to index 2)
echo name[2:]   // b (from index 2 to end)

s := "Hello, World!"
echo s[0:5]     // Hello
echo s[:5]      // Hello (start defaults to 0)
echo s[7:]      // World! (end defaults to string length)
```

Slicing syntax:
- `s[start:end]` - from index `start` to `end` (exclusive)
- `s[:end]` - from beginning to `end`
- `s[start:]` - from `start` to end of string

### String Conversion

#### Converting Strings to Integers

Strings can be easily converted to integers:

```go
s := "12"
a, err := s.int    // Returns value and error (safe conversion)
b := s.int!        // Panics if s isn't a valid integer (unsafe conversion)

// Example with error handling
if num, err := s.int; err == nil {
    echo "Valid number:", num
} else {
    echo "Invalid number"
}
```

#### Converting Other Types to Strings

To convert other types to strings, use the `.string` property:

```go
age := 10
ageStr := age.string
echo "age = " + ageStr  // age = 10

pi := 3.14159
piStr := pi.string
echo "π = " + piStr  // π = 3.14159
```

## String Operators

### Concatenation

Use the `+` operator to concatenate strings:

```go
name := "Bob"
bobby := name + "by"
echo bobby // Bobby

s := "Hello "
s += "world"
echo s // Hello world

// Multiple concatenations
greeting := "Hello" + " " + "World" + "!"
echo greeting  // Hello World!
```

### Type Consistency

XGo operators require values of the same type on both sides. You cannot concatenate an integer directly to a string:

```go failcompile
age := 10
echo "age = " + age // Error: not allowed
```

You must convert `age` to a string first:

```go
age := 10
echo "age = " + age.string  // age = 10
```

## String Interpolation

XGo supports string interpolation using `${expression}` syntax, which provides a cleaner alternative to concatenation:

```go
age := 10
echo "age = ${age}"  // age = 10

name := "Alice"
greeting := "Hello, ${name}!"
echo greeting  // Hello, Alice!
```

### Complex Interpolation

You can use any expression inside `${...}`:

```go
// Arithmetic expressions
x := 5
y := 3
echo "${x} + ${y} = ${x + y}"  // 5 + 3 = 8

// Function calls and method calls
name := "bob"
echo "Hello, ${name.toUpper}!"  // Hello, BOB!

// Complex example
host := "example.com"
page := 0
limit := 20
url := "https://${host}/items?page=${page+1}&limit=${limit}"
echo url  // https://example.com/items?page=1&limit=20
```

### Escaping the Dollar Sign

To include a literal `$` in a string, use `$$`:

```go
echo "Price: $$50"  // Price: $50
echo "Total: $$${100 + 50}"  // Total: $150
```

## String Methods

XGo provides built-in methods for common string operations. These methods do not modify the original string (strings are immutable) but return new strings.

### String Length

You can get the length of a string using the `len` method:

```go
name := "Bob"
echo name.len  // 3

chinese := "你好"
echo chinese.len  // 6 (bytes, not characters - each Chinese character is 3 bytes in UTF-8)
```

**Important**: `len` returns the number of bytes, not the number of characters. For strings containing non-ASCII characters (like Chinese, emojis), the byte length will be larger than the character count.

### Case Conversion

```go
// Convert to uppercase
echo "Hello".toUpper  // HELLO
echo "hello world".toUpper  // HELLO WORLD

// Convert to lowercase
echo "Hello".toLower  // hello
echo "HELLO WORLD".toLower  // hello world

// Capitalize first letter of each word
echo "hello world".capitalize  // Hello World
echo "the quick brown fox".capitalize  // The Quick Brown Fox
```

### String Repetition

```go
// Repeat a string n times
echo "XGo".repeat(3)  // XGoXGoXGo
echo "Ha".repeat(5)  // HaHaHaHaHa
echo "-".repeat(10)  // ----------

// Useful for formatting
separator := "=".repeat(40)
echo separator
echo "Title"
echo separator
```

### String Replacement

```go
// Replace all occurrences
echo "Hello".replaceAll("l", "L")  // HeLLo
echo "banana".replaceAll("a", "o")  // bonono

// Practical example
text := "The quick brown fox"
censored := text.replaceAll("fox", "***")
echo censored  // The quick brown ***
```

### Joining Strings

Join a list of strings into a single string with a separator:

```go
// Join with comma
fruits := ["apple", "banana", "cherry"]
echo fruits.join(",")  // apple,banana,cherry

// Join with space
words := ["Hello", "World"]
echo words.join(" ")  // Hello World

// Join without separator
letters := ["H", "e", "l", "l", "o"]
echo letters.join("")  // Hello

// Practical example with newlines
lines := ["Line 1", "Line 2", "Line 3"]
text := lines.join("\n")
echo text
// Output:
// Line 1
// Line 2
// Line 3
```

### Splitting Strings

Split a string into a list of substrings using a separator:

```go
// Split by delimiter
subjects := "Math-English-Science-History"
subjectList := subjects.split("-")
echo subjectList  // [Math English Science History]

// Split by space
sentence := "The quick brown fox"
words := sentence.split(" ")
echo words  // [The quick brown fox]

// Split CSV data
csv := "Alice,30,Engineer"
fields := csv.split(",")
echo fields  // [Alice 30 Engineer]

// Process split results
for field in fields {
    echo "Field:", field
}
```

## Characters and Bytes

In XGo, strings can be traversed by character (`rune`) or by byte. Understanding the difference is crucial when working with non-ASCII characters.

### Character Encoding Basics

- **ASCII characters** (like English letters, digits): 1 byte per character
- **Non-ASCII characters** (like Chinese, emojis): 2-4 bytes per character (UTF-8 encoding)
- **`len`** returns byte count, not character count
- **Indexing** returns bytes, not complete characters

### Iterating by Character (Rune)

Use `for in` loop to iterate over characters (runes):

```go
// English text (1 byte per character)
s := "Hello"
for c in s {
    echo c
}
// Output:
// H
// e
// l
// l
// o

// Mixed text with Chinese characters
s := "你好XGo"
for c in s {
    echo c
}
// Output:
// 你
// 好
// X
// G
// o
```

### Iterating by Byte

Use traditional index-based loop to iterate over bytes:

```go
s := "Hello"
for i := 0; i < len(s); i++ {
    echo s[i]  // Prints byte values: 72, 101, 108, 108, 111
}

// With non-ASCII characters
s := "你好XGo"
for i := 0; i < len(s); i++ {
    echo s[i]
}
// Outputs byte values (each Chinese character is 3 bytes)
// For '你': 228, 189, 160
// For '好': 229, 165, 189
// For 'X': 88
// For 'G': 71
// For 'o': 111
```

### Working with Non-ASCII Characters

**⚠️ Important Warnings**:

1. **Length discrepancy**: `len()` returns bytes, not character count
2. **Indexing multi-byte characters**: Accessing individual bytes of multi-byte characters yields incomplete data
3. **Use character iteration**: When processing text with non-ASCII characters, use `for c in s` instead of index-based loops

```go
// Example: Chinese characters
s := "你好"

// WRONG: This returns byte count, not character count
echo s.len  // 6 (bytes)

// WRONG: This returns part of a character
echo s[0]  // 228 (first byte of '你')

// CORRECT: Count characters
count := 0
for _ in s {
    count++
}
echo count  // 2 (characters)

// CORRECT: Process characters
for char in s {
    echo char  // Prints: 你, then 好
}
```

### Practical Example: Character vs Byte Processing

```go
// Process English text (safe with either method)
english := "Hello"
echo "Length:", english.len  // 5
echo "Characters:", countChars(english)  // 5

// Process mixed text (need character iteration)
mixed := "Hello你好"
echo "Byte length:", mixed.len  // 11 (5 ASCII + 6 for Chinese)

charCount := 0
for _ in mixed {
    charCount++
}
echo "Character count:", charCount  // 7

// Extract characters correctly
for i, char in mixed {
    echo "Character ${i}: ${char}"
}
// Output:
// Character 0: H
// Character 1: e
// Character 2: l
// Character 3: l
// Character 4: o
// Character 5: 你
// Character 6: 好
```

## Common Patterns

### String Validation

```go
// Check if string is a valid integer
input := "12345"
if num, err := input.int; err == nil {
    echo "Valid number:", num
} else {
    echo "Invalid number"
}

// Check string length
username := "alice"
if username.len < 3 {
    echo "Username too short"
} else if username.len > 20 {
    echo "Username too long"
} else {
    echo "Username OK"
}
```

### String Formatting

```go
// Build formatted strings
name := "Alice"
age := 30
city := "New York"

// Using interpolation
profile := "Name: ${name}, Age: ${age}, City: ${city}"
echo profile

// Building multi-line strings
header := "=".repeat(40)
title := "User Profile"
content := "${header}\n${title}\n${header}\nName: ${name}\nAge: ${age}\nCity: ${city}"
echo content
```

### String Parsing

```go
// Parse CSV data
csv := "Alice,30,Engineer,New York"
fields := csv.split(",")
name := fields[0]
age := fields[1].int!
job := fields[2]
city := fields[3]

echo "Name: ${name}, Age: ${age}, Job: ${job}, City: ${city}"

// Parse key-value pairs
config := "host=localhost;port=8080;debug=true"
pairs := config.split(";")
settings := {}
for pair in pairs {
    parts := pair.split("=")
    key := parts[0]
    value := parts[1]
    settings[key] = value
}
echo settings  // map[host:localhost port:8080 debug:true]
```

### String Templates

```go
// Email template
func generateEmail(name, action, link string) string {
    return "Hello ${name},\n\nPlease click the link below to ${action}:\n${link}\n\nBest regards,\nThe Team"
}

email := generateEmail("Alice", "verify your email", "https://example.com/verify")
echo email

// HTML template
func generateHTML(title, content string) string {
    return "\n${title}\n\n${title}\n${content}\n\n"
}

html := generateHTML("Welcome", "Welcome to our website!")
echo html
```

### Text Processing

```go
// Word count
text := "The quick brown fox jumps over the lazy dog"
words := text.split(" ")
echo "Word count:", words.len

// Capitalize each word
capitalized := []
for word in words {
    capitalized = append(capitalized, word.capitalize)
}
result := capitalized.join(" ")
echo result  // The Quick Brown Fox Jumps Over The Lazy Dog

// Remove extra spaces
messyText := "  Too   many    spaces   "
cleaned := messyText.split(" ").join(" ")
echo cleaned
```

### Building Complex Strings

```go
// Building a URL with query parameters
func buildURL(base string, params map[string]any) string {
    if params.len == 0 {
        return base
    }
    
    queryParts := []
    for key, value in params {
        queryParts = append(queryParts, "${key}=${value}")
    }
    
    return "${base}?${queryParts.join("&")}"
}

url := buildURL("https://api.example.com/search", {
    "q": "xgo",
    "page": 1,
    "limit": 20,
})
echo url  // https://api.example.com/search?q=xgo&page=1&limit=20

// Building a report
func buildReport(title string, items []string) string {
    separator := "=".repeat(50)
    header := "${separator}\n${title}\n${separator}"
    
    itemList := []
    for i, item in items {
        itemList = append(itemList, "${i+1}. ${item}")
    }
    
    return "${header}\n${itemList.join("\n")}"
}

report := buildReport("Task List", ["Review code", "Write tests", "Update docs"])
echo report
```

## Best Practices

1. **Use string interpolation** (`"${expr}"`) instead of concatenation for better readability
2. **Use `.string` method** to convert other types to strings
3. **Check string length** before accessing indices to avoid runtime errors
4. **Use character iteration** (`for c in s`) when processing text with non-ASCII characters
5. **Prefer string methods** over manual manipulation for common operations
6. **Handle conversion errors** when converting strings to numbers using the comma-ok form
7. **Remember strings are immutable** - methods return new strings rather than modifying originals
8. **Use escape sequences** for special characters rather than trying to insert them literally
9. **Be aware of byte vs. character distinction** when working with internationalized text
10. **Use appropriate string methods** (`.toUpper`, `.toLower`, etc.) for case-insensitive operations

## Performance Tips

1. **Avoid excessive concatenation in loops**: Build string slices and join them instead
   ```go
   // Less efficient
   result := ""
   for i := 0; i < 1000; i++ {
       result += "item${i},"
   }
   
   // More efficient
   parts := []
   for i := 0; i < 1000; i++ {
       parts = append(parts, "item${i}")
   }
   result := parts.join(",")
   ```

2. **Use string interpolation**: It's more efficient than multiple concatenations
   ```go
   // Less efficient
   message := "Hello, " + name + "! You are " + age.string + " years old."
   
   // More efficient
   message := "Hello, ${name}! You are ${age} years old."
   ```

3. **Reuse string slices**: When splitting strings multiple times, consider reusing slices

4. **Consider byte operations**: For performance-critical ASCII-only operations, byte-level processing can be faster

5. **Preallocate when building large strings**: If you know the approximate size, preallocate capacity

