/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cl_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goplus/xgo/cl/cltest"
)

func codeErrorTest(t *testing.T, msg, src string) {
	cltest.ErrorEx(t, "main", "bar.xgo", msg, src)
}

func codeErrorTestEx(t *testing.T, pkgname, filename, msg, src string) {
	cltest.ErrorEx(t, pkgname, filename, msg, src)
}

func codeErrorTestAst(t *testing.T, pkgname, filename, msg, src string) {
	cltest.ErrorAst(t, pkgname, filename, msg, src)
}

func TestErrTplLit(t *testing.T) {
	codeErrorTest(t, `bar.xgo:1:18: not enough arguments to return
	have ()
	want (interface{})`, "tpl`a = INT => { return }`")
}

func TestErrSendStmt(t *testing.T) {
	codeErrorTest(t, `bar.xgo:3:7: can't send multiple values to a channel`, `
	var a chan int
	a <- 1, 2
`)
}

func TestErrVargCommand(t *testing.T) {
	codeErrorTest(t, `bar.xgo:5:1: not enough arguments in call to Ls
	have ()
	want (int)`, `
func Ls(int) {
}

ls
`)
	codeErrorTest(t, `bar.xgo:8:1: not enough arguments in call to f.Ls
	have ()
	want (int)`, `
type foo int

func (f foo) Ls(int) {
}

var f foo
f.ls
`)
}

func TestErrUnsafe(t *testing.T) {
	codeErrorTest(t, `bar.xgo:2:9: undefined: Sizeof`, `
println Sizeof(0)
`)
}

func TestErrLambdaExpr(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:7:6: too few arguments in lambda expression\n\thave ()\n\twant (int, int)", `

func foo(func(int, int)) {
}

func main() {
	foo(=> {})
}
`)
	codeErrorTest(t,
		"bar.xgo:7:6: too many arguments in lambda expression\n\thave (x, y, z)\n\twant (int, int)", `

func foo(func(int, int)) {
}

func main() {
	foo((x, y, z) => {})
}
`)
	codeErrorTest(t, "bar.xgo:6:8: cannot use lambda literal as type int in field value to Plot", `
type Foo struct {
	Plot int
}
foo := &Foo{
	Plot: x => (x * 2, x * x),
}
`)
	codeErrorTest(t, "bar.xgo:6:8: cannot use lambda literal as type int in field value to Plot", `
type Foo struct {
	Plot int
}
foo := &Foo{
	Plot: x => {
		return x * 2, x * x
	},
}
`)
	codeErrorTest(t,
		"bar.xgo:4:5: cannot use lambda literal as type int in argument to foo", `
func foo(int) {
}
foo(=> {})
`)
	codeErrorTest(t,
		"bar.xgo:4:5: cannot use lambda literal as type func() in argument to foo", `
func foo(func()) {
}
foo => (100)
`)
	codeErrorTest(t,
		"bar.xgo:6:8: cannot use lambda literal as type func() int in field value to Plot", `
type Foo struct {
	Plot func() int
}
foo := &Foo{
	Plot: x => (x * 2, x * x),
}
`)

	codeErrorTest(t,
		"bar.xgo:2:18: cannot use lambda literal as type func() in assignment to foo", `
var foo func() = => (100)
`)
	codeErrorTest(t,
		"bar.xgo:3:7: cannot use lambda literal as type func() in assignment to foo", `
var foo func()
foo = => (100)
`)
	codeErrorTest(t,
		"bar.xgo:2:29: lambda unsupport multiple assignment", `
var foo, foo1 func() = nil, => {}
`)
	codeErrorTest(t,
		"bar.xgo:3:15: lambda unsupport multiple assignment", `
var foo func()
_, foo = nil, => {}
`)
	codeErrorTest(t,
		"bar.xgo:4:9: cannot use lambda expression as type int in return statement", `
func intSeq() int {
	i := 0
	return => {
		i++
		return i
	}
}
`)
	codeErrorTest(t,
		"bar.xgo:6:10: cannot use i (type int) as type string in return argument", `
func intSeq() func() string {
	i := 0
	return => {
		i++
		return i
	}
}
`)
}

func TestErrErrWrap(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:2:2: undefined: a", `func main() {
	a!
}
`)
}

func TestErrVar(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:6:5: assignment mismatch: 1 variables but fmt.Println returns 2 values", `import "fmt"

func main() {
}

var a = fmt.Println(1)
`)
	codeErrorTest(t,
		"bar.xgo:4:5: assignment mismatch: 1 variables but 2 values", `func main() {
}

var a = 1, 2
`)
	codeErrorTest(t,
		"bar.xgo:2:2: undefined: foo", `func main() {
	foo.x = 1
}
`)
	codeErrorTest(t,
		"bar.xgo:2:2: use of builtin len not in function call", `func main() {
	len.x = 1
}
`)
	codeErrorTest(t,
		"bar.xgo:2:10: undefined: foo", `func main() {
	println(foo.x)
}
`)
	codeErrorTest(t,
		"bar.xgo:2:10: use of builtin len not in function call", `func main() {
	println(len.x)
}
`)
	codeErrorTest(t,
		"bar.xgo:2:10: undefined: foo", `func main() {
	println(foo)
}
`)
	codeErrorTest(t,
		"bar.xgo:3:20: use of builtin len not in function call", `package main

func foo(v map[int]len) {
}
`)
	codeErrorTest(t,
		"bar.xgo:5:20: bar is not a type", `package main

var bar = 1

func foo(v map[int]bar) {
}
`)
	codeErrorTest(t,
		"bar.xgo:2:6: use of builtin len not in function call", `func main() {
	new(len)
}
`)
	codeErrorTest(t,
		"bar.xgo:2:2: undefined: foo", `func main() {
	foo = 1
}
`)
	codeErrorTest(t,
		"bar.xgo:2:9: cannot use _ as value", `func main() {
	foo := _
}
`)
	codeErrorTest(t,
		"bar.xgo:2:9: use of builtin len not in function call", `func main() {
	foo := len
}
`)
	codeErrorTest(t,
		"bar.xgo:2:2: println is not a variable", `func main() {
	println = "hello"
}
`)
}

func TestErrImport(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:8:2: confliction: NewEncoding declared both in "encoding/base64" and "encoding/base32"`, `
import (
	. "encoding/base32"
	. "encoding/base64"
)

func foo() {
	NewEncoding("Hi")
}`)
	codeErrorTest(t,
		"bar.xgo:5:2: cannot refer to unexported name os.undefined", `
import "os"

func foo() {
	os.undefined
}`)
	codeErrorTest(t,
		"bar.xgo:5:2: undefined: os.UndefinedObject", `
import "os"

func foo() {
	os.UndefinedObject
}`)
	codeErrorTest(t,
		"bar.xgo:2:13: undefined: testing", `
func foo(t *testing.T) {
}`)
	codeErrorTest(t,
		"bar.xgo:4:12: testing.Verbose is not a type", `
import "testing"

func foo(t testing.Verbose) {
}`)
}

func TestErrConst(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:3:7: a redeclared in this block\n\tprevious declaration at bar.xgo:2:5", `
var a int
const a = 1
`)
	codeErrorTest(t,
		"bar.xgo:4:2: missing value in const declaration", `
const (
	a = iota
	b, c
)
`)
}

func TestErrNewVar(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:3:5: a redeclared in this block\n\tprevious declaration at bar.xgo:2:5", `
var a int
var a string
`)
}

func TestErrDefineVar(t *testing.T) {
	codeErrorTest(t, "bar.xgo:3:1: no new variables on left side of :=\n"+
		"bar.xgo:3:6: cannot use \"Hi\" (type untyped string) as type int in assignment", `
a := 1
a := "Hi"
`)
}

func TestErrAssignMismatchT(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:2:16: cannot use []string{} (type []string) as type string in assignment`, `
var a string = []string{}
`)
	codeErrorTest(t,
		`bar.xgo:2:16: cannot use [2]string{} (type [2]string) as type string in assignment`, `
var a string = [2]string{}
`)
	codeErrorTest(t,
		`bar.xgo:2:16: cannot use map[int]string{} (type map[int]string) as type string in assignment`, `
var a string = map[int]string{}
`)
	codeErrorTest(t,
		`bar.xgo:3:16: cannot use T{} (type T) as type string in assignment`, `
type T struct{}
var a string = T{}
`)
	codeErrorTest(t,
		`bar.xgo:2:16: cannot use func(){} (type func()) as type string in assignment`, `
var a string = func(){}
`)
}

func TestErrAssign(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:8:1: assignment mismatch: 1 variables but bar returns 2 values`, `

func bar() (n int, err error) {
	return
}

x := 1
x = bar()
`)
	codeErrorTest(t,
		`bar.xgo:4:1: assignment mismatch: 1 variables but 2 values`, `

x := 1
x = 1, "Hi"
`)
}

func TestErrReturn(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:4:2: too few arguments to return\n\thave (untyped int)\n\twant (int, error)", `

func foo() (int, error) {
	return 1
}
`)
	codeErrorTest(t,
		"bar.xgo:4:2: too many arguments to return\n\thave (untyped int, untyped int, untyped string)\n\twant (int, error)", `

func foo() (int, error) {
	return 1, 2, "Hi"
}
`)
	codeErrorTest(t,
		`bar.xgo:4:12: cannot use "Hi" (type untyped string) as type error in return argument`, `

func foo() (int, error) {
	return 1, "Hi"
}
`)
	codeErrorTest(t,
		"bar.xgo:8:2: too few arguments to return\n\thave (byte)\n\twant (int, error)", `

func bar() (v byte) {
	return
}

func foo() (int, error) {
	return bar()
}
`)
	codeErrorTest(t,
		"bar.xgo:8:2: too many arguments to return\n\thave (n int, err error)\n\twant (v byte)", `

func bar() (n int, err error) {
	return
}

func foo() (v byte) {
	return bar()
}
`)
	codeErrorTest(t,
		`bar.xgo:8:2: cannot use byte value as type error in return argument`, `

func bar() (n int, v byte) {
	return
}

func foo() (int, error) {
	return bar()
}
`)
	codeErrorTest(t,
		"bar.xgo:4:2: not enough arguments to return\n\thave ()\n\twant (byte)", `

func foo() byte {
	return
}
`)
}

func TestErrForRange(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:4:8: cannot assign type string to a (type int) in range`, `
a := 1
var b []string
for _, a = range b {
}
`)
}

func TestErrInitFunc(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:2:6: func init must have no arguments and no return values`, `
func init(v byte) {
}
`)
}

func TestErrRecv(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:5:9: invalid receiver type a (a is a pointer type)`, `

type a *int

func (p a) foo() {
}
`)
	codeErrorTest(t,
		`bar.xgo:2:9: invalid receiver type error (error is an interface type)`, `
func (p error) foo() {
}
`)
	codeErrorTest(t,
		`bar.xgo:2:9: invalid receiver type []byte ([]byte is not a defined type)`, `
func (p []byte) foo() {
}
`)
	codeErrorTest(t,
		`bar.xgo:2:10: invalid receiver type []byte ([]byte is not a defined type)`, `
func (p *[]byte) foo() {
}
`)
}

func TestErrEnvOp(t *testing.T) {
	codeErrorTest(t, `bar.xgo:2:6: operator $name undefined`, `
echo ${name}
`)
	codeErrorTest(t, `bar.xgo:2:1: operator $id undefined`, `
$id
`)
}

func TestErrStringLit(t *testing.T) {
	codeErrorTest(t, `bar.xgo:2:9: [].string undefined (type []interface{} has no field or method string)`, `
echo "${[]}"
`)
}

func TestErrStructLit(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:39: too many values in struct{x int; y string}{...}`, `
x := 1
a := struct{x int; y string}{1, "Hi", 2}
`)
	codeErrorTest(t,
		`bar.xgo:3:30: too few values in struct{x int; y string}{...}`, `
x := 1
a := struct{x int; y string}{1}
`)
	codeErrorTest(t,
		`bar.xgo:3:33: cannot use x (type int) as type string in value of field y`, `
x := 1
a := struct{x int; y string}{1, x}
`)
	codeErrorTest(t,
		`bar.xgo:2:30: z undefined (type struct{x int; y string} has no field or method z)`, `
a := struct{x int; y string}{z: 1}
`)
}

func TestErrArray(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:8: non-constant array bound n`, `
var n int
var a [n]int
`)
}

func TestErrArrayLit(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:14: cannot use a as index which must be non-negative integer constant`,
		`
a := "Hi"
b := [10]int{a: 1}
`)
	codeErrorTest(t,
		`bar.xgo:3:20: array index 10 out of bounds [0:10]`,
		`
a := "Hi"
b := [10]int{9: 1, 3}
`)
	codeErrorTest(t,
		`bar.xgo:3:16: array index 1 out of bounds [0:1]`,
		`
a := "Hi"
b := [1]int{1, 2}
`)
	codeErrorTest(t,
		`bar.xgo:3:14: array index 12 (value 12) out of bounds [0:10]`,
		`
a := "Hi"
b := [10]int{12: 2}
`)
	codeErrorTest(t,
		`bar.xgo:3:14: cannot use a+"!" (type string) as type int in array literal`,
		`
a := "Hi"
b := [10]int{a+"!"}
`)
	codeErrorTest(t,
		`bar.xgo:3:17: cannot use a (type string) as type int in array literal`,
		`
a := "Hi"
b := [10]int{2: a}
`)
}

func TestErrSliceLit(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:12: cannot use a as index which must be non-negative integer constant`,
		`
a := "Hi"
b := []int{a: 1}
`)
	codeErrorTest(t,
		`bar.xgo:3:12: cannot use a (type string) as type int in slice literal`,
		`
a := "Hi"
b := []int{a}
`)
	codeErrorTest(t,
		`bar.xgo:3:15: cannot use a (type string) as type int in slice literal`,
		`
a := "Hi"
b := []int{2: a}
`)
}

func TestErrMapLit(t *testing.T) {
	codeErrorTest(t, `bar.xgo:4:6: cannot use 1 (type untyped int) as type string in map key`, `
func foo(map[string]string) {}

foo {1: 2}
`)
	codeErrorTest(t, `bar.xgo:2:1: invalid composite literal type int`, `
int{2}
`)
	codeErrorTest(t, `bar.xgo:2:1: missing key in map literal`, `
map[string]int{2}
`)
	codeErrorTest(t, `bar.xgo:2:21: cannot use 1+2 (type untyped int) as type string in map key
bar.xgo:3:27: cannot use "Go" + "+" (type untyped string) as type int in map value`,
		`
a := map[string]int{1+2: 2}
b := map[string]int{"Hi": "Go" + "+"}
`)
	codeErrorTest(t, `bar.xgo:2:13: invalid map literal`, `
var v any = {1:2,1}
`)
	codeErrorTest(t, `bar.xgo:2:21: invalid map literal`, `
var v map[int]int = {1:2,1}
`)
}

func TestErrSlice(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:4:6: cannot slice a (type *byte)`,
		`
var a *byte
x := 1
b := a[x:2]
`)
	codeErrorTest(t,
		`bar.xgo:3:6: cannot slice a (type bool)`,
		`
a := true
b := a[1:2]
`)
	codeErrorTest(t,
		`bar.xgo:3:6: invalid operation a[1:2:5] (3-index slice of string)`,
		`
a := "Hi"
b := a[1:2:5]
`)
}

func TestErrIndex(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:10: assignment mismatch: 2 variables but 1 values`,
		`
a := "Hi"
b, ok := a[1]
`)
	codeErrorTest(t,
		`bar.xgo:3:6: invalid operation: a[1] (type bool does not support indexing)`,
		`
a := true
b := a[1]
`)
}

func TestErrIndexRef(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:1: cannot assign to a[1] (strings are immutable)`,
		`
a := "Hi"
a[1] = 'e'
`)
}

func TestErrStar(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:2: invalid indirect of a (type string)`,
		`
a := "Hi"
*a = 'e'
`)
	codeErrorTest(t,
		`bar.xgo:3:7: invalid indirect of a (type string)`,
		`
a := "Hi"
b := *a
`)
}

func TestErrMember(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:6: a.x undefined (type string has no field or method x)`,
		`
a := "Hello"
b := a.x
`)
}

func TestErrMemberRef(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:3:1: a.x undefined (type string has no field or method x)`,
		`
a := "Hello"
a.x = 1
`)
	codeErrorTest(t,
		`bar.xgo:5:1: a.x undefined (type aaa has no field or method x)`,
		`
type aaa byte

a := aaa(0)
a.x = 1
`)
	codeErrorTest(t,
		`bar.xgo:5:1: a.z undefined (type aaa has no field or method z)`,
		`
type aaa struct {x int; y string}

a := aaa{}
a.z = 1
`)
	codeErrorTest(t,
		`bar.xgo:3:1: a.z undefined (type struct{x int; y string} has no field or method z)`,
		`
a := struct{x int; y string}{}
a.z = 1
`)
}

func TestErrLabel(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:4:1: label foo already defined at bar.xgo:2:1
bar.xgo:2:1: label foo defined and not used`,
		`x := 1
foo:
	i := 1
foo:
	i++
`)
	codeErrorTest(t,
		`bar.xgo:2:6: label foo is not defined`,
		`x := 1
goto foo`)
	codeErrorTest(t,
		`bar.xgo:2:7: label foo is not defined`,
		`x := 1
break foo`)
}

func TestErrBranchStmt(t *testing.T) {
	codeErrorTest(t,
		`bar.xgo:2:2: fallthrough statement out of place`,
		`func foo() {
	fallthrough
}`)
}

func TestErrNoEntrypoint(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:2:2: undefined: println1", `func main() {
	println1 "hello"
}
`)
	codeErrorTest(t,
		"bar.xgo:1:1: undefined: println1", `println1 "hello"`)

	codeErrorTest(t,
		"bar.xgo:2:2: undefined: println1", `
	println1 "hello"
`)
	codeErrorTest(t,
		"bar.xgo:2:2: undefined: println1", `package main
	println1 "hello"
`)
	codeErrorTest(t,
		`bar.xgo:1:9: undefined: abc`,
		`println abc
`)
	codeErrorTestEx(t, "bar", "bar.xgo",
		`bar.xgo:2:9: undefined: abc`,
		`package bar
println abc
`)
}

func TestErrTypeRedefine(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:9:6: Point redeclared in this block\n\tprevious declaration at bar.xgo:5:6",
		`import "fmt"
func (p *Point) String() string {
	return fmt.Sprintf("%v-%v",p.X,p.Y)
}
type Point struct {
	X int
	Y int
}
type Point struct {
	X int
	Y int
}
`)
}

func TestErrSwitchDuplicate(t *testing.T) {
	codeErrorTest(t,
		"bar.xgo:4:7: duplicate case 100 in switch\n\tprevious case at bar.xgo:3:7",
		`var n int
switch n {
	case 100:
	case 100:
}`)
	codeErrorTest(t,
		"bar.xgo:4:7: duplicate case int(100) (value 100) in switch\n\tprevious case at bar.xgo:3:7",
		`var n int
switch n {
	case 100:
	case int(100):
}`)
	codeErrorTest(t,
		"bar.xgo:4:7: duplicate case 50 + 50 (value 100) in switch\n\tprevious case at bar.xgo:3:7",
		`var n int
switch n {
	case 100:
	case 50 + 50:
}`)
	codeErrorTest(t,
		"bar.xgo:5:7: duplicate case int(100) (value 100) in switch\n\tprevious case at bar.xgo:3:7",
		`var n interface{}
switch n {
	case 100:
	case uint(100):
	case int(100):
}`)
	codeErrorTest(t,
		"bar.xgo:4:7: duplicate case 100.0 in switch\n\tprevious case at bar.xgo:3:7",
		`var n interface{}
switch n {
	case 100.0:
	case 100.0:
}`)
	codeErrorTest(t,
		"bar.xgo:5:7: duplicate case v (value 100) in switch\n\tprevious case at bar.xgo:4:7",
		`var n interface{}
const v = 100.0
switch n {
	case 100.0:
	case v:
}`)
	codeErrorTest(t,
		"bar.xgo:5:7: duplicate case v (value \"hello\") in switch\n\tprevious case at bar.xgo:4:7",
		`var n interface{}
const v = "hello"
switch n {
	case "hello":
	case v:
}`)
	codeErrorTest(t,
		`bar.xgo:4:7: duplicate case 100 in switch
	previous case at bar.xgo:3:7
bar.xgo:5:7: duplicate case 50 + 50 (value 100) in switch
	previous case at bar.xgo:3:7`,
		`var n int
switch n {
	case 100:
	case 100:
	case 50 + 50:
}`)
	codeErrorTest(t,
		"bar.xgo:4:2: multiple defaults in switch (first at bar.xgo:3:2)",
		`var n interface{}
switch n {
	default:
	default:
}`)
	codeErrorTest(t, `bar.xgo:4:2: multiple defaults in switch (first at bar.xgo:3:2)
bar.xgo:5:2: multiple defaults in switch (first at bar.xgo:3:2)`,
		`var n interface{}
switch n {
	default:
	default:
	default:
}`)
}

func TestErrTypeSwitchDuplicate(t *testing.T) {
	codeErrorTest(t, `bar.xgo:4:7: duplicate case int in type switch
	previous case at bar.xgo:3:7
bar.xgo:5:7: duplicate case int in type switch
	previous case at bar.xgo:3:7`,
		`var n interface{} = 100
switch n.(type) {
	case int:
	case int:
	case int:
}
`)
	codeErrorTest(t, `bar.xgo:4:7: multiple nil cases in type switch (first at bar.xgo:3:7)
bar.xgo:5:7: multiple nil cases in type switch (first at bar.xgo:3:7)`,
		`var n interface{} = 100
switch n.(type) {
	case nil:
	case nil:
	case nil:
}
`)
	codeErrorTest(t, `bar.xgo:4:2: multiple defaults in type switch (first at bar.xgo:3:2)
bar.xgo:5:2: multiple defaults in type switch (first at bar.xgo:3:2)`,
		`var n interface{} = 100
switch n.(type) {
	default:
	default:
	default:
}
`)
}

func TestErrAutoProperty(t *testing.T) {
	codeErrorTest(t, `bar.xgo:4:11: cannot refer to unexported name fmt.println`, `
import "fmt"

n, err := fmt.println
`)
}

func TestFiledsNameRedecl(t *testing.T) {
	codeErrorTest(t, `bar.xgo:6:2: Id redeclared at bar.xgo:5:2.
	rename one of the fields to resolve the naming conflict.
bar.xgo:7:2: Id redeclared at bar.xgo:5:2.
	rename one of the fields to resolve the naming conflict.
bar.xgo:9:2: name redeclared at bar.xgo:8:2.
	rename one of the fields to resolve the naming conflict.`, `
type Id struct {
}
type A struct {
	Id   int
	Id   string
	Id
	name string
	name string
}
`)
}

func TestErrImportPkg(t *testing.T) {
	root := filepath.Join(runtime.GOROOT(), "src", "fmt2")
	where := "GOROOT"
	ver := runtime.Version()[:6]
	if ver >= "go1.21" {
		where = "std"
	}
	codeErrorTest(t,
		fmt.Sprintf(`bar.xgo:3:2: package fmt2 is not in `+where+` (%v)
`, root), `
import (
	"fmt2"
)
`)

	codeErrorTest(t, `bar.xgo:3:2: no required module provides package github.com/goplus/xgo/fmt2; to add it:
	go get github.com/goplus/xgo/fmt2
`, `
import (
	"github.com/goplus/xgo/fmt2"
)
`)
}

func TestErrClassFileGopx(t *testing.T) {
	codeErrorTestEx(t, "main", "Rect.gox",
		`Rect.gox:5:2: A redeclared at Rect.gox:3:2.
	rename one of the fields to resolve the naming conflict.`, `
var (
	A
	i int
	A
)
type A struct{}
println "hello"
`)
}

func TestErrVarInFunc(t *testing.T) {
	codeErrorTest(t, `bar.xgo:6:10: not enough arguments in call to set
	have (untyped string)
	want (name string, v int)
bar.xgo:7:10: undefined: a`, `
func set(name string, v int) string {
	return name
}
func test() {
	var a = set("box")
	println(a)
}
`)
}

func TestErrInt128(t *testing.T) {
	codeErrorTest(t, `bar.xgo:2:16: cannot use 1<<127 (type untyped int) as type github.com/qiniu/x/xgo/ng.Int128 in assignment`, `
var a int128 = 1<<127
`)
	codeErrorTest(t, `bar.xgo:2:13: cannot convert 1<<127 (untyped int constant 170141183460469231731687303715884105728) to type Int128`, `
a := int128(1<<127)
`)
	codeErrorTest(t, `bar.xgo:2:13: cannot convert -1<<127-1 (untyped int constant -170141183460469231731687303715884105729) to type Int128`, `
a := int128(-1<<127-1)
`)
	codeErrorTest(t, `bar.xgo:3:13: cannot convert b (untyped int constant -170141183460469231731687303715884105729) to type Int128`, `
const b = -1<<127-1
a := int128(b)
`)
}

func TestErrUint128(t *testing.T) {
	codeErrorTest(t, `bar.xgo:2:17: cannot use 1<<128 (type untyped int) as type github.com/qiniu/x/xgo/ng.Uint128 in assignment`, `
var a uint128 = 1<<128
`)
	codeErrorTest(t, `bar.xgo:2:14: cannot convert 1<<128 (untyped int constant 340282366920938463463374607431768211456) to type Uint128`, `
a := uint128(1<<128)
`)
	codeErrorTest(t, `bar.xgo:2:17: cannot use -1 (type untyped int) as type github.com/qiniu/x/xgo/ng.Uint128 in assignment`, `
var a uint128 = -1
`)
	codeErrorTest(t, `bar.xgo:2:14: cannot convert -1 (untyped int constant -1) to type Uint128`, `
a := uint128(-1)
`)
	codeErrorTest(t, `bar.xgo:3:14: cannot convert b (untyped int constant -1) to type Uint128`, `
const b = -1
a := uint128(b)
`)
}

func TestErrCompileFunc(t *testing.T) {
	codeErrorTest(t, "bar.xgo:2:1: compile `printf(\"%+v\\n\", int32)`: unreachable", `
printf("%+v\n", int32)
`)
}

func TestToTypeError(t *testing.T) {
	codeErrorTestAst(t, "main", "bar.xgo", `bar.xgo:3:3: toType unexpected: *ast.BadExpr`, `
type
a := 1
`)
}

func TestCompileExprError(t *testing.T) {
	codeErrorTestAst(t, "main", "bar.go", `bar.go:5:1: compileExpr failed: unknown - *ast.BadExpr`, `
func Foo(){}
func _() {
	Foo(
}
`)
	codeErrorTestAst(t, "main", "bar.go", `bar.go:3:2: compileExprLHS failed: unknown - *ast.StructType`, `
func _() {
	struct() = nil
}
`)
}

func TestOverloadFuncDecl(t *testing.T) {
	codeErrorTest(t, "bar.xgo:3:2: invalid func (foo).mulInt", `
func mul = (
	(foo).mulInt
)
`)
	codeErrorTest(t, "bar.xgo:2:7: invalid recv type *foo", `
func (*foo).mul = (
	(foo).mulInt
)
`)
	codeErrorTest(t, "bar.xgo:3:2: invalid recv type (foo2)", `
func (foo).mul = (
	(foo2).mulInt
)
`)
	codeErrorTest(t, "bar.xgo:3:2: invalid method mulInt", `
func (foo).mul = (
	mulInt
)
`)
	codeErrorTest(t, "bar.xgo:3:2: invalid recv type (**foo)", `
func (foo).mul = (
	(**foo).mulInt
)
`)
	codeErrorTest(t, `bar.xgo:3:9: unknown func ("ok")`, `
func mul = (
	println("ok")
)
`)
	codeErrorTest(t, "bar.xgo:3:2: invalid method func(){}", `
func (foo).mul = (
	func(){}
)
`)
	codeErrorTest(t, "bar.xgo:2:12: invalid overload operator ++", `
func (foo).++ = (
	mulInt
)
`)
}

func TestCompositeLitError(t *testing.T) {
	codeErrorTest(t, `bar.xgo:2:22: cannot use 3.14 (type untyped float) as type int in slice literal`, `
var a [][]int = {[10,3.14,200],[100,200]}
echo a
`)
	codeErrorTest(t, `bar.xgo:2:17: cannot use lambda literal as type int in assignment`, `
var a []int = {(x => x)}
echo a
`)
	codeErrorTest(t, `bar.xgo:2:35: cannot use x (type int) as type string in return argument`, `
var a []func(int) string = {(x => x)}
echo a
`)
	codeErrorTest(t, `bar.xgo:2:27: cannot use lambda literal as type int in assignment to "A"`, `
var a map[any]int = {"A": x => x}
`)
	codeErrorTest(t, `bar.xgo:2:45: cannot use x (type int) as type string in return argument`, `
var a map[any]func(int) string = {"A": x => x}
`)
	codeErrorTest(t, `bar.xgo:2:24: cannot use lambda literal as type int in field value`, `
var a = struct{v int}{(x => x)}
`)
	codeErrorTest(t, `bar.xgo:2:27: cannot use lambda literal as type int in field value to v`, `
var a = struct{v int}{v: (x => x)}
`)
}
