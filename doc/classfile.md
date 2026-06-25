XGo Class Framework
=====

> *"Don't define a language for specific domain. Abstract domain knowledge for it."*
> — XGo Design Philosophy

---

## What Is a Class Framework?

A **class framework** is XGo's mechanism for abstracting domain knowledge into reusable application skeletons. It is as fundamental to XGo as `interface` is to Go.

Rather than defining a dedicated DSL for every domain, XGo promotes **Specific Domain Friendliness (SDF)**: write the scaffolding once in plain Go, then let application developers fill in lightweight XGo source files — called **classfiles** — that contain only their domain-specific logic. The XGo compiler handles all the wiring automatically.

A class framework consists of two roles:

- A **project class** — the singleton root of the application. There is exactly one per project.
- One or more **work class** kinds — the units of work the application is composed of. A project typically has many instances of these.

---

## Classfiles

A **classfile** is an XGo source file that implicitly defines a class. For example, `Rect.gox`:

```go
var (
    Width, Height int
)

func Area() int {
    return Width * Height
}
```

This is equivalent to the following Go code:

```go
type Rect struct {
    Width, Height int
}

func (this *Rect) Area() int {
    return this.Width * this.Height
}
```

The value of this form is ergonomics: developers — especially beginners — can define new types using the sequential-programming syntax they already know. Variables are variables; functions are functions. No `type` declarations, no receivers.

---

## The `gox.mod` File

Every class framework ships a `gox.mod` file that tells the XGo compiler how to interpret classfiles:

```
project <filePattern> <BaseType> <importPath> [extraImports...]
class   [flags] <filePattern> <BaseType> [ProtocolType]
```

- The `project` directive registers the project classfile pattern and its base type.
- The `class` directive (repeatable) registers work classfile patterns. When a framework has **multiple work class kinds**, each `class` directive carries a `ProtocolType` argument — an interface the compiler uses to route instances to the correct parameter of the Template Recv Method.

**Flags:**

| Flag | Effect |
|---|---|
| `-embed` | Auto-declare each work class instance as a named field on the project class struct |
| `-prefix=<P>` | Prepend `P` to the generated type name of every work class in this kind |

---

## What the Compiler Generates

Given the `gox.mod` declarations and the user's classfiles, the XGo compiler automatically produces:

### 1. Struct types

Each project classfile becomes a struct embedding its declared `BaseType`. Each work classfile becomes a struct embedding its own `BaseType` plus a pointer back to the project class instance. If `-embed` is set, the project struct also gets one named field per work class instance.

### 2. Entry functions

Top-level code in the **project classfile** compiles into `MainEntry()`. Top-level code in each **work classfile** compiles into `Main()`.

If the base type itself implements the entry function, the compiler automatically inserts a call to it before the user code runs. The `ProtocolType` interface serves a separate purpose: it defines the expected `Main` signature so the compiler can generate the correct method prototype.

### 3. A synthesized project `Main`

The compiler synthesizes a `Main` method on the project class that:

1. Instantiates all work class objects.
2. Initialises the `-embed` fields (if applicable).
3. Calls the framework's **Template Recv Method**, passing the project instance and all work class instances.

### 4. A global entry point

```go
func main() {
    new(ProjectClass).Main()
}
```

---

## The Template Recv Method

The **[Template Recv Method](template-recv-method.md)** is the hook a framework author writes to receive control after the compiler has assembled everything. Its name follows the convention `XGot_<BaseType>_Main` (historically also `Gopt_<BaseType>_Main`).

Its signature encodes the framework's structural contract:

```go
// Single work class kind — variadic
func XGot_Game_Main(app Gamer, sprites ...Sprite)

// Multiple work class kinds — one typed slice per kind
func Gopt_MCPApp_Main(app iAppProto, resources []ResourceProto, tools []ToolProto, prompts []PromptProto)
```

The compiler reads this signature to know how many kinds of work classes exist, and how to group instances when calling the method. The framework implementation of this function handles the rest: setting up the runtime, loading resources, calling each work class's `Main`, and so on.

---

## Framework Examples

### spx — 2D Game Engine

`github.com/goplus/spx`

spx is the earliest class framework in the XGo ecosystem, purpose-built for STEM education.

#### `gox.mod`

```go.mod
xgo 1.7

project main.spx Game github.com/goplus/spx/v2 math

class -embed *.spx SpriteImpl
```

This declares:

- `main.spx` is the project classfile; its class automatically embeds `spx.Game`. Top-level code in `main.spx` becomes the body of `MainEntry`.
- All other `*.spx` files are work classfiles; each embeds `spx.SpriteImpl`. Top-level code becomes the body of `Main`. The `-embed` flag causes each work class instance to be declared as a named field on the project class.

#### Example project

A "Aircraft War" game might have:

```
main.spx        // game loop and setup
Bullet.spx      // bullet behaviour
Enemy.spx       // enemy behaviour
MyAircraft.spx  // player behaviour
```

#### Generated Go code

```go
// From main.spx — project class

type Game struct {
    spx.Game           // embedded base type
    Bullet     *Bullet // auto-generated by -embed
    Enemy      *Enemy
    MyAircraft *MyAircraft
    ... // user defined members in main.spx
}

func (this *Game) MainEntry() {
    ... // user code from main.spx
}

// Synthesized by the XGo compiler:
func (this *Game) Main() {
    bullet     := new(Bullet)
    enemy      := new(Enemy)
    myAircraft := new(MyAircraft)
    // initialise -embed fields
    this.Bullet     = bullet
    this.Enemy      = enemy
    this.MyAircraft = myAircraft
    // call the Template Recv Method
    spx.XGot_Game_Main(this, bullet, enemy, myAircraft)
}

// From Bullet.spx — work class

type Bullet struct {
    spx.SpriteImpl // embedded base type
    *Game          // pointer back to project instance
    ...            // user defined members in Bullet.spx
}

func (this *Bullet) Main() {
    ... // user code from Bullet.spx
}

// Enemy and MyAircraft follow the same pattern.

// Global entry point

func main() {
    new(Game).Main()
}
```

#### Template Recv Method

```go
func XGot_Game_Main(game Gamer, sprites ...Sprite)
```

`Gamer` is satisfied by `*Game`; `Sprite` is satisfied by every work class. The engine loads each sprite's resources and calls its `Main` to run its behaviour loop.

---

### cobra — CLI Tool Framework

`github.com/goplus/cobra`

#### `gox.mod`

```go.mod
xgo 1.7

project *_app.gox App github.com/goplus/cobra/xcmd

class -prefix=Cmd_ *_cmd.gox Command
```

This declares:

- `*_app.gox` is the project classfile pattern; the class embeds `xcmd.App`.
- `*_cmd.gox` files are work classfiles; each embeds `xcmd.Command`. The `-prefix=Cmd_` flag prepends `Cmd_` to each generated type name. For example, `run_cmd.gox` produces type `Cmd_run` rather than `run`, avoiding collisions with common identifiers.

#### Example project

```
myapp_app.gox   // application entry
run_cmd.gox     // "run" subcommand
build_cmd.gox   // "build" subcommand
```

#### Custom `Main` signature

The `xcmd.Command` base type implements `iCommandProto`:

```go
type iCommandProto interface {
    ...
    Main(fname string)
}
```

By defining this interface, the framework tells the compiler that every work class `Main` must accept a `string` argument. The compiler generates accordingly, and also inserts a call to the base class `Main` first, since `xcmd.Command` itself implements `Main`:

```go
type Cmd_run struct {
    xcmd.Command
    *xcmd.App
}

func (this *Cmd_run) Main(_xgo_arg0 string) {
    this.Command.Main(_xgo_arg0) // base class call, auto-inserted by compiler
    ...                          // user code from run_cmd.gox
}
```

The same mechanism works for the project class: if the framework defines a custom `MainEntry` signature via a protocol interface, the compiler synthesises the base class call there too.

#### Template Recv Method

```go
func XGot_App_Main(app iAppProto, cmds ...iCommandProto)
```

`iAppProto` is satisfied by the project class; `iCommandProto` is satisfied by every `Cmd_*` work class. The framework implementation registers each command with the underlying `cobra.Command` infrastructure and launches the CLI.

---

### yap — Web Framework

`github.com/goplus/yap`

YAP uses filenames to define routes: `get.yap` handles `GET /`; `get_p_#id.yap` handles `GET /p/:id`. A minimal web server is a single file:

```go
// get.yap
html `<html><body>Hello, YAP!</body></html>`
```

#### `gox.mod`

```go.mod
xgo 1.7

project main.yap AppV2 github.com/goplus/yap

class *.yap Handler
```

#### Custom `Main` signature

`iHandlerProto` declares:

```go
type iHandlerProto interface {
    Main(ctx *Context)
}
```

Every handler's `Main` is generated with a `*Context` parameter, giving each handler direct access to the HTTP request and response. As with cobra, the compiler inserts a call to the base class `Main` automatically if `yap.Handler` implements it.

#### Template Recv Method

```go
func XGot_AppV2_Main(app AppType, handlers ...iHandlerProto)
```

---

### mcp — MCP Server Framework

`github.com/goplus/mcp`

The MCP framework is an example of a framework with **multiple distinct work class kinds**.

#### `gox.mod`

```go.mod
xgo 1.7

project *_mcp.gox MCPApp github.com/goplus/mcp/server

class *_res.gox    ResourceApp ResourceProto
class *_tool.gox   ToolApp     ToolProto
class *_prompt.gox PromptApp   PromptProto
```

| File suffix | Base type | Protocol |
|---|---|---|
| `*_res.gox` | `server.ResourceApp` | `ResourceProto` |
| `*_tool.gox` | `server.ToolApp` | `ToolProto` |
| `*_prompt.gox` | `server.PromptApp` | `PromptProto` |

#### Example project structure

```
MyServer_mcp.gox    // MCP server entry
Files_res.gox       // resource: file listing
Search_tool.gox     // tool: search
Summarize_tool.gox  // tool: summarize
Greet_prompt.gox    // prompt: greeting template
```

#### Generated Go code (abbreviated)

```go
// From myserver_mcp.gox — project class

type MyServer struct {
    server.MCPApp
}

func (this *MyServer) MainEntry() {
    ... // user code from myserver_mcp.gox
}

// Synthesized by the XGo compiler:
func (this *MyServer) Main() {
    filesRes      := new(Files)
    searchTool    := new(Search)
    summarizeTool := new(Summarize)
    greetPrompt   := new(Greet)

    server.Gopt_MCPApp_Main(this,
        []server.ResourceProto{filesRes},
        []server.ToolProto{searchTool, summarizeTool},
        []server.PromptProto{greetPrompt},
    )
}

// From search_tool.gox — work class

type Search struct {
    server.ToolApp
    *MyServer
}

func (this *Search) Main() {
    ... // user code from search_tool.gox
}

// Resources and prompts follow the same pattern with their respective base types.
```

#### Template Recv Method

```go
func Gopt_MCPApp_Main(
    app       iAppProto,
    resources []ResourceProto,
    tools     []ToolProto,
    prompts   []PromptProto,
)
```

This is the key structural difference from the previous frameworks. Because the MCP protocol distinguishes resources, tools, and prompts at runtime, the framework needs to receive them as separate typed slices rather than a single variadic. The compiler groups each work class instance by its declared ProtocolType and passes them into the corresponding slice argument. The framework implementation then registers each group with the MCP server accordingly.

---

## Built-in: Unit Test Framework

XGo ships a built-in class framework for unit testing, with file suffix `_test.gox`. Given a function:

```go
func foo(v int) int {
    return v * 2
}
```

A `foo_test.gox` file can test it directly — no `TestXXX` function boilerplate required:

```go
if v := foo(50); v != 100 {
    t.error "foo(50) ret: ${v}"
}

t.run "foo -10", t => {
    if foo(-10) != -20 {
        t.fatal "foo(-10) != -20"
    }
}
```

---

## Design Patterns at a Glance

| Concern | Mechanism |
|---|---|
| Group all work class instances together | Single variadic parameter in Template Recv Method |
| Distinguish work class kinds at runtime | Multiple `class` directives + `ProtocolType` + typed slice parameters |
| Access work instances from the project class | `-embed` flag |
| Avoid type name collisions | `-prefix=<P>` flag |
| Custom `Main` / `MainEntry` parameter signature | `ProtocolType` interface declares the desired signature |
| Auto-call base class entry functions | Compiler inserts the call unconditionally when the base type implements the entry function |

---

## Existing Class Frameworks

| Framework | Domain | File Pattern |
|---|---|---|
| [spx](https://github.com/goplus/spx) | 2D game engine (STEM) | `*.spx` |
| [mcp](https://github.com/goplus/mcp) | MCP server (AI) | `*_mcp.gox`, `*_tool.gox`, etc. |
| [mcptest](https://github.com/goplus/mcp/tree/main/mtest) | MCP testing | `*_mtest.gox` |
| [yap](https://github.com/goplus/yap) | HTTP web framework | `*.yap` |
| [yaptest](https://github.com/goplus/yap/tree/main/ytest) | HTTP test framework | `*_ytest.gox` |
| [ydb](https://github.com/goplus/yap/tree/main/ydb) | Database framework | `*_ydb.gox` |
| [cobra](https://github.com/goplus/cobra) | CLI framework | `*_app.gox`, `*_cmd.gox` |
| [gsh](https://github.com/qiniu/x/tree/main/gsh) | Shell scripting | `*.gsh` |
| *(built-in)* | Unit testing | `*_test.gox` |
