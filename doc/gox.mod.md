## Overview
 
`gox.mod` is the module configuration file for **XGo** (formerly known as Go+). It belongs to **class framework packages** — not to ordinary XGo application projects. Its role is analogous to a plugin manifest: it tells the XGo toolchain how a given framework's class system is structured, what file extensions map to which class types, and which packages should be auto-imported.
 
> **Note:** Ordinary XGo projects do **not** need a `gox.mod` file. They use a standard `go.mod` file, just like any Go project. `gox.mod` only lives inside class framework packages themselves.
 
> **Legacy alias:** In older versions of XGo, this file was named `gop.mod`. The `gop` directive is still accepted for backward compatibility.
 
---
 
## How XGo Finds `gox.mod`
 
When you run `xgo run` (or `xgo build`, `xgo install`, `xgo test`, etc.) on a project, XGo does **not** look for a `gox.mod` in your project directory. Instead, it follows this discovery process:
 
### 1. Scan `go.mod` for class framework dependencies
 
XGo reads your project's `go.mod` and inspects every `require` entry for an `//xgo:class` annotation:
 
```
require (
    github.com/goplus/spx/v2 v2.0.0 //xgo:class
    github.com/goplus/yap v0.8.0 //xgo:class
)
```
 
Any dependency annotated with `//xgo:class` is recognized as a class framework.
 
### 2. Locate the framework's `gox.mod`
 
For each identified class framework, XGo locates the corresponding `gox.mod` file from the framework package — either from the local module cache or a local `replace` path.
 
### 3. Parse `gox.mod` for class metadata
 
XGo parses the `gox.mod` to learn:
 
- Which file extension patterns map to which project class (e.g. `main.spx` → `Game`)
- Which file extension patterns map to which work class (e.g. `*.spx` → `SpriteImpl`)
- Which packages should be auto-imported into every source file
 
### 4. Parse source files via `xgo/parser`
 
Armed with the file glob patterns from `gox.mod`, the `xgo/parser` package knows exactly which files to treat as which class type and parses them accordingly.
 
This design means your project stays clean — just a `go.mod` and your source files. All the class system configuration is owned and versioned by the framework package itself.
 
---
 
## File Structure
 
A `gox.mod` file is composed of a small set of directives. Each directive occupies one line (or a line block). The supported directives are:
 
| Directive | Scope | Purpose |
|-----------|-------|---------|
| `xgo`     | File  | Declares the required XGo version |
| `project` | File  | Declares a classfile project entry point |
| `class`   | Project | Declares a work class within the current project |
| `import`  | Project | Declares auto-imported packages for the current project |
 
---
 
## Directives
 
### `xgo` — XGo Version
 
```
xgo <version>
```
 
Specifies the minimum XGo language version required by this module. The version must follow the `1.x` or `1.x.y` format.
 
**Example:**
```
xgo 1.6.0
```
 
The directive was historically also written as `gop` and is still accepted under that name.
 
---
 
### `project` — Project Declaration
 
```
project [*.projExt ProjectClass] classFilePkgPath ...
```
 
Declares a classfile project. A project defines the entry-point file extension, the Go class that represents the project, and one or more Go package paths that implement the classfile framework.
 
**Arguments:**
 
| Argument | Required | Description |
|----------|----------|-------------|
| `*.projExt` or `*_projTag.gox` | Optional | Glob pattern for the project file extension (e.g. `main.spx`, `*_app.gox`) |
| `ProjectClass` | Required if ext given | The Go type name used as the project class (e.g. `Game`) |
| `classFilePkgPath ...` | Required | One or more Go package import paths; the first is the primary classfile package |
 
**Examples:**
```
# Project with extension and class
project main.spx Game github.com/goplus/spx/v2 math
 
# Project with package path only (no dedicated entry file)
project github.com/example/myframework
```
 
**Extension formats:**
 
The project extension (`*.projExt`) can be expressed in several forms:
 
| Pattern | Meaning |
|---------|---------|
| `main.spx` | Only `main.spx` is a project file |
| `*.spx` | Any `.spx` file is a project file |
| `main_yap.gox` | Only `main_yap.gox` is a project file |
| `*_yap.gox` | Any `*_yap.gox` file is a project file |
 
---
 
### `class` — Work Class Declaration
 
```
class [-embed] [-prefix=Prefix] *.workExt WorkClass [WorkPrototype]
```
 
Declares a **work class** within the most recently declared `project`. A work class represents the individual actor or object type (e.g. a sprite in a game) that operates within the project. Multiple `class` directives can follow a single `project`.
 
**Flags:**
 
| Flag | Description |
|------|-------------|
| `-embed` | The class instance is embedded in the project struct (composition via embedding) |
| `-prefix=Prefix` | Attaches a name prefix to the work class |
 
**Arguments:**
 
| Argument | Required | Description |
|----------|----------|-------------|
| `*.workExt` or `*_workTag.gox` | Required | Glob pattern for work class source files (e.g. `*.spx`, `*_cmd.gox`) |
| `WorkClass` | Required | The Go type name of the work class (e.g. `SpriteImpl`) |
| `WorkPrototype` | Optional | The prototype types are required when there are multiple work classes |
 
**Example:**
```
class -embed *.spx SpriteImpl
```
 
This declares that every `*.spx` file (other than the project entry file) is compiled as a `SpriteImpl` work class, and that the class instance is embedded into the project.
 
---
 
### `import` — Auto-Import Declaration
 
```
import [name] pkgPath
```
 
Declares a package to be **automatically imported** into every source file of the current project. This allows classfile frameworks to inject utility packages transparently.
 
**Arguments:**
 
| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Optional | Local alias for the imported package |
| `pkgPath` | Required | The Go package import path |
 
**Example:**
```
project .yap YapApp github.com/goplus/yap
 
import "github.com/goplus/yap/test"
```
 
---
 
## Real-World Example: SPX Game Framework
 
The following is the `gox.mod` configuration for the **spx** classfile framework, which powers XGo's 2D game programming environment:
 
```
xgo 1.6.0
 
project main.spx Game github.com/goplus/spx/v2 math
 
class -embed *.spx SpriteImpl
```
 
### What this means
 
1. **`xgo 1.6.0`** — This module requires XGo version 1.6.0 or later.
 
2. **`project main.spx Game github.com/goplus/spx/v2 math`**
   - The file `main.spx` is the **project entry point**.
   - It is compiled as a `Game` class (from `github.com/goplus/spx/v2`).
   - The `math` standard library package is also part of the classfile's package set.
 
3. **`class -embed *.spx SpriteImpl`**
   - Every other `*.spx` file in the project (i.e., files that are *not* `main.spx`) is compiled as a `SpriteImpl` work class.
   - The `-embed` flag means each sprite instance is embedded directly into the `Game` project struct, enabling direct method calls between the game and its sprites.
 
### How an application project uses this framework
 
An application project built on `spx` does **not** have a `gox.mod`. It has a `go.mod` that references the framework with the `//xgo:class` annotation:
 
```
module mygame
 
go 1.21
 
require github.com/goplus/spx/v2 v2.0.0 //xgo:class
```
 
When `xgo run` is invoked, it reads this annotation, finds `spx/v2`'s `gox.mod`, and learns that `main.spx` files are `Game` instances and all other `*.spx` files are `SpriteImpl` instances.
 
### Typical project layout
 
```
mygame/
├── go.mod          # standard Go module file (no gox.mod needed here)
├── main.spx        # project file → compiled as Game
├── Cat.spx         # work file → compiled as SpriteImpl
└── Dog.spx         # work file → compiled as SpriteImpl
```
 
---
 
## Summary
 
`gox.mod` is the heart of XGo's classfile system, but it lives in **framework packages**, not in user projects. Ordinary XGo projects use a plain `go.mod` with `//xgo:class` annotations on their framework dependencies — that's the signal `xgo run` and other toolchain commands use to discover the relevant `gox.mod` files and learn the class structure (file patterns, class types, auto-imports) before parsing and compiling the project's source files.
 
By combining a handful of expressive directives — `project`, `class`, and `import` — framework authors can define rich, type-safe, domain-specific programming environments (games, web servers, data pipelines, etc.) that XGo source files compile into directly, without any boilerplate.
