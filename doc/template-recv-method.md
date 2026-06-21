# Template Recv Method

A **Template Recv Method** lets a framework base type call back into a method that you — the user — implement on your own type. The dispatch is handled automatically at the call site; you do not need to register anything.

## Abstract

This proposal introduces **Template Recv Method** — a mechanism that gives Go-style embedded-struct polymorphism the ergonomics of C++ virtual dispatch, without adding virtual functions or vtables to XGo's type system. A Template Recv Method is a function whose *receiver type* is a type parameter constrained to types that embed a designated base struct. Because XGo compiles to Go, the feature is defined at the Go (assembly) level using a naming convention, and the XGo compiler then exposes it as a regular method call — with no new XGo declaration syntax required.

---

## Motivation

Go intentionally omits virtual functions. Polymorphism is achieved through interfaces, which is the right default. However, a specific pattern — a *framework base type* that needs to call back into a method supplied by the concrete user type — is awkward in plain Go.

### The C++ baseline

```cpp
class Game {
public:
    virtual void OnUpdate() = 0;   // subclass must implement
    void Run() { /* calls OnUpdate() */ }
};

class MyGame : public Game {
    void OnUpdate() override { /* ... */ }
};
```

`Run` calls the override of `OnUpdate` that belongs to the concrete subclass. This is straightforward virtual dispatch.

### The idiomatic Go equivalent

```go
type Updater interface {
    OnUpdate()
}

type Game struct {
    updater Updater
}

func (g *Game) SetUpdater(updater Updater) {
    g.updater = updater
}

func (g *Game) Run() {
    // calls g.updater.OnUpdate() internally
}

type MyGame struct {
    Game
}

func NewMyGame() *MyGame {
    g := new(MyGame)
    g.SetUpdater(g)   // ← caller must remember this
    return g
}

func (g *MyGame) OnUpdate() { /* ... */ }
```

The approach is correct and explicit, but it imposes two burdens on framework users:

1. They must call `SetUpdater(self)` in every constructor, an easy mistake to forget.
2. The relationship between `Game` and the callback interface is invisible at the type-declaration site.

---

## Proposal

### Concept

A **Template Recv Method** is a method whose *receiver type* is itself a type parameter `T`, constrained to types that embed (aggregate) a specific base struct `B`. When such a method is called on a value of type `T`, the method can call other methods on the concrete type `T` — achieving virtual-dispatch semantics without runtime indirection beyond what Go's interface machinery already provides.

### Pseudocode semantics

The following is *illustrative pseudocode* — not XGo syntax — that expresses the intended semantics:

```
// A default implementation of OnUpdate, attached to Game itself.
func [g *Game] OnUpdate() { ... }

// Run is a template method: T must embed Game.
// Calling g.OnUpdate() inside Run dispatches to T's OnUpdate, not Game's.
func [T Game] (g *T) Run() {
    g.OnUpdate()   // dispatches to MyGame.OnUpdate when T = MyGame
}
```

Call sites:

```go
var a Game
a.Run()       // T inferred as Game  → calls Game.OnUpdate

var b MyGame
b.Run()       // T inferred as MyGame → calls MyGame.OnUpdate
```

---

## Go-Level Encoding ("Assembly" Representation)

The design of any new XGo feature must begin with a precise Go encoding. For Template Recv Method, that encoding is a package-level generic function following the `XGot_` naming convention.

### Naming convention: `XGot_`

A Template Recv Method is encoded in Go as a package-level generic function following the convention:

```
XGot_<BaseType>_<MethodName>[T <constraint>](recv T, ...)
```

| Part | Meaning |
|---|---|
| `XGot_` | Prefix: **XGo** **t**emplate recv method |
| `<BaseType>` | The base struct the receiver must embed |
| `<MethodName>` | The logical method name |
| `T <constraint>` | Type parameter; `constraint` is an interface that lists the callbacks the base method needs |

### Example

```go
// The callback interface: what Game.Run needs from the concrete type.
type gamer interface {
    OnUpdate()
}

// Template Recv Method for Game.Run
func XGot_Game_Run[T gamer](g T) {
    // g is the concrete receiver; g.OnUpdate() calls the concrete OnUpdate.
    g.OnUpdate()
}
```

The `XGot_` prefix signals to the XGo compiler that this function is the Go-level encoding of a Template Recv Method and should be lifted into a proper method on any type embedding `Game`.

### Default method body

When `Game` itself needs a default implementation of `OnUpdate` (so that `Game` satisfies `gamer` and can be used standalone), it is declared normally:

```go
func (g *Game) OnUpdate() { /* default, possibly no-op */ }
```

### Full Example

This example reflects the intended division of labour: the framework author writes Go; the end user writes XGo.

#### Go library (framework side)

```go
// Package game provides the Game framework base type.
package game

type Game struct{ /* ... */ }

// gamer is the internal constraint interface.
// It lists exactly the callbacks that Game's template methods require.
type gamer interface {
    OnUpdate()
}

// Default OnUpdate on Game itself (makes *Game satisfy gamer).
func (g *Game) OnUpdate() {}

// XGot_Game_Run is the Go encoding of the Template Recv Method Run.
// The XGo compiler lifts this into a proper method on any type embedding Game.
func XGot_Game_Run[T gamer](g T) {
    g.OnUpdate()
}
```

#### XGo consumer (user side)

```go
import "game"

type MyGame struct {
    game.Game
}

func (g *MyGame) OnUpdate() {
    // user-supplied drawing logic
}

func main() {
    var a game.Game
    a.Run()       // dispatches to (*game.Game).OnUpdate

    var b MyGame
    b.Run()       // dispatches to (*MyGame).OnUpdate
}
```

The XGo compiler recognises `XGot_Game_Run` and exposes it as the method `Run` on any type embedding `game.Game`. The call `b.Run()` is rewritten to `game.XGot_Game_Run(&b)`, and Go's generic instantiation at `T = MyGame` ensures the dispatch reaches `(*MyGame).OnUpdate`. No new XGo syntax is required on either side.

---

## Go as XGo's "Assembly Language"

> Go Is XGo's "Assembly Language"
>
> That framing sounds bold, but it is precise. XGo does not treat Go as a mere transport medium the way some "compiles-to-Go" languages do. XGo treats Go as its **semantic foundation**:
> - Every XGo package can be translated one-to-one into a Go package.
> - The translated Go package **preserves semantics exactly** and can be consumed by any standard Go toolchain.
> - Conversely, that translated Go package can also be imported by other XGo code, with no bridging layer required.
>
> This means XGo's type system, memory model, and concurrency primitives are all inherited directly from Go. Go's compiler-level verification applies uniformly to all XGo source — no exceptions, no carve-outs.

This proposal is a direct example of that pattern. The `XGot_`-prefixed function **must be defined in Go**, not in XGo. XGo currently has no grammar to express a generic function declaration with the `XGot_` prefix semantics — the prefix convention is a contract between the Go author and the XGo compiler, not something XGo source can produce itself.

This is intentional and consistent with XGo's broader design philosophy: when a feature requires low-level or structurally complex declaration syntax that XGo does not yet surface, Go serves as the "assembly language" in which that declaration is written. The XGo layer then provides the ergonomic call-site syntax that makes the feature accessible to end users — in this case, allowing `b.Run()` on a `MyGame` value to dispatch polymorphically to `(*MyGame).OnUpdate`, with no manual wiring required.

Notably, this feature does not require XGo to introduce any new declaration syntax. The `XGot_` convention is sufficient to bridge the expressive gap permanently — not as a temporary measure, but as a deliberate design choice consistent with XGo's conservatism toward syntax additions. The framework author writes Go; the end user writes XGo; the compiler connects the two through the naming convention alone.

---

## Design Decisions

### Why a prefix rather than a new keyword?

XGo's Go-encoding layer (its "assembly language") must be valid, idiomatic Go. Introducing a keyword or special syntax at the Go level would break `go build` compatibility. A naming convention (`XGot_`) is zero-cost: the Go toolchain sees an ordinary generic function; the XGo compiler layer recognizes the pattern by name and promotes it.

### Why move the receiver to a parameter position?

Go does not allow generic receivers of the form `func [T C] (g T) M()` — the receiver's type must be the package's own named type. Moving the receiver to a regular parameter position (`func XGot_Base_Method[T C](g T)`) is therefore the only way to encode the semantics in standard Go generics.

### Why a separate constraint interface (`gamer`)?

The constraint interface makes the dependency explicit: it states precisely which callbacks `Run` requires. This is more maintainable than relying on the full method set of a concrete type and ensures the XGo compiler can verify that any type passed as `T` provides exactly the methods that the template method calls.

### No implicit `SetUpdater` required

Unlike the idiomatic Go workaround, the user of a Template Recv Method does not need to call any registration function. The dispatch is handled at the call site by Go's generic instantiation — no runtime bookkeeping is introduced.

---

## Comparison

| | C++ virtual | Idiomatic Go (interface + SetUpdater) | Template Recv Method |
|---|---|---|---|
| Runtime cost | vtable lookup | interface dispatch | interface dispatch (same) |
| User boilerplate | none | `SetUpdater(self)` in ctor | none |
| Explicit contract | no (implicit override) | interface in framework code | constraint interface |
| Go-compatible encoding | n/a | yes | yes (`XGot_` convention) |
| Requires new XGo syntax | n/a | no | no |

---

## Summary

Template Recv Method fills the specific gap where a framework base type (`Game`) needs to call back into a method provided by a concrete embedding type (`MyGame`), without requiring the user to wire up a dispatcher manually. The feature is grounded in standard Go generics via the `XGot_` naming convention, ensuring full compatibility with the Go toolchain. The framework author writes Go; the end user writes XGo and calls `b.Run()` naturally. No new XGo declaration syntax is needed — the naming convention alone is sufficient, and that is by design.
