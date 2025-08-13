TPL: Text Processing Language
=====

Text processing is a common task in programming, and regular expressions have long been the go-to solution. However, regular expressions are notorious for their cryptic syntax and poor readability. Enter XGo TPL (Text Processing Language), an enhanced alternative that offers both power and intuitive syntax.

XGo TPL is a grammar-based language similar to EBNF (Extended Backus-Naur Form) that seamlessly integrates with XGo. It provides a more readable and maintainable approach to text processing while offering capabilities beyond what regular expressions can achieve.

## Understanding XGo TPL

To understand XGo TPL, you need to grasp three key concepts:

### 1. Naming Rules

The foundation of TPL is its naming rules, expressed as `name = rule`. A TPL grammar consists of a series of named rules, with the first one being the root rule. The `rule` can be a combination of:

* **Basic Tokens**: Fundamental syntax units like `INT`, `FLOAT`, `CHAR`, `STRING`, `IDENT`, `"+"`, `"++"`, `"+="`, `"<<="`, etc.
* **Keywords**: An `IDENT` enclosed in quotes, such as `"if"`, `"else"`, `"for"`.
* **References**: References to other named rules, including self-references.
* **Sequence**: `R1 R2 ... Rn` - matches a sequence of rules.
* **Alternatives**: `R1 | R2 | ... | Rn` - matches any one of the rules.
* **Repetition Operators**:
  * `*R` - matches the rule zero or more times
  * `+R` - matches the rule one or more times
  * `?R` - matches the rule zero or one time (optional)
* **List Operator**: `R1 % R2` - shorthand for `R1 *(R2 R1)`, representing a sequence of R1 separated by R2. For example, `INT % ","` represents a comma-separated list of integers.
* **Adjacency Operator**: `R1 ++ R2` - indicates that R1 and R2 must be adjacent with no whitespace or comments between them.

The default operator precedence is: unary operators (`*R`, `+R`, `?R`) > `++` > `%` > sequence (space) > `|`. Parentheses can be used to change the precedence.

#### String Literals in Detail

`STRING` (string literals) can take two forms:

```go
"Hello\nWorld\n"  // QSTRING (quoted string)

`Hello
World
`               // RAWSTRING (raw string)
```

`STRING` can be defined as:

```go
STRING = QSTRING | RAWSTRING
```

#### The Adjacency Operator Explained

Since TPL rules automatically filter whitespace and comments, the sequence `R1 R2` doesn't express that R1 and R2 are adjacent. This is where the adjacency operator `++` comes in.

For example, XGo [domain text literal](../doc/domian-text-lit.md) is defined as `IDENT ++ RAWSTRING`, making these valid:

```go
tpl`expr = INT % ","`
json`{"name": "Ken", age: 15}`
```

While these would match `IDENT STRING` but are not valid domain text literals:

```go
tpl"expr = *INT"              // IDENT must be followed by RAWSTRING, not QSTRING
tpl/* comment */`expr = *INT` // No whitespace or comments allowed between IDENT and RAWSTRING
```

### 2. Matching Results

Each rule has its built-in matching result:

* **Tokens and Keywords**: Result is `*tpl.Token`.
* **Sequence** (`R1 R2 ... Rn`): Result is a list (`[]any`) with n elements.
* **Repetition** (`*R`, `+R`): Result is a list (`[]any`) with elements depending on how many times R matches.
* **Alternatives** (`R1 | R2 | ... | Rn`): Result depends on which rule matches.
* **Optional** (`?R`): Result is either the result of R or `nil` if no match.
* **List Operator** (`R1 % R2`): Result is a complex tree-like structure with three levels. 

  Let's explain why it has three levels:
  1. The first level is the result of the entire expression `R1 % R2` (i.e.`R1 *(R2 R1)`), which is a list with two elements.
  2. The first element of this list is the result of the first `R1`.
  3. The second element is a list containing the results of all subsequent `(R2 R1)` matches.
     - Each element in this second-level list is itself a list with two elements: the result of `R2` and the result of `R1`.
     
  For example, when parsing `"1, 2, 3"` with `INT % ","`, the result structure would be:
  ```
  [
    <INT:1>,                // First R1
    [
      [<COMMA>, <INT:2>],   // First (R2 R1)
      [<COMMA>, <INT:3>]    // Second (R2 R1)
    ]
  ]
  ```
  This tree-like structure preserves all the information about the matched elements and their relationships, but can be complex to work with directly. That's why TPL provides helper functions like `ListOp` and `BinaryOp` to transform this structure into more usable forms.

* **Adjacency Operator** (`R1 ++ R2`): Result is a list (`[]any`) with 2 elements, similar to a `R1 R2` sequence.

### 3. Rewriting Matching Results

The default matching result is called "self" in TPL. You can rewrite this result using an XGo closure `=> { ... }`.

This feature is crucial as it allows seamless integration between TPL and XGo. In XGo, you reference TPL through [domain text literal](../doc/domian-text-lit.md), and within TPL, you can call XGo code through result rewriting.

## Practical Examples

### Basic Example: Parsing Integers

```go
import "xgo/tpl"

cl := tpl`
expr = INT % "," => {
    return tpl.ListOp[int](self, v => {
        return v.(*tpl.Token).Lit.int!
    })
}
`!

echo cl.parseExpr("1, 2, 3", nil)!  // Outputs: [1 2 3]
```

This example parses a comma-separated list of integers and converts it to a flat list of integers using TPL's `ListOp` function.

### Building a Calculator

Creating a calculator with XGo TPL is remarkably concise:

```go
import "xgo/tpl"

cl := tpl`
expr = operand % ("*" | "/") % ("+" | "-") => {
    return tpl.BinaryOp(true, self, (op, x, y) => {
        switch op.Tok {
        case '+': return x.(float64) + y.(float64)
        case '-': return x.(float64) - y.(float64)
        case '*': return x.(float64) * y.(float64)
        case '/': return x.(float64) / y.(float64)
        }
        panic("unexpected")
    })
}

operand = basicLit | unaryExpr

unaryExpr = "-" operand => {
    return -(self[1].(float64))
}

basicLit = INT | FLOAT => {
    return self.(*tpl.Token).Lit.float!
}
`!

echo cl.parseExpr("1 + 2 * -3", nil)!  // Outputs: -5
```

This calculator handles basic arithmetic operations with proper operator precedence in less than 30 lines of code.

## Conclusion

XGo TPL offers a powerful yet intuitive alternative to regular expressions for text processing. By combining grammar-based parsing with seamless XGo integration, it enables developers to create clear, maintainable text processing solutions.

For more examples of TPL in action, check out the XGo demos starting with `tpl-` at [https://github.com/goplus/xgo/tree/main/demo](https://github.com/goplus/xgo/tree/main/demo). These examples showcase how to implement calculators, parse text to generate ASTs, and even implement entire languages in just a few hundred lines of code.

Whether you're parsing structured text, building domain-specific languages, or implementing complex text transformations, XGo TPL provides a robust and readable approach that surpasses traditional regular expressions.
