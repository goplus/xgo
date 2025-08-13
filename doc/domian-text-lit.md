Domain Text Literals
=====

The journey of **Domain Text Literals** in XGo began with a proposal in early 2024 when a community member suggested adding JSX syntax support to XGo:

* https://github.com/goplus/xgo/issues/1770

While JSX has gained widespread adoption in frontend development, particularly in React-based applications, the immediate benefits of building JSX syntax directly into XGo weren't immediately clear, causing the proposal to be temporarily shelved.

The turning point came when XGo needed to support [TPL (Text Processing Language)](../tpl/README.md) syntax for the [XGo Mini Spec](spec-mini.md) project. This necessity prompted a reconsideration of how XGo should handle domain-specific notations more broadly.

A common understanding in programming language design suggests that **Domain-Specific Languages (DSLs)** often struggle to compete with general-purpose languages. However, this perspective overlooks the fact that numerous domain languages exist and thrive in specialized contexts:

* **Interface description**: HTML, JSX
* **Configuration and data representation**: JSON, YAML, CSV
* **Text syntax representation**: EBNF-like grammar (including TPL syntax), regular expressions
* **Document formats**: Markdown, DOCX, HTML

What distinguishes these domain languages is that they aren't Turing-complete. They lack the full capabilities of general-purpose languages, such as I/O operations, function definitions, and comprehensive flow control structures.

Rather than competing with general-purpose languages, these domain languages typically complement them. Most mainstream programming languages either officially support or have community-built libraries to interact with these domain languages.

This complementary relationship led to the term "**Domain Text Literals**" rather than "**Domain-Specific Languages**", emphasizing their role as specialized text formats that can be embedded within general-purpose code.

## XGo's Approach to Domain Text Literals

After considerable deliberation on how XGo should support domain text literals, inspiration came from Markdown's code block syntax:

<img src=images/dtl/image-1.png width=960>

Initially, there was consideration to make XGo's domain text syntax identical to Markdown's. However, this would have prevented XGo code from being embedded as a domain text within Markdown documents, potentially reducing interoperability between XGo and Markdown. After careful consideration, the current syntax was chosen to ensure optimal compatibility.

## Built-in Domain Text Literals in XGo

XGo currently supports several domain text literals natively:

* [TPL](../tpl/README.md)
* JSON, XML, CSV
* Regular expressions (regexp, regexposix)
* HTML (requiring import of `"golang.org/x/net/html"`)

Here are some examples demonstrating their usage:

<img src=images/dtl/image-2.png width=960>

## Extensibility and Implementation

One of the powerful aspects of Domain Text Literals in XGo is their extensibility. Users can add support for new domain text formats. The `domainTag` represents a package that must have a global `func New(string)` function (with any return type). The domain text is essentially just a call to this function, making the underlying mechanism remarkably simple.

## Beyond Syntactic Sugar

Domain Text Literals offer more than just convenient syntax. They enable XGo tooling to understand the semantics of these embedded texts rather than treating them as ordinary strings. This semantic understanding enables:

* Code formatters like `xgo fmt` to format both XGo code and supported domain texts simultaneously
* IDE plugins to provide syntax highlighting and advanced features for recognized domain texts
