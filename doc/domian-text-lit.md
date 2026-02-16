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

XGo currently supports several domain text literals natively:

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

Parse and validate JSON structures inline:

```go
config := json`{
	"server": "localhost",
	"port": 8080,
	"features": ["auth", "logging"]
}`!

echo config.port
```

### XML

Work with XML documents directly:

```go
doc := xml`
<configuration>
	<database>
		<host>localhost</host>
		<port>5432</port>
	</database>
</configuration>
`!
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

Embed HTML with proper parsing (requires `golang.org/x/net/html`):

```go
import "golang.org/x/net/html"

page := html`
<html>
	<body>
		<h1>Welcome</h1>
		<p>Domain-specific literals in action</p>
	</body>
</html>
`!
```

### Regular Expressions

Define regex patterns with improved readability. XGo supports both standard and POSIX regex:

```go
pattern := regexp`^[a-z]+\[[0-9]+\]$`!

if pattern.matchString("item[42]") {
	echo "Match found"
}

// POSIX variant
posixPattern := regexposix`[[:alpha:]]+`!
```

## Implementation Details

Domain text literals compile to function calls to the corresponding package's `New()` function. For example:

```go
json`{"key": "value"}`
// Compiles to:
json.New(`{"key": "value"}`)
```

This design keeps the feature simple while allowing seamless integration with existing Go packages. The `domainTag` represents a package that must have a global `func New(string)` function with any return type.

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
