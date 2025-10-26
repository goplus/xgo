<div align="center">
<p></p>
<p>
    <img width="80" src="https://xgo.dev/favicon.svg">
</p>
<h1>The XGo Programming Language</h1>


[xgo.dev](https://xgo.dev) | [Docs](doc/docs.md) | [XGo vs. Go](doc/xgo-vs-go.md) | [Tutorials](https://tutorial.xgo.dev) | [Playground](https://play.xgo.dev) | [XGo REPL (iXGo)](https://repl.xgo.dev) | [Contributing & compiler design](doc/contributing.md)
</div>

<div align="center">
<!--
[![VSCode](https://img.shields.io/badge/vscode-XGo-teal.svg)](https://github.com/gopcode/vscode-goplus)
[![Discord](https://img.shields.io/discord/983646982100897802?label=Discord&logo=discord&logoColor=white)](https://discord.gg/mYjWCJDcAr)
[![Interpreter](https://img.shields.io/badge/interpreter-iXGo-seagreen.svg)](https://github.com/goplus/ixgo)
-->

[![Build Status](https://github.com/goplus/xgo/actions/workflows/go.yml/badge.svg)](https://github.com/goplus/xgo/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/goplus/xgo)](https://goreportcard.com/report/github.com/goplus/xgo)
[![Coverage Status](https://codecov.io/gh/goplus/xgo/branch/main/graph/badge.svg)](https://codecov.io/gh/goplus/xgo)
[![GitHub release](https://img.shields.io/github/v/tag/goplus/xgo.svg?label=release)](https://github.com/goplus/xgo/releases)
[![Discord](https://img.shields.io/badge/Discord-online-success.svg?logo=discord&logoColor=white)](https://discord.com/invite/mYjWCJDcAr)

</div>

XGo is the first AI-native programming language that integrates software engineering into a unified whole.

```
XGo := C * Go * Python * JavaScript + Scratch
```

Our vision is to **enable everyone to become a builder of the world**.

#### Easy to learn

* Simple and easy to understand
* Smaller syntax set than Go and Python in best practices

#### Ready for large projects

* Integrate C/C++, Go, Python, and JavaScript into a unified ecosystem
* Derived from Go and easy to build large projects from its good engineering foundation

The XGo programming language is designed for engineering, STEM education, and data science.

* **For engineering**: working in the simplest language that can be mastered by children.
* **For STEM education**: studying an engineering language that can be used for work in the future.
* **For data science**: communicating with engineers in the same language.

For more details, see [Quick Start](doc/docs.md).


## Key Features of XGo

* Approaching natural language expression and intuitive (see [How XGo simplifies Go's expressions](#how-xgo-simplifies-gos-expressions)).
* Smallest but Turing-complete syntax set in best practices (see [The XGo Mini Specification](doc/spec-mini.md)).
* Fully compatible with [Go](https://github.com/golang/go) and can mix Go/XGo code in the same package (see [The XGo Full Specification](doc/spec.md) and [Go/XGo Hybrid Programming](doc/docs.md#gogo-hybrid-programming)).
* Integrating with the C ecosystem including Python/JavaScript and providing limitless possibilities based on [LLGo](https://github.com/goplus/llgo) (see [Importing C/C++ and Python libraries](#importing-cc-and-python-libraries)).
* Does not support DSL (Domain-Specific Languages), but supports SDF (Specific Domain Friendliness) (see [XGo Classfiles](#xgo-classfiles) and [Domain Text Literals](doc/domian-text-lit.md)).


## How XGo simplifies Go's expressions

Different from the function call style of most languages, XGo recommends command style code:

```coffee
println "Hello world"
```

To emphasize our preference for command style, we introduce `echo` as an alias for `println`:

```coffee
echo "Hello world"
```

For more discussion on coding style, see https://tutorial.xgo.dev/hello-world.

Code style is just the first step. We have made many efforts to make the code more intuitive and closer to natural language expression. These include:

| Go code | XGo code | Note |
| ---- | ---- | ---- |
| package main<br><br>import "fmt"<br><br>func main() {<br>&nbsp;&nbsp;&nbsp;&nbsp;fmt.Println("Hi")<br>} | import "fmt"<br><br>fmt.Println("Hi")<br> | Program structure: XGo allows omitting `package main` and `func main` |
| fmt.Println("Hi") | echo("Hi") | More builtin functions: It simplifies the expression of the most common tasks |
| fmt.Println("Hi") | echo "Hi" | Command-line style code: It reduces the number of parentheses in the code as much as possible, making it closer to natural language |
| name := "Ken"<br>fmt.Printf(<br>&nbsp;&nbsp;"Hi %s\n", name) | name := "Ken"<br>echo "Hi ${name}" | [Goodbye printf](doc/goodbye-printf.md), use `${expr}` in string literals |
| a := []int{1, 2, 3} | a := [1, 2, 3] | List literals |
| a = append(a, 4)<br>a = append(a, 5, 6, 7) | a <- 4<br>a <- 5, 6, 7 | Append values to a list |
| a := map[string]int{<br>&nbsp;&nbsp;&nbsp;&nbsp;"Monday": 1,<br>&nbsp;&nbsp;&nbsp;&nbsp;"Tuesday": 2,<br>} | a := {<br>&nbsp;&nbsp;&nbsp;&nbsp;"Monday": 1,<br>&nbsp;&nbsp;&nbsp;&nbsp;"Tuesday": 2,<br>} | Mapping literals |
| OnStart(func() {<br>&nbsp;&nbsp;&nbsp;&nbsp;...<br>}) | onStart => {<br>&nbsp;&nbsp;&nbsp;&nbsp;...<br>} | Lambda expressions |
| Play("1.mp3", &Options{Loop: true}) | play "1.mp3", loop = true | Python-like [keyword arguments](doc/docs.md#keyword-arguments) (kwargs) |
| type Rect struct {<br>&nbsp;&nbsp;&nbsp;&nbsp;Width&nbsp; float64<br>&nbsp;&nbsp;&nbsp;&nbsp;Height float64<br>}<br><br>func (this *Rect) Area() float64 { <br>&nbsp;&nbsp;&nbsp;&nbsp;return this.Width * this.Height<br>} | var (<br>&nbsp;&nbsp;&nbsp;&nbsp;Width&nbsp; float64<br>&nbsp;&nbsp;&nbsp;&nbsp;Height float64<br>)<br><br>func Area() float64 { <br>&nbsp;&nbsp;&nbsp;&nbsp;return Width * Height<br>} | [XGo Classfiles](doc/classfile.md): We can express OOP with global variables and functions. |

For more details, see [The XGo Mini Specification](doc/spec-mini.md).


## Importing C/C++ and Python libraries

XGo can choose different Go compilers as its underlying support. Currently known supported Go compilers include:

* [go](https://go.dev/) (The official Go compiler supported by Google)
* [llgo](https://github.com/goplus/llgo) (The Go compiler supported by the XGo team)
* [tinygo](https://tinygo.org/) (A Go compiler for small places)

Currently, XGo defaults to using [go](https://go.dev/) as its underlying support, but in the future, it will be [llgo](https://github.com/goplus/llgo).

LLGo is a Go compiler based on [LLVM](https://llvm.org/) in order to better integrate Go with the C ecosystem including Python and JavaScript. It aims to expand the boundaries of Go/XGo, providing limitless possibilities such as:

* Game development
* AI and data science
* WebAssembly
* Embedded development
* ...

If you wish to use [llgo](https://github.com/goplus/llgo), specify the `-llgo` flag when initializing an XGo module:

```sh
xgo mod init -llgo YourModulePath
```

This will generate a `go.mod` file with the following contents (It may vary slightly depending on the versions of local XGo and LLGo):

```go
module YourModulePath

go 1.21 // llgo 1.0

require github.com/goplus/lib v0.2.0
```

Based on LLGo, XGo can import libraries written in C/C++ and Python.

Here is an example (see [chello](demo/_llgo/chello/hello.xgo)) of printing `Hello world` using C's `printf`:

```go
import "c"

c.printf c"Hello world\n"
```

Here, `c"Hello world\n"` is a syntax supported by XGo, representing a null-terminated C-style string.

To run this example, you can:

```sh
cd YourModulePath  # set work directory to your module
xgo mod tidy       # for generating go.sum file
xgo run .
```

And here is an example (see [pyhello](demo/_llgo/pyhello/hello.xgo)) of printing `Hello world` using Python's `print`:

```go
import "py/std"

std.print py"Hello world"
```

Here, `py"Hello world"` is a syntax supported by XGo, representing a Python string.

Here are more examples of XGo calling C/C++ and Python libraries:

* [pytensor](demo/_llgo/pytensor/tensor.xgo): a simple demo using [py/torch](https://pkg.go.dev/github.com/goplus/lib/py/torch)
* [tetris](demo/_llgo/tetris/tetris.xgo): a tetris game based on [c/raylib](https://pkg.go.dev/github.com/goplus/lib/c/raylib)
* [sqlitedemo](demo/_llgo/sqlitedemo/sqlitedemo.xgo): a demo using [c/sqlite](https://pkg.go.dev/github.com/goplus/lib/c/sqlite)

To find out more about LLGo/XGo's support for C/C++ and Python in detail, please refer to homepage of [llgo](https://github.com/goplus/llgo).


## XGo Classfiles

```
One language can change the whole world.
XGo is a "DSL" for all domains.
```

Rob Pike once said that if he could only introduce one feature to Go, he would choose `interface` instead of `goroutine`. `classfile` (and `class framework`) is as important to XGo as `interface` is to Go.

In the design philosophy of XGo, we do not recommend `DSL` (Domain Specific Language). But `SDF` (Specific Domain Friendliness) is very important. The XGo philosophy about `SDF` is:

```
Don't define a language for specific domain.
Abstract domain knowledge for it.
```

XGo introduces `classfile` and `class framework` to abstract domain knowledge.

* [What's Classfile?](doc/classfile.md#whats-classfile)
* [Dive into XGo Classfiles](doc/classfile.md)

Sound a bit abstract? Let's see some XGo class frameworks.

* STEM Education: [spx: An XGo 2D Game Engine](https://github.com/goplus/spx)
* AI Programming: [mcp: An XGo implementation of the Model Context Protocol (MCP)](https://github.com/goplus/mcp)
* AI Programming: [mcptest: An XGo MCP Test Framework](https://github.com/goplus/mcp/tree/main/mtest)
* Web Programming: [yap: Yet Another HTTP Web Framework](https://github.com/goplus/yap)
* Web Programming: [yaptest: An XGo HTTP Test Framework](https://github.com/goplus/yap/tree/main/ytest)
* Web Programming: [ydb: An XGo Database Framework](https://github.com/goplus/yap/tree/main/ydb)
* CLI Programming: [cobra: A Commander for modern XGo CLI interactions](https://github.com/goplus/cobra)
* CLI Programming: [gsh: An alternative to write shell scripts](https://github.com/qiniu/x/tree/main/gsh)
* Unit Test: [test: Unit Test](doc/classfile.md#class-framework-unit-test)


### yap: Yet Another HTTP Web Framework

This classfile has the file suffix `.yap`.

Create a file named [get.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_hello/get.yap) with the following content:

```go
html `<html><body>Hello, YAP!</body></html>`
```

Execute the following commands:

```sh
xgo mod init hello
xgo get github.com/goplus/yap@latest
xgo mod tidy
xgo run .
```

A simplest web program is running now. At this time, if you visit http://localhost:8080, you will get:

```
Hello, YAP!
```

YAP uses filenames to define routes. `get.yap`'s route is `get "/"` (GET homepage), and `get_p_#id.yap`'s route is `get "/p/:id"` (In fact, the filename can also be `get_p_:id.yap`, but it is not recommended because `:` is not allowed to exist in filenames under Windows).

Let's create a file named [get_p_#id.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_hello/get_p_%23id.yap) with the following content:

```coffee
json {
	"id": ${id},
}
```

Execute `xgo run .` and visit http://localhost:8080/p/123, you will get:

```
{"id": "123"}
```

See [yap: Yet Another HTTP Web Framework](https://github.com/goplus/yap) for more details.


### spx: An XGo 2D Game Engine

![Screen Shot1](https://github.com/goplus/spx/blob/v1/tutorial/01-Weather/1.jpg) ![Screen Shot2](https://github.com/goplus/spx/blob/v1/tutorial/01-Weather/2.jpg)

Through this example you can learn how to implement dialogues between multiple actors.

Here are some codes in [Kai.spx](https://github.com/goplus/spx/blob/v1/tutorial/01-Weather/Kai.spx):

```coffee
onStart => {
	say "Where do you come from?", 2
	broadcast "1"
}

onMsg "2", => {
	say "What's the climate like in your country?", 3
	broadcast "3"
}
```

We call `onStart` and `onMsg` to listen events. `onStart` is called when the program is started. And `onMsg` is called when someone calls `broadcast` to broadcast a message.

When the program starts, Kai says `Where do you come from?`, and then broadcasts the message `1`. Who will recieve this message? Let's see codes in [Jaime.spx](https://github.com/goplus/spx/blob/v1/tutorial/01-Weather/Jaime.spx):

```coffee
onMsg "1", => {
	say "I come from England.", 2
	broadcast "2"
}
```

Yes, Jaime recieves the message `1` and says `I come from England.`. Then he broadcasts the message `2`. Kai recieves it and says `What's the climate like in your country?`.

The following procedures are very similar. In this way you can implement dialogues between multiple actors.

See [spx: An XGo 2D Game Engine](https://github.com/goplus/spx) for more details.


### gsh: XGo DevOps Tools

Yes, now you can write `shell script` in XGo. It supports all shell commands.

Let's create a file named [example.gsh](https://github.com/qiniu/x/blob/main/gsh/demo/hello/example.gsh) and write the following code:

```coffee
mkdir "testgsh"
```

Don't need a `go.mod` file, just enter `xgo run ./example.gsh` directly to run.

See [gsh: XGo DevOps Tools](https://github.com/qiniu/x/tree/main/gsh) for more details.


## How to install

Note: Requires go1.19 or later

### on Windows

```sh
winget install goplus.xgo
```

### on Debian/Ubuntu

```sh
sudo bash -c ' echo "deb [trusted=yes] https://pkgs.xgo.dev/apt/ /" > /etc/apt/sources.list.d/goplus.list'
sudo apt update
sudo apt install xgo
```

### on RedHat/CentOS/Fedora

```sh
sudo bash -c 'echo -e "[goplus]\nname=XGo Repo\nbaseurl=https://pkgs.xgo.dev/yum/\nenabled=1\ngpgcheck=0" > /etc/yum.repos.d/goplus.repo'
sudo yum install xgo
```

### on macOS/Linux (Homebrew)

Install via [brew](https://brew.sh/)

```sh
$ brew install xgo
```

### from source code

```bash
git clone https://github.com/goplus/xgo.git
cd xgo

# On mac/linux run:
./all.bash
# On Windows run:
all.bat
```

## XGo Applications

### 2D Games powered by XGo

* [An XGo 2D Game Engine for STEM education](https://github.com/goplus/spx)
* [Aircraft War](https://github.com/goplus/AircraftWar)
* [Flappy Bird](https://github.com/goplus/FlappyCalf)
* [Maze Play](https://github.com/goplus/MazePlay)
* [BetaGo](https://github.com/xushiwei/BetaGo)
* [Gobang](https://github.com/xushiwei/Gobang)
* [Dinosaur](https://github.com/xushiwei/Dinosaur)

### Web Programming

* [yap: Yet Another HTTP Web Framework](https://github.com/goplus/yap)
* [yaptest: HTTP Test Framework](https://github.com/goplus/yap/tree/main/ytest)
* [ydb: Database Framework](https://github.com/goplus/yap#ydb-database-framework)

### DevOps Tools

* [gsh: XGo DevOps Tools](https://github.com/qiniu/x/tree/main/gsh)

### Data Processing

* [hdq: HTML DOM Query Language for XGo](https://github.com/goplus/hdq)


## IDE Plugins

* vscode: [Go/XGo for Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=goplus.gop)


## Contributing

The XGo project welcomes all contributors. We appreciate your help!

For more details, see [Contributing & compiler design](doc/contributing.md).


## Give a Star! ⭐

If you like or are using XGo to learn or start your projects, please give it a star. Thanks!
