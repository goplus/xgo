This is an example to show how XGo interacts with C.

```go
import "c"

c.printf c"Hello, llgo!\n"
c.fprintf c.Stderr, c"Hi, %6.1f\n", 3.14
```

Here we use `import "c"` to import libc. It's an abbreviation for `import "github.com/goplus/lib/c"`. It is equivalent to the following code:

```go
import "github.com/goplus/lib/c"

c.printf c"Hello, llgo!\n"
c.fprintf c.Stderr, c"Hi, %7.1f\n", 3.14
```

In this example we call two C standard functions `printf` and `fprintf`, pass a C variable `stderr` and two C strings in the form of `c"xxx"`.

To run this demo, you need to set the `XGO_GOCMD` environment variable first.

```sh
export XGO_GOCMD=llgo  # default is `go`
```

Then execute `xgo run .` to see the output of this example:

```
Hello, llgo!
Hi,    3.1
```

### Give a Star! ⭐

If you like or are using XGo to learn or start your projects, please give it a star. Thanks!
