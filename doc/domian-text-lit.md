Domain Text Literals
=====

XGo's domain-specific text literals provide a powerful way to embed specialized languages directly into your code with full syntax highlighting and type safety. This feature bridges the gap between general-purpose programming and domain-specific needs, making your code more expressive and maintainable.

## Overview

Domain-specific text literals allow you to write inline code in specialized formats—such as JSON, XML, regular expressions, or custom DSLs—without sacrificing the benefits of compile-time checking and editor support.

**Basic syntax:**

```go
result := domainTag`content`
```

**With parameters:**

```go
result := domainTag`> param1, param2
content
`
```

The `!` suffix forces error handling, causing a panic if parsing fails—useful for literals you expect to always be valid.

## Design Inspiration

This syntax is inspired by **Markdown's code blocks**. Just as Markdown uses triple backticks with a language identifier (` ```json`) to denote code blocks in a specific language, XGo's domain-specific literals use a similar pattern—a tag followed by backticks—to embed domain-specific content directly in your code. This familiar syntax makes the feature intuitive for developers already comfortable with Markdown while bringing the same clarity and language-specific semantics to your programming workflow.

## Core Benefits

- **Type Safety**: Catch errors at compile time rather than runtime
- **Syntax Highlighting**: Full editor support for embedded languages
- **Readability**: Keep domain-specific code inline where it's used
- **Maintainability**: Easier to update and refactor than string concatenation
- **Tooling Support**: Enables semantic understanding by XGo tools like formatters and IDEs

## Built-in Formats

XGo provides built-in support for domain text literals. All built-in DTLs, except for TPL, are located in the `encoding` directory where the directory name corresponds to the DTL name. Currently supported DTLs include:

- **tpl** - Text Processing Language (located in `tpl/` directory)
- **csv** - CSV data
- **json** - JSON data
- **yaml** - YAML data
- **html** - HTML documents
- **xml** - XML documents
- **regexp** - Regular expressions (RE2 syntax)
- **regexposix** - POSIX regular expressions
- **golang** - Go source code
- **xgo** - XGo source code
- **fs** - File system

### Text Processing Language (tpl)

A grammar-based alternative to regular expressions that emphasizes clarity and composability. Ideal for defining parsers and text processors.

```go
grammar := tpl`
expr = term % ("+" | "-")
term = INT % ("*" | "/")
`!

result := grammar.parseExpr("10+5*2", nil)
echo result
```

Learn more in the [TPL documentation](../tpl/README.md).

### JSON

Parse and validate JSON structures inline. The result is a DOM that supports DQL (DOM Query Language) operations.

```go
config := json`{
	"server": "localhost",
	"port": 8080,
	"features": ["auth", "logging"],
	"database": {
		"host": "db.example.com",
		"port": 5432
	}
}`!

// Access properties using DQL syntax
echo config.server         // "localhost"
echo config.port           // 8080
echo config.database.host  // "db.example.com"

// Query all nested nodes with .**
for item <- config.**.* {
	echo item
}
```

### YAML

Parse YAML content inline. Similar to JSON, the result supports DQL operations.

```go
config := yaml`
server: localhost
port: 8080
database:
  host: db.example.com
  port: 5432
  credentials:
    username: admin
`!

// Access properties using DQL syntax
echo config.server              // "localhost"
echo config.database.host       // "db.example.com"
echo config.database.credentials.username  // "admin"
```

### XML

Work with XML documents directly. The result is a DOM that supports DQL operations.

```go
doc := xml`
<configuration>
	<database>
		<host>localhost</host>
		<port>5432</port>
	</database>
	<servers>
		<server name="primary">192.168.1.1</server>
		<server name="secondary">192.168.1.2</server>
	</servers>
</configuration>
`!

// Navigate using DQL syntax
echo doc.configuration.database.host  // Access nested elements

// Query all server elements
for server <- doc.**.server {
	echo server.$name  // Access attributes with $
}
```

### CSV

Define tabular data inline:

```go
data := csv`
name,age,city
Alice,30,NYC
Bob,25,SF
`!
```

### HTML

Embed HTML with proper parsing. The result is a DOM that supports DQL operations for querying elements.

```go
page := html`
<html>
	<body>
		<h1>Welcome</h1>
		<div class="content">
			<p>First paragraph</p>
			<p>Second paragraph</p>
		</div>
		<a href="https://xgo.dev">XGo Website</a>
	</body>
</html>
`!

// Navigate to specific elements
echo page.html.body.h1  // Access the h1 element

// Query all paragraph elements using .**
for p <- page.**.p {
	echo p
}

// Query all links and get href attribute
for link <- page.**.a {
	echo link.$href  // Access attributes with $
}

// Access elements by class
for div <- page.**.div {
	if div.$class == "content" {
		echo div
	}
}
```

### Regular Expressions

Define regex patterns with improved readability. XGo supports both standard (RE2) and POSIX regex:

```go
pattern := regexp`^[a-z]+\[[0-9]+\]$`!

if pattern.matchString("item[42]") {
	echo "Match found"
}

// POSIX variant
posixPattern := regexposix`[[:alpha:]]+`!
```

### Go Source Code (golang)

Parse Go source code and query its AST (Abstract Syntax Tree). The result is a DOM representing the Go file's structure that supports DQL operations.

```go
code := golang`
package main

import "fmt"

func hello(name string) {
	fmt.Println("Hello,", name)
}

func main() {
	hello("World")
}
`!

// Query all function declarations
for fn <- code.**.FuncDecl {
	echo fn  // Access function AST nodes
}

// Access package name
echo code.Name  // "main"
```

### XGo Source Code (xgo)

Parse XGo source code and query its AST. Similar to `golang`, but for XGo syntax.

```go
code := xgo`
echo "Hello, XGo!"

for i <- 1:10 {
	echo i
}
`!

// Query all statements in the file
for stmt <- code.** {
	echo stmt
}
```

### File System (fs)

The `fs` DTL provides file system access and returns a NodeSet directly (unlike other DTLs which return a single root node). This allows querying directories and files using DQL syntax.

```go
// Get a NodeSet of the current directory
files := fs`.`!

// List all files in current directory
for f <- files.* {
	name, _ := f.Name()
	echo name
}

// List all files recursively (all descendants)
for f <- files.**.file {
	path, _ := f.Path()
	echo path
}

// List only directories
for d <- files.*.Dir() {
	name, _ := d.Name()
	echo name
}

// Query a specific directory
srcFiles := fs`./src`!
for f <- srcFiles.**.file {
	echo f
}

// Filter by pattern
for f <- files.*.Match("*.go") {
	name, _ := f.Name()
	echo name
}
```

## Relationship with DQL (DOM Query Language)

Many DTLs parse content into a DOM (Document Object Model) structure, including:
- **json** - JSON documents
- **yaml** - YAML documents
- **xml** - XML documents
- **html** - HTML documents
- **golang** - Go source code AST
- **xgo** - XGo source code AST

While these aren't typically the NodeSets required by DQL, they can be understood as NodeSets containing only a single root node. Therefore, these DOMs also support NodeSet query operations, including:

- **`.name`** - Access a child element by name
- **`.*`** - Access all direct children
- **`.**`** - Access all descendants recursively
- **`.$attr`** - Access an attribute value

This means you can directly query a DTL result without first converting it to a DQL NodeSet:

```go
// JSON example - direct query without conversion
config := json`{"database": {"host": "localhost", "port": 5432}}`!
echo config.database.host  // "localhost"

// HTML example - query descendants
page := html`<div><p>Hello</p><p>World</p></div>`!
for p <- page.**.p {
	echo p
}
```

### Special Case: File System DTL

The `fs` DTL is unique because it returns a NodeSet directly (not a single root node). For example, `fs`.`` means a NodeSet containing only the current directory as the root:

```go
// fs`.` returns a NodeSet of the current directory
files := fs`.`!

// Query direct children (files and directories in current dir)
for item <- files.* {
	echo item
}

// Query all descendants recursively
for file <- files.**.file {
	echo file
}
```

## Implementation Details

Domain text literals compile to function calls to the corresponding package's `New()` function. For example:

```go
json`{"key": "value"}`
// Compiles to:
json.New(`{"key": "value"}`)
```

This design keeps the feature simple while allowing seamless integration with existing Go packages. The `domainTag` represents a package that must have a global `func New(string)` function with any return type.

All built-in DTLs (except TPL) are implemented in the `encoding/` directory:

```
encoding/
├── csv/       # CSV parser
├── json/      # JSON parser
├── yaml/      # YAML parser
├── html/      # HTML parser (uses github.com/goplus/xgo/dql/html internally)
├── xml/       # XML parser
├── regexp/    # RE2 regular expressions
├── regexposix/# POSIX regular expressions
├── golang/    # Go source code parser
├── xgo/       # XGo source code parser
└── fs/        # File system access
```

Each package exports a `New()` function that accepts the literal content as a string and returns the parsed result.

## Creating Custom Formats

Extend XGo with your own domain-specific languages by implementing a package with a global `New(string)` function:

```go
// Package sql provides SQL query literals
package sql

type Query struct {
	text string
}

func New(query string) (*Query, error) {
	// Validate and parse SQL
	if err := validateSQL(query); err != nil {
		return nil, err
	}
	return &Query{text: query}, nil
}
```

**Usage:**

```go
import "myproject/sql"

query := sql`
SELECT id, name, email 
FROM users 
WHERE active = true
`!
```

## Beyond Syntactic Sugar

Domain text literals offer more than just convenient syntax. They enable XGo tooling to understand the semantics of these embedded texts rather than treating them as ordinary strings. This semantic understanding enables:

- **Code formatters** like `xgo fmt` to format both XGo code and supported domain texts simultaneously
- **IDE plugins** to provide syntax highlighting and advanced features for recognized domain texts
- **Static analysis tools** to validate domain-specific content at build time
- **Documentation generators** to extract and document embedded domain content

## Best Practices

1. **Use the `!` suffix for static literals** that should always be valid—this catches errors early
2. **Handle errors explicitly for dynamic content** that might fail validation
3. **Keep literals focused** on their domain—avoid mixing concerns
4. **Leverage syntax highlighting** by configuring your editor for the embedded languages
5. **Document custom formats** clearly to help other developers understand their usage

## Error Handling

Without the `!` suffix, domain literals return an error that you can handle:

```go
query, err := sql`SELECT * FROM ${table}`
if err != nil {
	return fmt.Errorf("invalid query: %w", err)
}
```

With the `!` suffix, invalid literals cause a panic:

```go
// This panics if the JSON is malformed
data := json`{"invalid": }`!
```

---

## Historical Background

The journey of domain text literals in XGo began with a [community proposal in early 2024](https://github.com/goplus/xgo/issues/1770) suggesting adding JSX syntax support to XGo. While JSX has gained widespread adoption in frontend development, particularly in React-based applications, the immediate benefits of building JSX syntax directly into XGo weren't immediately clear, causing the proposal to be temporarily shelved.

The turning point came when XGo needed to support [TPL (Text Processing Language)](../tpl/README.md) syntax for the [XGo Mini Spec](spec-mini.md) project. This necessity prompted a reconsideration of how XGo should handle domain-specific notations more broadly.

### The Philosophy Behind Domain Text Literals

A common understanding in programming language design suggests that **Domain-Specific Languages (DSLs)** often struggle to compete with general-purpose languages. However, this perspective overlooks the fact that numerous domain languages exist and thrive in specialized contexts:

- **Interface description**: HTML, JSX
- **Configuration and data representation**: JSON, YAML, CSV
- **Text syntax representation**: EBNF-like grammar (including TPL syntax), regular expressions
- **Document formats**: Markdown, DOCX, HTML

What distinguishes these domain languages is that they aren't Turing-complete. They lack the full capabilities of general-purpose languages, such as I/O operations, function definitions, and comprehensive flow control structures.

Rather than competing with general-purpose languages, these domain languages typically complement them. Most mainstream programming languages either officially support or have community-built libraries to interact with these domain languages.

This complementary relationship led to the term "**Domain Text Literals**" rather than "**Domain-Specific Languages**", emphasizing their role as specialized text formats that can be embedded within general-purpose code.

### Syntax Evolution

After considerable deliberation on how XGo should support domain text literals, inspiration came from Markdown's code block syntax. Initially, there was consideration to make XGo's domain text syntax identical to Markdown's. However, this would have prevented XGo code from being embedded as a domain text within Markdown documents, potentially reducing interoperability between XGo and Markdown. After careful consideration, the current syntax was chosen to ensure optimal compatibility while maintaining the familiar, intuitive pattern that developers already know from Markdown.

---

Domain-specific text literals make XGo uniquely suited for projects that need to work with multiple specialized formats. By treating domain-specific languages as first-class citizens, XGo helps you write cleaner, safer, and more maintainable code.
