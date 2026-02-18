DQL - DOM Query Language
=====

DQL is a universal, expressive query language for structured and tree-shaped data, built for [XGo](https://github.com/goplus/xgo). It provides a unified query interface across JSON, XML, HTML, ASTs, file systems, and any other domain that can be modeled as a tree.

---

## Table of Contents

- [Overview](#overview)
- [Supported Domains](#supported-domains)
- [Core Concepts](#core-concepts)
- [Syntax Reference](#syntax-reference)
- [Query Examples](#query-examples)
  - [JSON](#json)
  - [HTML](#html)
  - [XGo AST](#xgo-ast)
  - [File System](#file-system)
- [Error Handling](#error-handling)
- [Performance and Caching](#performance-and-caching)
- [Implementing a NodeSet](#implementing-a-nodeset)
- [Standard Errors](#standard-errors)

---

## Overview

DQL brings a concise, chainable query syntax to XGo. Think of it as XPath or CSS selectors — but designed for XGo's type system, with first-class support for lazy evaluation, error propagation, and domain-specific adaptations.

```go
// Find all hyperlinks in an HTML document
for a in doc.**.a {
    if url := a.$href; url != "" {
        echo url
    }
}

// Filter JSON records
for animal in doc.animals.*@($class == "zebra") {
    echo animal.$at
}

// Walk the file system
for e in fs`.`.**.file.match("*.xgo") {
    echo e.path
}
```

---

## Supported Domains

DQL is not tied to any single format. Any data that can be represented as a tree can be queried with DQL:

- **JSON** — query fields, arrays, and nested objects
- **YAML** — same model as JSON
- **HTML / XML** — traverse elements, attributes, and text
- **Go / XGo AST** — inspect and filter syntax trees
- **File System** — walk directories, match files by name or extension
- **Any custom tree structure** — implement the NodeSet interface

---

## Core Concepts

### NodeSet

Everything in DQL is a `NodeSet`. Queries consume a NodeSet and produce a new NodeSet. The entry point is typically the root document node, called `doc`.

```go
doc NodeSet  // root of the query
```

NodeSets are **lazily evaluated** — they describe a query plan that is executed only when you actually iterate or read data from them. This keeps memory usage low and allows complex query composition without overhead.

### Child Access

Navigate the tree with dot notation:

```
ns.name          // direct child named "name"
ns."elem-name"   // child with a hyphenated name
ns.*             // all direct children (wildcard)
ns.**.name       // all descendants named "name" (deep query)
ns.**.*          // all descendants
ns[0]            // first element of an array/list node (0-based)
```

### Filtering (Select)

Filter nodes using `@`:

```
ns@name          // keep only nodes named "name"
ns@(condExpr)    // keep nodes matching a boolean expression
ns@fn(args)      // keep nodes where fn(...) returns true
```

Examples:

```go
doc.users.*@($role == "admin")         // users whose role attribute is "admin"
doc.users.*@($age.int?:100 < 18)       // users under 18
doc.**.div@hasClass("widget")          // all divs with a given CSS class
```

### Attributes

Read node-level data with `$`:

```go
val := ns.$name              // single-value: zero value on error
val, err := ns.$name         // dual-value: explicit error handling
val := ns.$name!             // panic on error
val := ns.$name?:"default"   // use a fallback value on error
val := ns.$"attr-name"       // hyphenated attribute names
```

Attribute access always operates on the **first node** in the NodeSet. To collect an attribute from every node, use a list comprehension:

```go
names := [user.$name for user in doc.users.*]
```

### Methods

Methods provide access to computed or typed data:

```go
text := div.text             // call Text() method (single-value)
text, err := div.text        // dual-value
count := div.count?:0        // explicit default for numeric methods
age, err := node.$age.int    // numeric methods require dual-value
```

Methods prefixed with `_` map to `XGo_`-prefixed implementations:

```go
node._text    // → node.XGo_text()
node._count   // → node.XGo_count()
node._first   // → node.XGo_first()
```

> **Note on numeric methods**: Methods that return numeric types (like `int`, `float`, `count`) do **not** have a single-value form, to prevent silent bugs where a zero default masks a real error. Always use the dual-value form or an explicit `?:` default.

---

## Syntax Reference

| Syntax | Meaning |
|---|---|
| `ns.name` | Direct child named `name` |
| `ns."elem-name"` | Direct child with special characters |
| `ns.*` | All direct children |
| `ns[n]` | Child at index `n` (0-based) |
| `ns.**.name` | Deep search for `name` |
| `ns.**.*` | All descendants |
| `ns@name` | Filter: keep nodes named `name` |
| `ns@(expr)` | Filter: keep nodes matching expression |
| `ns.$name` | Attribute (single-value, zero on error) |
| `val, err := ns.$name` | Attribute (dual-value) |
| `ns.$name!` | Attribute, panic on error |
| `ns.$name?:def` | Attribute, custom default on error |
| `ns.method(args)` | Method call |
| `ns._method` | Method call via `XGo_method()` |
| `ns.all` / `ns._all` | Materialize and cache all results |
| `ns.one` / `ns._one` | First match, early termination |
| `ns.single` / `ns._single` | Exactly one match (validates uniqueness) |

---

## Query Examples

### JSON

```go
doc := json`{
    "animals": [
        {"class": "gopher", "at": "Line 1"},
        {"class": "armadillo", "at": "Line 2"},
        {"class": "zebra", "at": "Line 3"},
        {"class": "unknown", "at": "Line 4"},
        {"class": "gopher", "at": "Line 5"},
        {"class": "bee", "at": "Line 6"},
        {"class": "gopher", "at": "Line 7"},
        {"class": "zebra", "at": "Line 8"}
    ]
}`!

// Iterate filtered records
for animal in doc.animals.*@($class == "zebra") {
    echo animal.$at
}

// Collect all names into a list
names := [a.$class for a in doc.animals.*]

// Access by index
first := doc.animals[0].$class

// Find unique admin (validate exactly one exists)
user := doc.users.*@($role == "admin")._single
name, err := user.$name
```

### HTML

```go
import "os"
import "github.com/goplus/xgo/dql/html"

doc := html.source(os.Args[1])

// Print all hyperlink URLs
for a in doc.**.a {
    if url := a.$href; url != "" {
        echo url
    }
}

// Collect text from all paragraphs
texts := [p.text for p in doc.**.p]

// Find the first element with a specific class (early termination)
widget := doc.**.*@($class == "widget").one
id := widget.$id
```

### XGo AST

```go
doc := xgo`
x, y := "Hi", 123
echo x
print y
`!

// Find all expression statements
stmts := doc.shadowEntry.body.list.*@(self.class == "ExprStmt")

// Extract function names from call expressions
for fn in stmts.x@(self.class == "CallExpr").fun@(self.class == "Ident") {
    echo fn.$name
}
```

### File System

```go
// Walk current directory and print all .xgo files
for e in fs`.`.**.file.match("*.xgo") {
    echo e.path
}

// Collect all Go source file names
names := [f.$name for f in root.**.file@match("*.go", $name)]
```

---

## Error Handling

DQL integrates with XGo's error handling operators and follows a consistent two-version design for attribute and method access.

### Two-Version Design

Most accessors come in two forms:

```go
// Single-value: convenient, returns zero on error
name := node.$name

// Dual-value: explicit, lets you inspect the error
name, err := node.$name
```

Additionally, XGo's error operators work directly on DQL expressions:

```go
name := node.$name!           // panic if error
name := node.$name?:"N/A"     // custom fallback
```

### NodeSet Error State

Some operations return a NodeSet that internally carries an error (e.g., `_one` when no match is found, `_single` when zero or multiple matches are found). This error propagates automatically to any subsequent attribute or method access:

```go
admin := doc.users.*@($role == "admin")._one

// All of these reflect the internal ErrNotFound:
name := admin.$name              // returns zero value
name, err := admin.$name         // err == dql.ErrNotFound
name := admin.$name!             // panics
```

An empty NodeSet is still a valid NodeSet — loops simply don't execute:

```go
for user in admin {
    // Not reached if admin has ErrNotFound
}
```

---

## Performance and Caching

DQL uses **lazy evaluation** by default. Queries are not executed until you iterate or read from the NodeSet. This is memory-efficient and enables query composition, but means a NodeSet re-executes its query each time it is accessed.

Use cache control methods to avoid repeated execution:

### `_all` / `all` — Materialize Everything

Executes the query and caches all results. Use when you need to access the same NodeSet multiple times.

```go
users := doc.users.*@($active == true)._all

names  := [u.$name  for u in users]   // uses cache
emails := [u.$email for u in users]   // uses cache, no re-query
```

### `_one` / `one` — First Match, Early Exit

Stops after finding the first matching node. Use when you know (or assume) there is at most one result.

```go
admin := doc.users.*@($role == "admin")._one
name := admin.$name   // ErrNotFound if no match
```

### `_single` / `single` — Uniqueness Validation

Validates that exactly one node matches. Returns `ErrMultipleEntities` if more than one is found.

```go
user := doc.users.*@($id == 12345)._single
name, err := user.$name
// err may be ErrNotFound or ErrMultipleEntities
```

### Choosing the Right Method

| Need | Use |
|---|---|
| Multiple accesses to same results | `_all` |
| Expect one result, want fast exit | `_one` |
| Require exactly one result | `_single` |
| Single-pass iteration | no cache (default lazy) |

---

## Implementing a NodeSet

To make a custom data source queryable with DQL, implement the following interface on your NodeSet type.

### Required Methods

```go
// Iteration
func (ns NodeSet) XGo_Enum() iter.Seq[NodeSet]

// Extract first node
func (ns NodeSet) XGo_first() (NodeType, error)

// Type conversion from raw sequence
func NodeSet_Cast(seq func(func(NodeType) bool)) NodeSet
```

### Child Navigation

```go
func (ns NodeSet) XGo_Elem(name string) NodeSet    // ns.name
func (ns NodeSet) XGo_Child() NodeSet              // ns.*
func (ns NodeSet) XGo_Index(index int) NodeSet     // ns[n]
func (ns NodeSet) XGo_Any(name string) NodeSet     // ns.**.name  ("" = all)
```

### Filtering

```go
func (ns NodeSet) XGo_Select(name string) NodeSet  // ns@name
// Conditional filters are compiler-generated using NodeSet_Cast
```

### Attribute Access

```go
func (ns NodeSet) XGo_Attr__0(name string) ValueType          // single-value
func (ns NodeSet) XGo_Attr__1(name string) (ValueType, error) // dual-value
```

### Cache Control

Provide either general-form (`XGo_all`, `XGo_one`, `XGo_single`) or domain-specific form (`All`, `One`, `Single`) methods:

```go
func (ns NodeSet) XGo_all()    NodeSet  // or All()
func (ns NodeSet) XGo_one()    NodeSet  // or One()
func (ns NodeSet) XGo_single() NodeSet  // or Single()
```

Domain types (`NodeType`, `ValueType`, `IndexType`) are customized per implementation.

---

## Standard Errors

The `dql` package defines two standard sentinel errors:

```go
package dql

var (
    ErrNotFound         = errors.New("node not found")
    ErrMultipleEntities = errors.New("multiple entities found, expected single")
)
```

These are returned by `_one` (when no match exists) and `_single` (when zero or more than one match exists), and propagate through the NodeSet to any subsequent attribute or method call.
