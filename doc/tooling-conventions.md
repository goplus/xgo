# XGo tooling conventions

This document describes shared XGo conventions used by editors, language servers, documentation generators, and build
tools to interpret XGo source and Go packages used by XGo.

XGo uses `github.com/goplus/gogen` for Go package encodings and `github.com/goplus/mod` for module metadata. Dependency
versions are selected in [go.mod](../go.mod). See the [language specification](spec.md) for source syntax.

## Representations

Source analysis and package imports expose different representations of the same program:

| Representation | Contains |
| --- | --- |
| XGo source | Source spellings, declarations, comments, and implicit language constructs |
| XGo AST and type information | Source nodes, bindings, selected overloads, and instantiated types |
| Generated Go and Go export data | Go declarations and exported naming conventions |
| Initialized Go type information | Reconstructed overload groups, template methods, static members, and other XGo extensions |

A source declaration, its generated implementation, and a reconstructed callable can be different objects. Preserve
their relationship when implementing navigation, rename, documentation, and signature display.

## Modules and classfiles

### Module metadata

| Location | Convention | Tool behavior |
| --- | --- | --- |
| `go.mod`, `require` | `//xgo:class` | Load the dependency's class framework metadata |
| `go.mod`, `go` | Compiler comment such as `// llgo 1.0` or `// tinygo 0.32` | Read the selected compiler and version |
| Framework module root | `gox.mod` | Load project, work-class, and implicit-import metadata |
| `gox.mod` | `xgo` version directive | Read the XGo version requirement |

The historical forms `//gop:class`, `gop.mod`, and the `gop` version directive remain accepted. The module loader tries
`gop.mod` when reading `gox.mod` fails. A parse error in `gox.mod` does not trigger this fallback.

Use `github.com/goplus/mod` for module discovery and parsing. Its lax parser ignores unknown directives. Tools that
rewrite module files must preserve class annotations and metadata they do not interpret. XGo's
[tidy integration](../tool/tidy.go) retains class framework dependencies.

See [go.mod extensions](go.mod.md) and [gox.mod](gox.mod.md).

### Framework registry

Framework metadata affects parsing as well as compilation. Initialize the module's class registry with
`xgomod.Module.ImportClasses` before parsing framework source files. Query registrations through `ClassInfo` and
`LookupClass`.

| Directive | Meaning |
| --- | --- |
| `project` | Project entry pattern, project class, and framework packages |
| `class` | Work-class pattern, base type, and optional prototype |
| `import` | Implicit package import for the current project |
| `autolambda` | Commands whose trailing source block is parsed as a lambda |
| `pack` | Resource directory and index filename for project configuration packing |

The `class`, `import`, `autolambda`, and `pack` directives apply to the most recent preceding `project`.

| Option | Meaning |
| --- | --- |
| `project -flat` | Declares a project without work classes |
| `class -embed` | Embeds work instances in the project instance during assembly |
| `class -prefix=Prefix` | Prepends `Prefix` to generated work-class type names |

An `autolambda` entry such as `times(1)` records the number of arguments preceding the block. A `pack` entry has the
form `pack <directory> <indexfile>`. The directory must be relative and must not contain `..`. The index filename must
not contain path separators.

Built-in registrations include `.gsh` and `_test.gox`. Later registrations for the same class extension replace earlier
ones.

See [framework registration](classfile-spec.md#framework-registration-metadata) for project and work patterns, base
types, and implicit imports. [tool.PackProject](../tool/pack.go) merges project configuration in JSON or YAML without
writing output files.

Editor display aliases, callback classifications, and resource schemas are framework conventions beyond the class
registry's project/work types and implicit imports.

### Source file classification

| Source suffix | Classification |
| --- | --- |
| `.go` | Go source in a mixed Go/XGo package |
| `.xgo` | Ordinary XGo source |
| `.gop` | Historical spelling of `.xgo` |
| `.gox` | Ordinary or framework classfile |
| Registered class extension | Framework classfile |

Use [parser.ParseEntry](../parser/parser_xgo.go) with the module's `ClassInfo` callback to classify XGo source.
`modfile.ClassExt` extracts compound class extensions, including `_tag.gox` forms. An unregistered `.gox` file is an
ordinary classfile. Test classification includes `_test.xgo`, the historical `_test.gop`, and classfile test
registrations.

The parser records classfile properties on the XGo AST:

| AST field | Meaning |
| --- | --- |
| `File.IsClass` | Classfile, including an ordinary `.gox` file |
| `File.IsProj` | Framework project file |
| `File.IsNormalGox` | Ordinary `.gox` classfile |
| `File.ClassFields` | Instance field declaration block |
| `File.ShadowEntry` | Implicit entry function for top-level statements |

### Class types and generated members

Class type naming follows the [classfile specification](classfile-spec.md#type-naming). `cl.GetFileClassType` is a
deprecated helper for computing names from a file and registry metadata. In a flat project, non-project files contribute
fragments to the project instead of defining work-class types.

During compilation, `FuncDecl.IsClass` marks a receiver supplied by classfile lowering. `FuncDecl.Static` identifies a
static method. See the [classfile specification](classfile-spec.md) for implicit receivers, embedded bases, field
initialization, and name lookup.

The methods synthesized for a classfile depend on its field initializers, top-level statements, and framework base type:

- `XGo_Init`: [field initialization](classfile-spec.md#field-initialization)
- `MainEntry`, `Main`: [shadow entries](classfile-spec.md#shadow-entry) and
  [project assembly](classfile-spec.md#work-instance-assembly-for-project-main)
- `Classfname`, `Classclone`: [synthesized helper methods](classfile-spec.md#synthesized-helper-methods)

Package-level entry functions follow the [package main synthesis rules](classfile-spec.md#package-main-synthesis).

For package entry selection, non-test projects whose primary framework package declares `XGoTestClass` count as having
an entry even without top-level statements. The historical marker `GopTestClass` remains accepted.

A framework's primary package can export `Gop_sched` as a string constant containing one or two comma-separated
scheduler function names. [Classfile compilation](../cl/classfile.go) inserts the first zero-argument hook into nonempty
generated loop bodies and the second into empty ones. A single name supplies both hooks.

## Packages and imports

### Import paths

The compiler normalizes these source imports:

| Source import | Canonical import |
| --- | --- |
| Prefix `xgo/` | Prefix `github.com/goplus/xgo/` |
| Prefix `gop/` (historical) | Prefix `github.com/goplus/xgo/` |
| `c` | `github.com/goplus/lib/c` |
| Prefix `c/` | Prefix `github.com/goplus/lib/c/` |
| `py` | `github.com/goplus/lib/py` |
| Prefix `py/` | Prefix `github.com/goplus/lib/py/` |
| Prefix `cpp/` | Prefix `github.com/goplus/lib/cpp/` |

Preserve the source spelling for edits and the canonical path for package identity. See
[import path normalization](../cl/c.go).

### Package markers

A package-level `XGoPackage` constant marks a package for XGo import initialization:

```go
const XGoPackage = true
```

The marker can instead contain a comma-separated list of XGo package import paths needed to interpret exported types:

```go
const XGoPackage = "example.com/framework,example.com/library"
```

Detection uses the declaration name for both marker forms. The generator does not emit the marker automatically for
package `main`.

The historical marker `GopPackage` remains accepted. When both names exist, `XGoPackage` takes precedence.

### Import initialization

`gogen.InitXGoPackage` reconstructs XGo declarations from Go names, adding objects to package scopes and methods to
named types. Tools that load Go export data directly must initialize each marked package once before XGo member lookup
or signature inspection.

The helper reconstructs one package. The importer must also load and initialize dependencies listed in a string marker,
using the same package graph.

Gogen's import handling records initialization with the in-memory sentinel `__xgo_inited` in the package scope. The
sentinel is separate from the package's source marker.

## Go symbol encodings

### Overloads

| Go representation | XGo meaning |
| --- | --- |
| `Name__0`, `Name__1`, ... | Indexed implementations of the overload `Name` |
| `XGoo_Name` | String constant listing implementations of a package-level overload |
| `XGoo_Type_Method` | String constant listing implementations of an overload on `Type` |
| `XGoo__Type__Method` | Escaped form when the type or method name contains an underscore |

The index suffix is one character from `0123456789abcdefghijklmnopqrstuvwxyz`. Overload resolution tries candidates in
index or list order. Indexed named types use the same suffix convention.

An `XGoo_` string is a comma-separated list. A bare entry names a package-level function. A leading `.` names a method
of the owning type. An empty entry denotes the corresponding indexed implementation, such as `Name__2` for the third
entry. A package-level overload name containing underscores uses `XGoo__Name` to escape the name. After the `XGoo_`
prefix, split the name as described under [extended members](#extended-members).

An overload group and the implementation selected at a call site are distinct objects. An independently named function
remains a declaration even when it is referenced by an overload group.

See [overload encodings and operator names](overload.md#under-the-hood-how-xgo-encodes-overloading-in-go).

### Extended members

| Go name | XGo meaning | Interpretation |
| --- | --- | --- |
| `XGot_Type_Method` | Template receiver method | The first parameter of the underlying function supplies the receiver |
| `XGos_Type_Name` | Static member of a named type | Associates a package-level function, variable, or constant with the type |
| `XGox_Name` | Function with types passed as leading source arguments | Binds source type arguments before ordinary value arguments |
| `XGot__Type__XGox_Method` | Template method with leading type arguments | Combines receiver binding and type arguments |

After the `XGot_` or `XGos_` prefix, split an unescaped remainder at its first `_`. If the remainder starts with `_`,
remove that leading character and split at the first `__` instead, as in `XGot__Type_Name__Method_Name`.

Type-as-parameter calls can supply leading generic type arguments and infer the remaining ones from value arguments.

The historical prefixes `Gopt_`, `Gops_`, and `Gopx_` remain recognized for the corresponding member kinds.

See [template receiver methods](template-recv-method.md) and
[static methods in classfiles](classfile-spec.md#static-method-declaration).

### Reconstructed symbols

Reconstructed symbols can be represented by a synthetic signature with a parameter named `__xgo_overload_args__`. That
parameter carries extension information and is not a source parameter. The signature can also represent a static value
member, so it does not by itself imply that the source member is callable.

Use gogen's inspection helpers:

- `CheckSigFuncEx` inspects the extended type carried by a signature
- `CheckSigFuncExObjects` obtains underlying objects for overloads, template methods, and type-as-parameter functions
- `CheckFuncEx` returns the `TyFuncEx` extension carried by a signature
- `CheckOverloadNamed` identifies overloaded named types

For a static member, inspect `*gogen.TyStaticMember` and its `Obj()` result. `CheckSigFuncExObjects` does not unwrap
static members.

Inspect the returned extension kind when deriving source signatures. Overloads, template methods, static members, and
type-as-parameter functions have different receiver and argument bindings.

## Calls and parameters

### Aliases and implicit calls

| Source form | Lookup or binding |
| --- | --- |
| Package function alias, such as `math.sqrt` | A lowercase ASCII initial resolves the exported callable `Sqrt` |
| Instance method alias, such as `value.name` | Can resolve an exact method name or an alias formed by capitalizing a lowercase ASCII initial |
| Method alias `value._Name` | Can resolve the protocol method `XGo_Name` |
| Alias without parentheses in a value context | Can invoke a zero-parameter function or method implicitly |
| Method expression | Includes an explicit receiver argument |
| Method value or template receiver method call | Binds the receiver separately from source arguments |

Lowercase package aliases are accepted when `gogen.IsFunc` recognizes the exported object's type. Constants and ordinary
type names keep their declared spelling. Package alias lookup does not use the same exact-name lookup as instance method
access.

An implicit call's result type differs from the callable's signature. For example, `time.now` in a value context yields
a `time.Time`, while its callable has signature `func() time.Time`.

See [commands and function calls](fncall.md) and the lookup implementation in [cl/expr.go](../cl/expr.go).

### Parameter conventions

| Representation | Meaning |
| --- | --- |
| `ast.Field.Optional` | Optional parameter in XGo source |
| `__xgo_optional_Name` | Optional parameter encoded in an imported Go signature |
| `__xgo_autoclosure_Name` | Source expression wrapped as a closure argument |
| `__xgo_loop_Name` | Automatic lambda with loop control-flow handling |
| `__xgo_cond_Name` | Automatic lambda with conditional control-flow handling |

An autoclosure parameter must have an underlying type of `func() T`. Its source argument is an expression of type `T`,
which the compiler wraps in a closure for evaluation by the callee.

Gogen maintains optionality for in-memory parameters. Use `gogen.Package.NewParam` to construct them. Imported
parameters carry optionality in the name encoding above.

For automatic lambda blocks, the parser also needs the command's argument count from framework metadata. The parameter
names select loop or conditional control-flow handling after the block has been identified.

Argument-to-parameter mapping must account for:

- The selected overload and any inferred type arguments
- Bound receivers and explicit source type arguments
- Optional and variadic parameters, including `...`
- Tuple expansion and synthesized arguments
- Automatic lambdas and decorator callbacks

See [optional parameters](func-closure.md#optional-parameters) and [call argument compilation](../cl/expr.go).

### Keyword arguments

Keyword arguments are recorded separately in `ast.CallExpr.Kwargs`. The compiler combines them into a value argument
according to the target parameter. For variadic calls, this container precedes the variadic arguments.

| Target | Interpretation |
| --- | --- |
| String-keyed map | Keyword names become string keys |
| Struct or pointer to struct | Keyword names bind to fields, with exported-name lookup where required |
| Named interface | Keywords build a chain of methods returning that interface |

Interface keywords require a method call whose receiver is an identifier, such as `client`, or a single selector, such
as `obj.client`. The receiver must provide a factory named after the interface, such as `client.Options()`.

A known keyword calls the corresponding builder method with one argument. A lowercase ASCII initial is capitalized for
method lookup. An unknown keyword requires `Set(string, any) Self`. Each builder method returns the same named
interface.

See [keyword arguments](func-closure.md#keyword-arguments) and [keyword argument lowering](../cl/expr.go).

## Type and library protocols

These declarations supply the library operations used by XGo syntax and type checking. They may be authored in Go
libraries, so the prefix alone does not identify generated code.

### Operators, conversions, and constants

| Convention | Meaning |
| --- | --- |
| `XGo_Add`, `XGo_Sub`, and other operator names | Operator implementation, including overload groups |
| `T_Cast` | Explicit conversion to the named type `T` |
| `T_Init` | Implicit initialization or assignment conversion to `T` |
| `T_Default` | Type alias selecting the default type for an extended type |
| `T_IsUntyped` | Boolean constant enabling untyped behavior for an extended constant type |
| `T_Min`, `T_Max` | Bounds checked during integer constant conversions |
| `XGo_Rcast` | Reverse-cast support on the source type |

Type-associated declarations such as `T_Cast` and `T_Default` are looked up in `T`'s defining package. Ordinary Go
conversion and assignment rules are tried before `T_Cast` and `T_Init`, respectively. `T_Init` is distinct from the
instance method `XGo_Init` generated for class field initialization. Conversions for overloaded named types use
`XGox_T_Cast`.

See the [operator table](overload.md#overload-operators-xgo_-names-and-the-full-operator-table).

### Numeric units

A string constant named `XGou_T` in type `T`'s defining package supplies its unit table. Its value contains
comma-separated `unit=factor` pairs:

```go
type Distance int64

const XGou_Distance = "mm=1,cm=10,m=1000"
```

Unit literals in call arguments use the expected parameter type to select the unit table. The type must be a named type
or a type alias, and an applicable unit table is required. The unit suffix alone does not determine the type.

The compiler multiplies the literal value by the unit's factor. For a type alias, lookup checks the alias's own table
first, then the fully unaliased named type's table. `time.Duration` has a built-in table. Unit table entries do not
declare package variables or constants.

See [unit argument compilation](../cl/expr.go).

### Iteration

`XGo_Enum()` takes no arguments and returns one iterator. Supported iterator function signatures are:

| Iterator signature | Yielded values |
| --- | --- |
| `func(yield func() bool)` | None |
| `func(yield func(V) bool)` | Value of type `V` |
| `func(yield func(K, V) bool)` | Key of type `K` and value of type `V` |

The method can instead return a named iterator object with `Next() (V, bool)` or `Next() (K, V, bool)`.

The historical method name `Gop_Enum` remains accepted. This container protocol is separate from
[enum type declarations](enum.md). See [iteration](for.md) for source loop forms and binding rules.

### Dynamic lookup and queries

Dynamic queries use the following protocols:

| Source form | Protocol | Meaning |
| --- | --- | --- |
| `value.name` or `value."name"` | `XGo_Elem("name")` | Child lookup when ordinary member lookup fails |
| `value.$name` | `XGo_Attr("name")` | Attribute lookup |
| `value.*` | `XGo_Child()` | Direct children |
| `value.**.name` | `XGo_Any("name")` | Descendants with the given name |
| `value.**.*` | `XGo_Any("")` | All descendants |
| `value @ name` or `value @ "name"` | `XGo_Select("name")` | Select nodes with the given name |

Maps and empty interfaces can use direct keyed access for attributes instead of `XGo_Attr`.

Predicate filters such as `value @ (condition)` iterate candidate node sets through `XGo_Enum()` and extract matching
nodes through `XGo_first()`. The latter returns a node and an error. The implicit `self` binding denotes the current
candidate node set. See [DQL protocols](../dql/README.md#implementing-a-nodeset).

Class receivers can additionally supply:

| Protocol | Meaning |
| --- | --- |
| `XGo_Exec` | Fallback for unresolved command calls |
| `XGo_Env` | Environment lookup for environment expressions and interpolated identifiers |

The historical names `Gop_Exec` and `Gop_Env` remain accepted. See [selector and command compilation](../cl/expr.go).

### Builtins

XGo builtins and extension methods on builtin Go types are supplied by the compiler's builtin configuration. Some
objects have an empty package path or come from support libraries. They are not all entries in the Go universe or
ordinary Go method sets.

See [builtins](builtin.md) and [builtin registration](../cl/builtin.go).

## Source analysis

### AST and positions

Use the [XGo AST](../ast/ast_xgo.go) and [parser](../parser/interface.go) for XGo source. XGo nodes include overload
declarations, enum types, number units, decorators, keyword arguments, and implicit class entries in addition to Go-like
nodes.

- `File.Code` holds the original source bytes
- `File.NoPkgDecl` distinguishes an omitted package clause from an explicit clause
- `token.Pos` and `token.FileSet` use Go's position representation, but XGo's `token.Token` is a distinct token type
- Source ranges use an inclusive `Pos()` and an exclusive `End()`
- File offsets are byte offsets. Convert them to the client's negotiated position encoding at the language-server
  boundary
- Distinguish physical file positions from positions adjusted by `//line` directives
- Generated objects can have `token.NoPos`, and generated identifiers can reuse positions without matching the source
  spelling

Check source text and declaration mappings before producing edits. Neither a valid position nor `Ident.Implicit()` alone
proves that an identifier was written in source. Retain the file set with the syntax trees and type information it
describes.

Compilation can modify the AST, including implicit receiver insertion and overload rewriting. Tools that need the
original syntax should retain a separate parsed tree or source snapshot.

### Type information

Use [typesutil.NewChecker](../x/typesutil/check.go) for mixed Go and XGo packages, or `cl.NewPackage` with a
[typesutil recorder](../x/typesutil/gopinfo.go). Allocate each `typesutil.Info` map needed by the consumer. Unallocated
maps are not populated, and errors can leave partial information.

| Information | Query |
| --- | --- |
| Declaration and reference bindings | `Defs`, `Uses`, and `ObjectOf` |
| Expression types | `Types` and `TypeOf` |
| Constant expression values | `Types[expr].Value` |
| Original overload group | `Overloads` and `OverloadOf` |
| Selected callable at a use | `ObjectOf`, together with the overload group |
| Generic type arguments and instantiated types | `Instances` |
| Promoted members and receiver selection | `Selections` |
| Lexical scopes | `Scopes` |
| Implicit declarations | `Implicits` |

For an overloaded call, the selected implementation can differ from the declaration stored in `Overloads`. For a bare
callable identifier, `TypeOf` can describe the callable rather than an implicit call's result. `OverloadOf` returns a
group only when `CheckSigFuncExObjects` yields more than one object.

`typesutil.Config` supplies the shared package, file set, module, and analysis options, including `IgnoreFuncBodies` and
`UpdateGoTypesOverload`. `typesutil.Error` carries `Fset`, `Pos`, `End`, `Msg`, `Code`, and `Soft` for diagnostics. See
the [checker API](../x/typesutil/check.go) and [recording and error definitions](../x/typesutil/gopinfo.go).

### Source comments

Enable `parser.ParseComments` when parsing source to collect comments. Read individual `Comment.Text` entries to inspect
directive text. `CommentGroup.Text()` returns normalized documentation text and omits directive-style comments.

Source comments are not included in Go export data.

## Generated Go

[Go generation](../tool/gengo.go) uses these output families:

| Output | Purpose |
| --- | --- |
| `xgo_autogen.go` | Package output |
| `xgo_autogen_test.go` | Tests in the package |
| `xgo_autogen2_test.go` | External test package |
| `<stem>_autogen.go` | Single-file generation mode |
| `xgo_autogen_<source-filename>.go` | Default file-specific output selected by `GenGoFiles` for one file in a directory with multiple XGo files |

Generated Go contains declarations derived from XGo source, so loading both as independent sources can create duplicate
declarations.

The mixed checker excludes Go files with the `xgo_autogen` prefix when XGo files are present. Directory parsing also
skips the historical `gop_autogen` prefix. Callers supplying Go ASTs must exclude historical output and other generated
copies that the mixed checker does not filter.

Generated output can contain `//line` mappings and the `Code generated` marker. Use original source locations for
diagnostics and edits. `cl.Config.NoFileLine` disables generated line mappings.
