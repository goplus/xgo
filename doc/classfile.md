XGo Classfiles
=====

```
One language can change the world.
XGo is a "DSL" for all domains.
```

Rob Pike once said that if he could only introduce one feature to Go, he would choose `interface` instead of `goroutine`. `classfile` (and `class framework`) is as important to XGo as `interface` is to Go.

In the design philosophy of XGo, we do not recommend `DSL` (Domain Specific Language). But `SDF` (Specific Domain Friendliness) is very important. The XGo philosophy about `SDF` is:

```
Don't define a language for specific domain.
Abstract domain knowledge for it.
```

XGo introduces `classfile` and `class framework` to abstract domain knowledge.

* STEM Education: [spx: A XGo 2D Game Engine](https://github.com/goplus/spx)
* AI Programming: [mcp: A XGo implementation of the Model Context Protocol (MCP)](https://github.com/goplus/mcp)
* AI Programming: [mcptest: A XGo MCP Test Framework](https://github.com/goplus/mcp/tree/main/mtest)
* Web Programming: [yap: Yet Another HTTP Web Framework](https://github.com/goplus/yap)
* Web Programming: [yaptest: A XGo HTTP Test Framework](https://github.com/goplus/yap/tree/main/ytest)
* Web Programming: [ydb: A XGo Database Framework](https://github.com/goplus/yap/tree/main/ydb)
* CLI Programming: [cobra: A Commander for modern XGo CLI interactions](https://github.com/goplus/cobra)
* CLI Programming: [gsh: An alternative to write shell scripts](https://github.com/qiniu/x/tree/main/gsh)
* Unit Test: [class framework: Unit Test](#class-framework-unit-test)
* Mechanism: [What's Classfile](#whats-classfile)

Sound a bit abstract? Let's take web programming as an example. First let us create a file named [get.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_hello/get.yap) with the following content:

```go
html `<html><body>Hello, YAP!</body></html>`
```

Execute the following commands:

```sh
gop mod init hello
gop get github.com/goplus/yap@latest
gop mod tidy
gop run .
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

Execute `gop run .` and visit http://localhost:8080/p/123, you will get:

```
{"id": "123"}
```

Why is `yap` so easy to use? How does it do it? Let us analyze the principles one by one.


### What's classfile

What's a classfile? And why it is called `classfile`?

First let's create a file called `Rect.gox`:

```go
var (
	Width, Height int
)

func Area() int {
	return Width * Height
}
```

Then we create `hello.xgo` file in the same directory:

```go
rect := &Rect{10, 20}
echo rect.area
```

Then we execute `gop run .` to run it and get the result:

```sh
200
```

This shows that the `Rect.gox` file actually defines a class named `Rect`. If we express it in Go syntax, it looks like this:

```go
type Rect struct {
	Width, Height int
}

func (this *Rect) Area() int {
	return this.Width * this.Height
}
```

So the name `classfile` comes from the fact that it actually defines a class.

You may ask: What is the value of doing this?

The value lies in its ease of use, especially for children and non-expert programmers. Let's look at this syntax:

```go
var (
	Width, Height int
)

func Area() int {
	return Width * Height
}
```

Defining variables and defining functions are all familiar to them while learning sequential programming. They can define new types using syntax they already know by heart. This will be valuable in getting a wider community to learn XGo.

### What's class framework

Of course, this is not enough to make classfiles an exciting feature. What's more important is its ability to abstract domain knowledge. It is accomplished by defining `base class` for a class and defining `relationships between multiple classes`.

What is a `class framework`? Usually it consists of a `project class` and multiple `work classes`. The class framework not only specifies the `base class` of all `project class` and `work classes`, but also organizes all these classes together by the base class of project class. There can be no work classes, that is, the entire classfile consists of only one project class.

This is a bit abstract. Let's take the [2D Game Engine spx](https://github.com/goplus/spx) as an example. The base class of project class of `spx class framework` is called `Game`. The base class of work class is called `Sprite`. Obviously, there will only be one Game instance in a game, but there are many types of sprites, so many types of work classes are needed, but they all have the same base class called `Sprite`. XGo's class framework allows you to specify different base classes for different work classes. Although this is rare, it can be done.

How does XGo identify various class files of a class framework? by its filename. By convention, if we define a class framework called `foo`, then its project class is usually called `main_foo.gox`, and the work class is usually called `xxx_foo.gox`. If this class framework does not have a work class, then the project class only needs to ensure that the suffix is `_foo.gox`, and the class name can be freely chosen.

The earliest version of XGo allows classfiles to be identified through custom file extensions. For example, the project class of the `spx class framework` is called `main.spx`, and the work class is called `xxx.spx`. Although this ability to customize extensions is still retained for now, we do not recommend its use and there is no guarantee that it will continue to be available in the future.


### class framework: Unit Test

XGo has a built-in class framework to simplify unit testing. This class framework has the file suffix `_test.gox`.

Suppose you have a function named `foo`:

```go
func foo(v int) int {
	return v * 2
}
```

Then you can create a `foo_test.gox` file to test it (see [unit-test/foo_test.gox](../demo/unit-test/foo_test.gox)):

```go
if v := foo(50); v != 100 {
	t.error "foo(50) ret: ${v}"
}

t.run "foo -10", t => {
	if foo(-10) != -20 {
		t.fatal "foo(-10) != -20"
	}
}

t.run "foo 0", t => {
	if foo(0) != 0 {
		t.fatal "foo(0) != 0"
	}
}
```

You don't need to define a series of `TestXXX` functions like Go, just write your test code directly.

If you want to run a subtest case, use `t.run`.


### yap: Yet Another Go/XGo HTTP Web Framework

This class framework has the file suffix `.yap`. See [yap: Yet Another HTTP Web Framework](https://github.com/goplus/yap) for more details.

#### Router and Parameters

YAP uses filenames to define routes. `get.yap`'s route is `get "/"` (GET homepage), and `get_p_#id.yap`'s route is `get "/p/:id"` (In fact, the filename can also be `get_p_:id.yap`, but it is not recommended because `:` is not allowed to exist in filenames under Windows).

Let's create a file named [get_p_#id.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_hello/get_p_%23id.yap) with the following content:

```coffee
json {
	"id": ${id},
}
```

Execute `gop run .` and visit http://localhost:8080/p/123, you will get:

```
{"id": "123"}
```


#### YAP Template

In most cases, we don't use the `html` directive to generate html pages, but use the `yap` template engine. See [get_p_#id.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_blog/get_p_%23id.yap):

```coffee
yap "article", {
	"id": ${id},
}
```

It means finding a template called `article` to render. See [yap/article_yap.html](https://github.com/goplus/yap/blob/main/demo/classfile2_blog/yap/article_yap.html):

```html
<html>
<head><meta charset="utf-8"/></head>
<body>Article {{.id}}</body>
</html>
```

#### Run at specified address

By default the YAP server runs on `localhost:8080`, but you can change it in [main.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_blog/main.yap) file:

```coffee
run ":8888"
```


#### Static files

Static files server demo ([main.yap](https://github.com/goplus/yap/blob/main/demo/classfile2_static/main.yap)):

```coffee
static "/foo", FS("public")
static "/"

run ":8080"
```


### yaptest: HTTP Test Framework

This class framework has the file suffix `_ytest.gox`.

Suppose we have a web server ([foo/get_p_#id.yap](https://github.com/goplus/yap/blob/main/ytest/demo/foo/get_p_%23id.yap)):

```go
json {
	"id": ${id},
}
```

Then we create a yaptest file ([foo/foo_ytest.gox](https://github.com/goplus/yap/blob/main/ytest/demo/foo/bar_ytest.gox)):

```go
mock "foo.com", new(AppV2)  // name of any YAP v2 web server is `AppV2`

id := "123"
get "http://foo.com/p/${id}"
ret 200
json {
	"id": id,
}
```

The directive `mock` creates the web server by [mockhttp](https://pkg.go.dev/github.com/qiniu/x/mockhttp). Then we write test code directly.

You can change the directive `mock` to `testServer` (see [foo/bar_ytest.gox](https://github.com/goplus/yap/blob/main/ytest/demo/foo/bar_ytest.gox)), and keep everything else unchanged:

```go
testServer "foo.com", new(AppV2)

id := "123"
get "http://foo.com/p/${id}"
ret 200
json {
	"id": id,
}
```

The directive `testServer` creates the web server by [net/http/httptest](https://pkg.go.dev/net/http/httptest#NewServer) and obtained a random port as the service address. Then it calls the directive [host](https://pkg.go.dev/github.com/goplus/yap/ytest#App.Host) to map the random service address to `foo.com`. This makes all other code no need to changed.

For more details, see [yaptest - XGo HTTP Test Framework](https://github.com/goplus/yap/blob/main/ytest).


### spx: A XGo 2D Game Engine for STEM education

This class framework has the file suffix `.spx`. It is the earliest class framework in the world.

#### tutorial/01-Weather

![Screen Shot1](https://github.com/goplus/spx/blob/main/tutorial/01-Weather/1.jpg) ![Screen Shot2](https://github.com/goplus/spx/blob/main/tutorial/01-Weather/2.jpg)

Through this example you can learn how to implement dialogues between multiple actors.

Here are some codes in [Kai.spx](https://github.com/goplus/spx/blob/main/tutorial/01-Weather/Kai.spx):

```coffee
onStart => {
	say "Where do you come from?", 2
	broadcast "1"
}

onMsg "2", => {
	say "What's the climate like in your country?", 3
	broadcast "3"
}

onMsg "4", => {
	say "Which seasons do you like best?", 3
	broadcast "5"
}
```

We call `onStart` and `onMsg` to listen events. `onStart` is called when the program is started. And `onMsg` is called when someone calls `broadcast` to broadcast a message.

When the program starts, Kai says `Where do you come from?`, and then broadcasts the message `1`. Who will recieve this message? Let's see codes in [Jaime.spx](https://github.com/goplus/spx/blob/main/tutorial/01-Weather/Jaime.spx):

```coffee
onMsg "1", => {
	say "I come from England.", 2
	broadcast "2"
}

onMsg "3", => {
	say "It's mild, but it's not always pleasant.", 4
	# ...
	broadcast "4"
}
```

Yes, Jaime recieves the message `1` and says `I come from England.`. Then he broadcasts the message `2`. Kai recieves it and says `What's the climate like in your country?`.

The following procedures are very similar. In this way you can implement dialogues between multiple actors.

#### tutorial/02-Dragon

![Screen Shot1](https://github.com/goplus/spx/blob/main/tutorial/02-Dragon/1.jpg)

Through this example you can learn how to define variables and show them on the stage.

Here are all the codes of [Dragon](https://github.com/goplus/spx/blob/main/tutorial/02-Dragon/Dragon.spx):

```coffee
var (
	score int
)

onStart => {
	score = 0
	for {
		turn rand(-30, 30)
		step 5
		if touching("Shark") {
			score++
			play chomp, true
			step -100
		}
	}
}
```

We define a variable named `score` for `Dragon`. After the program starts, it moves randomly. And every time it touches `Shark`, it gains one score.

How to show the `score` on the stage? You don't need write code, just add a `stageMonitor` object into [assets/index.json](https://github.com/goplus/spx/blob/main/tutorial/02-Dragon/assets/index.json):

```json
{
  "zorder": [
    {
      "type": "stageMonitor",
      "target": "Dragon",
      "val": "getVar:score",
      "color": 15629590,
      "label": "score",
      "mode": 1,
      "x": 5,
      "y": 5,
      "visible": true
    }
  ]
}
```

#### tutorial/03-Clone

![Screen Shot1](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/1.png)

Through this example you can learn:
* Clone sprites and destory them.
* Distinguish between sprite variables and shared variables that can access by all sprites.

Here are some codes in [Calf.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/Calf.spx):

```coffee
var (
	id int
)

onClick => {
	clone
}

onCloned => {
	gid++
	...
}
```

When we click the sprite `Calf`, it receives an `onClick` event. Then it calls `clone` to clone itself. And after cloning, the new `Calf` sprite will receive an `onCloned` event.

In `onCloned` event, the new `Calf` sprite uses a variable named `gid`. It doesn't define in [Calf.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/Calf.spx), but in [main.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/main.spx).


Here are all the codes of [main.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/main.spx):

```coffee
var (
	Arrow Arrow
	Calf  Calf
	gid   int
)

run "res", {Title: "Clone and Destory (by XGo)"}
```

All these three variables in [main.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/main.spx) are shared by all sprites. `Arrow` and `Calf` are sprites that exist in this project. `gid` means `global id`. It is used to allocate id for all cloned `Calf` sprites.

Let's back to [Calf.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/Calf.spx) to see the full codes of `onCloned`:

```coffee
onCloned => {
	gid++
	id = gid
	step 50
	say id, 0.5
}
```

It increases `gid` value and assigns it to sprite `id`. This makes all these `Calf` sprites have different `id`. Then the cloned `Calf` moves forward 50 steps and says `id` of itself.

Why these `Calf` sprites need different `id`? Because we want destory one of them by its `id`.

Here are all the codes in [Arrow.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/Arrow.spx):

```coffee
onClick => {
	broadcast "undo", true
	gid--
}
```

When we click `Arrow`, it broadcasts an "undo" message (NOTE: We pass the second parameter `true` to broadcast to indicate we wait all sprites to finish processing this message).

All `Calf` sprites receive this message, but only the last cloned sprite finds its `id` is equal to `gid` then destroys itself. Here are the related codes in [Calf.spx](https://github.com/goplus/spx/blob/main/tutorial/03-Clone/Calf.spx):

```coffee
onMsg "undo", => {
	if id == gid {
		destroy
	}
}
```

#### tutorial/04-Bullet

![Screen Shot1](https://github.com/goplus/spx/blob/main/tutorial/04-Bullet/1.jpg)

Through this example you can learn:
* How to keep a sprite following mouse position.
* How to fire bullets.

It's simple to keep a sprite following mouse position. Here are some related codes in [MyAircraft.spx](https://github.com/goplus/spx/blob/main/tutorial/04-Bullet/MyAircraft.spx):


```coffee
onStart => {
	for {
		# ...
		setXYpos mouseX, mouseY
	}
}
```

Yes, we just need to call `setXYpos mouseX, mouseY` to follow mouse position.

But how to fire bullets? Let's see all codes of [MyAircraft.spx](https://github.com/goplus/spx/blob/main/tutorial/04-Bullet/MyAircraft.spx):

```coffee
onStart => {
	for {
		wait 0.1
		Bullet.clone
		setXYpos mouseX, mouseY
	}
}
```

In this example, `MyAircraft` fires bullets every 0.1 seconds. It just calls `Bullet.clone` to create a new bullet. All the rest things are the responsibility of `Bullet`.

Here are all the codes in [Bullet.spx](https://github.com/goplus/spx/blob/main/tutorial/04-Bullet/Bullet.spx):

```coffee
onCloned => {
	setXYpos MyAircraft.xpos, MyAircraft.ypos+5
	show
	for {
		wait 0.04
		changeYpos 10
		if touching(Edge) {
			destroy
		}
	}
}
```

When a `Bullet` is cloned, it calls `setXYpos MyAircraft.xpos, MyAircraft.ypos+5` to follow `MyAircraft`'s position and shows itself (the default state of a `Bullet` is hidden). Then the `Bullet` moves forward every 0.04 seconds and this is why we see the `Bullet` is flying.

At last, when the `Bullet` touches screen `Edge` or any enemy (in this example we don't have enemies), it destroys itself.

These are all things about firing bullets.
