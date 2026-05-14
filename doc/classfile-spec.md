# The XGo classfile specification

This document defines the syntax and semantics of XGo classfiles.

A classfile is a source file that is compiled as a generated class type plus a set of generated methods and helper glue.

There are two kinds of classfiles:
- A normal classfile, which is an otherwise unregistered `.gox` file
- A framework classfile, which is a file whose class extension is registered by the classfile registry

## Terms

The following terms are used throughout this document:
- Class extension: the normalized classfile suffix used for framework lookup, e.g., `_app.gox` and `.gsh`
- Class file stem: the filename without the class extension
- Classfile project: the source tree rooted at the analyzed package directory and interpreted by the classfile mechanism
- Project root directory: the root directory of one classfile project
- Pack root: the directory named by one `pack` directive and resolved relative to one project root directory
- Pack document: the logical merged configuration object derived from one pack root and its declared index filename
- Project file kind: a framework file kind declared by one `project` directive and used for project classfiles
- Work file kind: a framework file kind declared by one `class` directive and used for work classfiles
- Project classfile: the framework file that represents the project-level class
- Work classfile: a framework file that represents a non-project class within the same framework
- Class type: the generated named type for a classfile
- Base class: an embedded framework type declared by classfile metadata
- Work prototype type: an exported type declared by classfile metadata and associated with one work file kind
- Field declaration block: the unique top-level `var` declaration that is interpreted as class fields rather than
  package variables
- Shadow entry: the synthetic function created from top-level statements

## File classification

### Class extension extraction

Class extension extraction operates on the base filename.

For a filename whose last path extension is not `.gox`, the class extension is the last path extension.

For a filename whose last path extension is `.gox`, the class extension is:
- The suffix beginning at the last underscore, if the underscore occurs before `.gox` and is not the first character of
  the filename
- Otherwise `.gox`

Examples:
- `Rect.gox` has class extension `.gox`
- `main_app.gox` has class extension `_app.gox`
- `main.gsh` has class extension `.gsh`

### Source-file kinds

A source file is treated as a classfile if and only if one of the following is true:
- Its filename is recognized by the classfile registry
- Its filename ends in `.gox` and its normalized class extension is not recognized by the registry

If the second rule applies, the file is a normal classfile.

If the first rule applies, the file is a framework classfile.

### Project and work classification

Each recognized class extension belongs to exactly one framework registration.

A recognized file is a project classfile if the registration marks it as a project file for the pair
`(class extension, filename)`. Otherwise it is a work classfile.

A recognized file is classified as follows:
- If a project extension and a work extension are different, every file whose normalized class extension equals the
  project extension is a project file
- If a project extension and a work extension are the same, only the filename `main` plus that extension is a project
  file and all other files with the same extension are work files

Examples:
- Under metadata where the project extension and work extension are both `_case.gox`, `main_case.gox` is a project file
  and `foo_case.gox` is a work file
- Under metadata where `_app.gox` is the project extension and `_cmd.gox` is the work extension, `demo_app.gox` is a
  project file and `list_cmd.gox` is a work file

## Source form

### Package clause

A classfile may omit the package clause.

If a classfile omits the package clause, it is compiled as if it declared `package main`.

### Imports

Import declarations follow ordinary XGo source-file rules:
- Import declarations must appear before all non-import declarations and top-level statements
- Ordinary source-level imports remain ordinary package imports after lowering

### Top-level structure

Ignoring comments, a classfile has the following top-level structure:

```ebnf
Classfile = [ PackageClause ] { ImportDecl } { ClassDecl } [ TopLevelStmtList ] .
ClassDecl = ConstDecl | TypeDecl | VarDecl | FuncDecl .
TopLevelStmtList = StatementList .
```

The optional `TopLevelStmtList` must be the final top-level construct in the file. After the first top-level statement
is encountered, the remainder of the file is parsed using ordinary function-body statement-list rules as the body of the
shadow entry.

Accordingly, constructs that are valid as statements inside a function body, including local declaration statements such
as `var x = 1`, become part of the shadow entry rather than remaining top-level declarations.

### Field declaration block

For the purposes of class lowering, at most one top-level `var` declaration is interpreted as the field declaration
block.

The field declaration block is identified semantically as follows:
- The declaration must be a top-level `var` declaration in the file's declaration list
- Every preceding top-level declaration, if any, must be an `import`, `const`, or `type` declaration
- It must be the first top-level `var` declaration satisfying those conditions

That declaration does not introduce package variables. Instead, it declares fields of the generated class type.

All other top-level `var` declarations follow ordinary variable-declaration rules. If they occur before the shadow entry
begins, they are package-level variables. If they occur after the shadow entry begins, they are local declaration
statements inside that entry method.

This specification defines embedded-field syntax only for the field declaration block.

### Field declaration syntax

Within the field declaration block, each spec has one of the following forms:

```ebnf
FieldSpec      = EmbeddedField [ Tag ]
               | IdentifierList [ Type ] [ "=" ExpressionList ] [ Tag ] .
EmbeddedField  = [ "*" ] TypeName
               | [ "*" ] PackageName "." identifier .
Tag            = string_lit .
```

The following special rule applies only inside the field declaration block:
- A spec consisting of exactly one identifier, with no explicit type and no initializer, declares an embedded field
  whose type is that identifier
- This is the only case in which `IdentifierList` may appear without either a `Type` or an `=` initializer

For any other `FieldSpec`, ordinary `var`-declaration validity rules apply. In particular, at least a `Type` or an `=`
initializer must be present.

Examples:

```xgo
var (
  Width, Height int
  *bytes.Buffer "buffer"
  BaseClass
)
```

Within the field declaration block:
- An embedded-field spec must not have an initializer
- A tag may appear on both ordinary fields and embedded fields
- The quoted-string tag syntax is accepted exactly as written in source, and if the string does not contain an explicit
  tag key, the generated Go tag becomes `_:"..."`

## Class type generation

### Type naming

For every classfile, the compiler generates one named class type.

The initial class type name is derived from the class file stem.

The class file stem is normalized as follows:
- `:` is removed
- `#` is removed
- `-` is replaced with `_`
- `.` is replaced with `_`

Examples:
- `Rect.gox` produces the initial type name `Rect`
- `get_p_#id_app.gox` has class file stem `get_p_#id` and normalized type name `get_p_id`

Framework metadata may further transform the type name:
- A project classfile whose class file stem is `main`, or a framework that has no explicit project classfile, uses the
  name of the base type named by the project base class as its default class type name
- A non-empty `-prefix=` on a work file kind is prepended to the normalized class file stem of a work classfile
- If neither of the previous rules applies and the class type name would otherwise equal one of the reserved names
  `init`, `main`, `go`, `goto`, `type`, `var`, `import`, `package`, `interface`, `struct`, `const`, `func`, `map`,
  `chan`, `for`, `if`, `else`, `switch`, `case`, `select`, `defer`, `range`, `return`, `break`, `continue`,
  `fallthrough`, or `default`, an underscore is prepended

The generated class type must be unique within the package after all such normalization.

### Underlying type

The underlying type of every class type is a struct type.

For a normal classfile, the struct fields are the user fields declared in the field declaration block, in source order.

For a framework classfile, framework-added fields precede user fields.

### Framework-added fields

Framework-added fields are inserted in the following order:
- For a project classfile, the embedded project base class
- For a project classfile, each embedded work class field requested by the `-embed` flag, in work file kind declaration
  order and lexicographic source-file path order
- For a work classfile, the embedded work base class
- For a work classfile, an embedded pointer to the project class type, if a project class type exists and its field name
  does not conflict with an already-present field name

User fields from the field declaration block are appended after all framework-added fields.

It is an error for two generated fields of the same class type to have the same field name.

## Functions and methods

### Implicit receiver rewriting

In a classfile, every top-level function declaration without an explicit receiver is rewritten as a method on the
generated class type.

The injected receiver is:
- Named `this`
- Of type `*T`, where `T` is the generated class type

This rewriting applies to ordinary function declarations, including a function named `init`.

As a consequence, `func init()` inside a classfile declares a method named `init`. It is not a package initialization
function.

The same rewriting rule also applies to `func main()`. Inside a classfile, `func main()` declares a method named `main`.
It is not a package-level entry function.

### Explicit receiver declarations

A top-level function declaration that already has an explicit receiver is not rewritten by the classfile mechanism.

### Static method declaration

The classfile parser also accepts the following classfile-only `FuncDecl` form:

```ebnf
StaticMethodDecl = "func" "." identifier Signature [ FunctionBody ] .
```

This form declares a static method associated with the generated class type. Its further lowering follows the ordinary
XGo static-method machinery.

### Shadow entry

Top-level statements are not compiled as package-level statements.

Instead, they are wrapped into a synthetic function with an initially empty parameter list, called the shadow entry.
After classfile lowering, the shadow entry is renamed as follows:
- `MainEntry` for a project classfile
- `Main` for any other classfile

If a framework classfile has no explicit top-level statement sequence, the compiler still synthesizes an empty shadow
entry with the same name. This ensures that framework entry methods always exist.

For a normal classfile, no empty shadow entry is synthesized. A normal classfile without top-level statements therefore
has no synthetic `Main` method.

### Base-entry forwarding

If a synthetic `Main` or `MainEntry` method is created for a framework classfile, and the embedded base class declares a
method with the same name, the synthetic method adopts that base method's parameter list and result list, with the
classfile receiver `*T` replacing the base receiver.

The generated body forwards the incoming arguments to the embedded base method before executing any user-written
top-level statements.

### Execution order inside a synthetic entry method

If a synthetic `Main` or `MainEntry` method is generated, its body executes in the following order:
1. Call `this.XGo_Init()` if the generated class type directly defines `XGo_Init`
2. Call the embedded base method of the same name, if one exists
3. Execute the user-written top-level statements wrapped by the shadow entry

This order is fixed.

If the adopted method signature has result parameters, the resulting body is checked under ordinary Go control-flow
rules for that signature after lowering.

## Field initialization

### `XGo_Init` generation

If at least one field in the field declaration block has an initializer, the compiler generates a method:

```xgo
func (this *T) XGo_Init() *T
```

The generated method assigns all field initializers in field-declaration order and then returns `this`.

If the field declaration block contains no initializers, no `XGo_Init` method is generated.

### What `XGo_Init` does not do

Field initializers are not attached to the struct type itself.

In particular, field initializers are not executed automatically by:
- `new(T)`
- `&T{...}`
- `T{...}`
- A user-declared method named `Main`
- A user-declared method named `MainEntry`
- Any ordinary method call other than the synthetic entry methods described above

If a class type has an `XGo_Init` method but no synthetic entry method invokes it, the method exists but is never called
automatically.

## Name resolution inside class methods

Within a class method body, unqualified identifier resolution differs from ordinary package-level XGo code.

A bare identifier is resolved in the following order:
1. Lexically local declarations
2. Members of `this`
3. Package-scope declarations in the current package
4. Ordinary XGo function-alias rewrites
5. Implicit framework-package exports from the framework's package lookup set
6. Universe-scope predeclared identifiers

Two additional rules apply:
- A source-level imported package name, or an auto-imported package name from module metadata, participates when the
  identifier is used as the left side of a selector expression
- If the same implicit framework export is found in more than one framework lookup package, compilation fails with a
  name-conflict error

Implicit framework-package export lookup uses the ordinary XGo package-member alias rules. In particular, an exported
function may be referenced through its lowercase alias.

The injected receiver name `this` is an ordinary method receiver identifier and may be referenced explicitly.

## Framework registration metadata

### Metadata sources

The classfile registry is built from three sources:
- Built-in framework registrations defined by the toolchain
- The current module's `gox.mod`
- Dependency registrations imported from `go.mod` `require` lines carrying the comment `//xgo:class`

### Built-in registrations

The toolchain defines the following built-in framework registrations and no others:

```text
project .gsh App github.com/qiniu/x/gsh math

project _test.gox App github.com/goplus/xgo/test testing
class _test.gox Case
```

A built-in registration is active without any module declaration.

Built-in registrations participate in file classification, lowering, and package assembly exactly as if they were loaded
from module metadata.

### Recognized classfile directives

The classfile loader recognizes the following module directives:

```ebnf
ProjectDirective = "project" [ ProjectExt ExportedName ] PackagePath { PackagePath } .
ClassDirective   = "class" { ClassDirectiveFlag } WorkExt ExportedName [ ExportedName ] .
ImportDirective  = "import" [ ImportName ] PackagePath .
PackDirective    = "pack" RelativeDirectoryPath PackIndexFile .
ClassDirectiveFlag = "-embed" | "-prefix=" string_without_space .
RelativeDirectoryPath = string_without_space .
PackIndexFile    = string_without_space .
```

`RelativeDirectoryPath` is one relative directory path token with no spaces.

`PackIndexFile` is one plain file name token with no spaces.

Every `class`, `import`, or `pack` directive belongs to the most recent preceding `project` directive.

A project group consists of one `project` directive together with all `class`, `import`, and `pack` directives that
belong to it.

The first package path of a `project` directive is the framework package used to resolve the project base class, each
work base class, and each work prototype type named by that project group.

Any additional package paths participate in implicit framework-package export lookup but are not searched for those
symbols.

### Extension forms and normalization

The extension token accepted by `project` and `class` directives has two forms:
- `_[class].gox`
- `.[class]`

Both forms are part of the classfile mechanism. Neither form is a compatibility alias of the other.

For newly defined framework registrations, `_[class].gox` is the recommended form.

The built-in registrations defined above remain valid as written.

Auxiliary tool behavior for arbitrary non-`.gox` class extensions is implementation-defined.

The textual extension token accepted by `project` and `class` directives may be written with a leading `*` and, for
projects, may also be written with a leading `main`.

The loader normalizes those forms by stripping the leading `*` or leading `main` and storing only the resulting class
extension for matching.

Examples:
- `*_cmd.gox` normalizes to `_cmd.gox`
- `main_app.gox` normalizes to `_app.gox`

### Meaning of class metadata

For each `project` directive:
- If `ProjectExt` and `ExportedName` are present, the directive defines a project file kind and names the project base
  class
- If `ProjectExt` and `ExportedName` are omitted, the directive defines no project file kind or project base class and
  still defines the package lookup set and the project group to which subsequent `class`, `import`, and `pack`
  directives belong
- When a project base class is named, the first package path is the package from which that symbol is resolved
- A project base-class name may be written with a leading `*`, in which case the generated project class embeds a
  pointer to the named base type rather than the base type itself

For each `class` directive:
- The exported symbol names the work base class and is resolved from the framework package of the containing project
  group
- The optional final exported symbol is the work prototype type
- If a project group declares more than one work file kind, every work file kind in that group must declare a work
  prototype type
- When a work prototype type is named, the work prototype type symbol is resolved from the framework package of the
  containing project group
- `-embed` causes the project class type to embed a field for each generated work class instance of that work file kind
- `-prefix=` prepends the given string to every generated work class type name

For each `import` directive:
- The imported package becomes available to classfiles as an auto-imported package name
- If no explicit import name is supplied, the package's declared package name is used
- If multiple `import` directives in the same project group resolve to the same auto-import name, the last directive
  wins

For each `pack` directive:
- The project group may declare at most one `pack` directive
- The directory path is resolved relative to the project root directory of the analyzed classfile project
- The directory path must not contain any `..` path component
- The index filename determines the root configuration filename of the pack root and the child configuration filename
  expected at each descendant directory level
- The index filename must not contain `/` or `\`
- The index filename must end in one of `.json`, `.yml`, or `.yaml`

## Project and work class assembly

A test framework registration is a framework registration whose project extension has the suffix `test.gox`.

All other framework registrations are non-test framework registrations.

### Default project synthesis

For each framework registration, a package may contain at most one explicit project classfile.

It is an error for a package to contain more than one explicit project classfile for the same framework registration.

If a framework registration provides a project base class but the package contains no explicit project classfile for
that framework, the compiler still synthesizes a default project class type.

The synthesized project class has no source file of its own. Its type name is derived by the project type-naming rules
described earlier.

### Work instance assembly for project `Main`

For every non-test framework registration that provides a project base class, the compiler generates a project method
named `Main` on the project class type. The project base class is therefore required to provide a method named `Main`.
The generated project method constructs work class instances and forwards them to the embedded project base-class method
`Main`.

The grouping rule is:
- If the framework has exactly one work file kind and that `Main` parameter is variadic, all work files of that kind are
  passed as variadic arguments
- Otherwise, work files are grouped by their declared work prototype type and passed as slices in `Main` parameter order

The project `Main` method constructs one fresh work class instance for each work file in the package. When `-embed` is
present on the corresponding `class` directive, the freshly created work instance is also assigned into the
corresponding embedded field on the project instance before the project `Main` call.

## Synthesized helper methods

The compiler may synthesize additional work class methods when the declared work prototype type requires them.

### `Classfname`

If the work prototype type contains a method named `Classfname`, the compiler generates:

```xgo
func (this *T) Classfname() string
```

The method returns the class file stem.

Examples:
- `hello_tool.gox` yields `hello`
- `get_p_#id_app.gox` yields `get_p_#id`

### `Classclone`

If the work prototype type contains a method named `Classclone`, the compiler generates a shallow-clone method named
`Classclone` with no parameters other than the receiver. Its result list is adopted from the work prototype type's
`Classclone` declaration.

The generated implementation copies `*this` by value into a temporary variable and returns the address of that temporary
value.

## Package `main` synthesis

When compiling a package named `main`, the compiler checks whether an explicit package-level `main` function already
exists.

If one exists, no class-based package `main` is synthesized.

If none exists, the compiler selects a class entrypoint as follows:
1. It considers only non-test framework registrations
2. Among framework project groups, it prefers a unique project group whose explicit project classfile has a shadow entry
3. If no such group exists, it prefers a unique remaining project group, including one that is represented only by a
   synthesized default project class
4. If no project group is selected, it selects the unique normal classfile that has a shadow entry, if exactly one
   exists

If this process selects a class type `T`, the compiler generates:

```go
func main() { new(T).Main() }
```

If no class type is selected, the compiler generates an empty `main` function unless automatic main generation is
disabled in compiler configuration.

## Pack roots and pack documents

One active project group may derive zero or one pack document from one classfile project.

If one active project group declares one `pack` directive, its pack root is the resolved directory named by that
directive.

Active project groups do not share pack roots or pack documents. Each active project group derives its own pack
document independently from its own `pack` directive, if any.

For one active project group whose `pack` directive resolves to one pack root `R` and one index filename `F`, the
corresponding pack document is one logical object tree derived as follows:
- `R/F` is parsed as one object value
- each descendant directory of `R` that contains `F` contributes one child object
- each such child object is parsed from that descendant `F` and merged into the root object at the relative
  directory path from `R`, creating intermediate objects as needed
- directories that do not contain `F` contribute no object of their own
- files other than contributing `F` files are outside the standardized pack-document model

If the project group declares no `pack` directive, no standardized pack document is derived for that project group.

If `R/F` does not exist, does not parse as one object value, or if one merge would overwrite an existing key at the
same object level, no standardized pack document is derived for that project group.

The project files remain the source of truth. Each pack document is derived from one analyzed project state and one
active project group.

An implementation may materialize one pack document as one generated sibling file of its root configuration file named:
- `index_pack.json` if `F` ends in `.json`
- `index_pack.yml` if `F` ends in `.yml`
- `index_pack.yaml` if `F` ends in `.yaml`

Its enablement mechanism is implementation-defined.

## Compatibility

The following compatibility aliases are also accepted:
- `gop.mod` is a legacy alias of `gox.mod` and is read only when `gox.mod` is absent
- `//gop:class` is a legacy alias of `//xgo:class` on dependency `go.mod` `require` lines

## Relationship to ordinary struct and package semantics

After lowering, a class type is an ordinary named struct type and its lowered methods are ordinary Go methods or
ordinary static-method helpers.

Accordingly:
- Composite literals for class types follow ordinary struct literal rules after lowering
- Ordinary package variables, constants, types, and explicit methods declared in the same package follow ordinary
  package semantics
- Package initialization order for ordinary package variables is unchanged

The classfile mechanism therefore adds a source-level lowering rule and preserves the ordinary runtime object model
produced by the lowered Go code.
