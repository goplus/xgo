# go.mod and gox.mod Extensions for XGo

XGo builds on the Go module system rather than replacing it. This proposal describes three small extensions that let a Go module carry the extra information XGo needs, while staying fully valid for the standard `go` command:

1. The `go` directive in `go.mod` can name the compiler that should build the module: the official Go compiler, LLGo, or TinyGo.
2. A Go module that contains a class framework provides a `gox.mod` file next to its `go.mod`.
3. A project that uses a class framework marks the corresponding `require` line with `//xgo:class`.

All three are expressed in places the Go toolchain already ignores (comments and an extra file), so existing Go tooling keeps working unchanged.

## Background

XGo is a superset of Go. An XGo project is still a Go module: it has a `go.mod`, it depends on other modules through `require`, and it can be consumed by ordinary Go projects.

Two XGo needs go beyond what `go.mod` says today:

- **Choosing a compiler.** XGo code can be compiled by the official Go compiler, by [LLGo](https://github.com/xgo-dev/llgo) (an LLVM-based Go compiler), or by TinyGo (for small targets such as microcontrollers). A module should be able to say which one it targets, together with the version it expects.
- **Class frameworks.** XGo supports *class frameworks*: libraries that define classfiles (files such as `main.gox` that XGo turns into classes with framework-provided base types and conventions). A command-line App framework such as `github.com/goplus/cobra` is a typical example. Both XGo and LLGo already use it. The toolchain needs a reliable way to know that a dependency contains a class framework and where its metadata lives.

## Motivation

Without these extensions, a build tool has to guess. It cannot tell from `go.mod` alone which compiler a module was written for, and it cannot tell which dependencies contain a class framework without fetching and inspecting each of them. Making both facts explicit in the module files keeps builds reproducible, lets tools such as `xgo`, `llgo` and editors decide what to do before any code is downloaded, and preserves the promise that an XGo module is still a normal Go module.

## Design

### 1. Compiler selection in the `go` directive

The regular `go` directive keeps its meaning: it states the Go language version. To select a compiler other than the official one, add a trailing comment naming the compiler and its version.

The general form is a trailing comment on the `go` line, naming the compiler and its own version:

```
go <go-version> // llgo <llgo-version>
```

or:

```
go <go-version> // tinygo <tinygo-version>
```

For example, with LLGo:

```
go 1.24 // llgo 1.0
```

And with TinyGo:

```
go 1.24 // tinygo 0.35
```

Rules of thumb for practitioners:

- **No comment means the official Go compiler.** A plain `go 1.24` line behaves exactly as it does today. There is no need to write a compiler name for the default case.
- **The version after the compiler name is that compiler's own version**, not a Go version. The Go language version stays in the `go` directive itself.
- **Supported values today are `llgo` and `tinygo`.** Other names are not recognized by this proposal; tools should report an unknown compiler name clearly rather than silently ignoring it.
- **The standard `go` command treats the trailing text as an ordinary comment**, so the module still loads, tidies and builds with the official toolchain.

### 2. `gox.mod` for class framework modules

A Go module that contains a class framework has two module files:

```
mymodule/
├── go.mod
├── gox.mod
└── ...
```

`go.mod` describes the module as Go always has. `gox.mod` marks it as a class framework module and carries the class framework's own metadata.

**Why `gox.mod` and not `xgo.mod`?** The standard file suffix for classfiles is `.gox`, not `.xgo`. The name `gox.mod` therefore says "this is a class framework extension", which is narrower than "this is an XGo extension". Using `xgo.mod` would suggest a general XGo configuration file, which is not what it is.

The content of `gox.mod` is outside the scope of this proposal; see [`gox.mod.md`](gox.mod.md) for the details. Here it matters only that its presence identifies the module as a class framework module.

### 3. The `//xgo:class` marker on `require`

A project that uses a class framework references the framework's Go module through a normal `require`, and adds the marker `//xgo:class` to say that this dependency contains a class framework:

```
require <go-module-path> //xgo:class
```

For example, a command-line App project built on the `cobra` class framework:

```
module example.com/myapp

go 1.24

require github.com/goplus/cobra vX.Y.Z //xgo:class
```

The same marker works inside a `require` block:

```
require (
	github.com/goplus/cobra vX.Y.Z //xgo:class
	github.com/qiniu/x     vX.Y.Z
)
```

Notes:

- The spelling follows the Go directive-comment convention (`//go:generate`, `//go:embed`): no space after `//`, a `tool:name` shape, and the `xgo:` prefix as the namespace. That keeps it easy to recognize and easy to distinguish from human-readable comments.
- The marker applies to the Go module as a whole. If the module contains more than one class framework, a single `//xgo:class` marker covers all of them.
- The marker lets the toolchain know, from the project's own `go.mod`, which dependencies contribute classfile support. It can then look for that module's `gox.mod` without probing every dependency in the build graph.
- Only modules that contain a class framework should carry the marker. Ordinary libraries are required as usual.
- As with the compiler comment, the official `go` command ignores it.

## Putting it together

An XGo command-line App project that is also built with LLGo would have a `go.mod` like this:

```
module example.com/myapp

go 1.24 // llgo 1.0

require github.com/goplus/cobra vX.Y.Z //xgo:class
```

The `cobra` module itself would ship both `go.mod` and `gox.mod`, and the project's source files would be classfiles that the framework knows how to turn into a runnable command-line App.

## Compatibility

- **Official Go toolchain:** all extensions live in comments or in an additional file, so `go build`, `go mod tidy`, `go list` and friends keep working. Comments on `go` and `require` lines are preserved by standard tooling.
- **Existing XGo projects:** projects without the new comment or marker keep their current behavior (official Go compiler, no declared class frameworks).
- **Existing Go modules:** an ordinary Go module without `gox.mod` is not a class framework module and needs no changes.

## Open questions

1. **Interaction with `// indirect`.** `go mod tidy` also writes trailing comments on `require` lines. The exact form for a line that is both indirect and a class framework should be settled and tested against the standard modfile handling.
2. **Compiler version syntax.** Whether compiler versions should accept only bare numbers (`0.11`) or also a leading `v` or a patch level (`0.11.2`), and how tools compare them.
3. **Unknown compiler names.** Whether an unrecognized compiler name should be an error or a warning in the XGo toolchain, and how third-party tools should treat it.
4. **Validation.** Whether the toolchain should verify that a module marked `//xgo:class` really contains `gox.mod`, and what the diagnostic should look like when it does not.

## Summary of the syntax

| Where | Form | Meaning |
| --- | --- | --- |
| `go.mod`, `go` directive | `go <go-version>` | Official Go compiler (default) |
| `go.mod`, `go` directive | `go <go-version> // llgo <llgo-version>` | Build with LLGo |
| `go.mod`, `go` directive | `go <go-version> // tinygo <tinygo-version>` | Build with TinyGo |
| module root | `gox.mod` next to `go.mod` | This module is a class framework module |
| `go.mod`, `require` | `require <module-path> <version> //xgo:class` | This dependency contains a class framework |
