This is an example to show how to mix Go/XGo code in the same package.

In this example, we have a Go source file named `a.go`:

```go
package main

import "fmt"

func p(a interface{}) {
	sayMix()
	fmt.Println("Hello,", a)
}
```

And we have an XGo source file named `b.xgo`:

```go
func sayMix() {
	println "Mix Go and XGo"
}

p "world"
```

You can see that Go calls an XGo function named `sayMix`, and XGo calls a Go function named `p`. As you are used to in Go programming, this kind of circular reference is allowed.

Run `xgo run .` to see the output of this example:

```
Mix Go and XGo
Hello, world
```

### Give a Star! ⭐

If you like or are using XGo to learn or start your projects, please give it a star. Thanks!
