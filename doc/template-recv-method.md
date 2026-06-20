Template Recv Method
=====

A **Template Recv Method** lets a framework base type call back into a method that you — the user — implement on your own type. The dispatch is handled automatically at the call site; you do not need to register anything.

### The Problem It Solves

Suppose you are using a game framework that provides a `Game` base struct. The framework's `Run` method needs to call your `OnDraw` method at the right moment. In plain Go you would have to wire up a callback interface yourself:

```go
// Plain Go — extra wiring required
type MyGame struct {
    game.Game
}

func NewMyGame() *MyGame {
    g := new(MyGame)
    g.SetDrawer(g)   // ← easy to forget
    return g
}

func (g *MyGame) OnDraw() {
    // your drawing logic
}
```

With a Template Recv Method, the framework handles the dispatch for you. There is no `SetDrawer` to call and nothing to forget:

```go
// XGo — no wiring required
type MyGame struct {
    game.Game
}

func (g *MyGame) OnDraw() {
    // your drawing logic
}

func main() {
    var b MyGame
    b.Run()   // automatically dispatches to (*MyGame).OnDraw
}
```

### Basic Usage

As a framework user, there are exactly three things to do:

1. Embed the framework's base struct in your type.
2. Implement the callback method(s) that the framework needs.
3. Call the method — XGo takes care of the rest.

```go
import "game"

// Step 1: embed the base struct.
type MyGame struct {
    game.Game
}

// Step 2: implement the callback.
func (g *MyGame) OnDraw() {
    println("drawing my game")
}

func main() {
    // Step 3: call the method naturally.
    var g MyGame
    g.Run()
}
```

Output:

```
drawing my game
```

### Calling on the Base Type

If you use the base type directly (without embedding it in a custom type), the base type's own default implementation is called:

```go
var g game.Game
g.Run()   // calls game.Game's default OnDraw
```

Both call sites use the same `Run()` syntax — XGo selects the right implementation based on the concrete type.

### Implementing Multiple Callbacks

A template method may require more than one callback. You implement all of them on your type; any that you omit fall back to the base type's default:

```go
type MyGame struct {
    game.Game
}

// Implement the ones you care about.
func (g *MyGame) OnDraw()   { println("draw") }
func (g *MyGame) OnUpdate() { println("update") }
// OnResize is omitted — the base type's default is used.
```

### Nesting and Composition

Template Recv Methods work correctly when your type is itself embedded further:

```go
type SpecialGame struct {
    MyGame   // MyGame already embeds game.Game
}

func (g *SpecialGame) OnDraw() {
    println("special draw")
}

func main() {
    var g SpecialGame
    g.Run()   // dispatches to (*SpecialGame).OnDraw
}
```

---

## Under the Hood: How the Framework Is Written in Go

This section is for framework authors who write the Go-side code that XGo users call. End users do not need to read it.

### The `XGot_` Naming Convention

A Template Recv Method is encoded in Go as a package-level generic function following the convention:

```
XGot_<BaseType>_<MethodName>[T <constraint>](recv T, ...)
```

| Part | Meaning |
|---|---|
| `XGot_` | Prefix marking a template recv method |
| `<BaseType>` | The base struct the receiver must embed |
| `<MethodName>` | The method name exposed to XGo callers |
| `T <constraint>` | Type parameter; `constraint` lists the callbacks the method needs |

The `XGot_` prefix is the signal to the XGo compiler. The compiler lifts the function into a proper method on any type that embeds `<BaseType>`, so the XGo call `g.Run()` is valid and dispatches correctly.

### Defining a Template Recv Method

```go
package game

type Game struct{ /* ... */ }

// gamer declares the callbacks that Run requires from the concrete type.
type gamer interface {
    OnDraw()
}

// Default implementation so that *Game itself satisfies gamer.
func (g *Game) OnDraw() {}

// XGot_Game_Run is the Go encoding of the template method Run.
// The XGo compiler exposes it as Run() on any type embedding Game.
func XGot_Game_Run[T gamer](g T) {
    g.OnDraw()   // dispatches to T's OnDraw
}
```

When an XGo user writes `b.Run()` on a `MyGame` value, the compiler rewrites it to `game.XGot_Game_Run(&b)`. Go's generic instantiation at `T = *MyGame` ensures that `g.OnDraw()` inside the function resolves to `(*MyGame).OnDraw`.

### Multiple Parameters

Template Recv Methods support additional parameters beyond the receiver:

```go
func XGot_Game_OnKeyPress[T gamer](g T, key int) {
    g.HandleKey(key)
}
```

XGo users call it as:

```go
g.OnKeyPress(keyCode)
```

### Constraint Interface

The constraint interface (`gamer` in the example above) lists exactly the callbacks that the template method calls. Keep it minimal — only the methods that the template body actually invokes. This makes the contract explicit and lets the XGo compiler verify that any concrete type passed as `T` provides exactly those methods.

### Summary: What Goes Where

| Author | Writes | How |
|---|---|---|
| Framework author | `XGot_` functions, constraint interfaces, default implementations | In Go |
| End user | Embedding struct, callback implementations, call sites | In XGo |

The `XGot_` naming convention is the only bridge needed. No new XGo declaration syntax is required on either side.
