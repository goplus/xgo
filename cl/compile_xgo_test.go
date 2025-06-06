/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"strings"
	"testing"
)

func TestArrowOp(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (a foo) -> (b foo) {
	println "a -> b"
}

func (a foo) <> (b foo) {
	println "a <> b"
}
`, `package main

import "fmt"

type foo struct {
}

func (a foo) Gop_PointTo(b foo) {
	fmt.Println("a -> b")
}
func (a foo) Gop_PointBi(b foo) {
	fmt.Println("a <> b")
}
`)
}

func TestMapLit(t *testing.T) {
	gopClTest(t, `
func foo(map[string]string) {}

foo {}
`, `package main

func foo(map[string]string) {
}
func main() {
	foo(map[string]string{})
}
`)
}

func TestMayBuiltinDelete(t *testing.T) {
	gopClTest(t, `
func Delete(a int) {}
func Foo(m map[string]int) {
	delete(m, "a")
}

delete 10
`, `package main

func Delete(a int) {
}
func Foo(m map[string]int) {
	delete(m, "a")
}
func main() {
	Delete(10)
}
`)
}

func TestVargCommand(t *testing.T) {
	gopClTest(t, `
type foo int

func (f foo) Ls(args ...string) {
}

var f foo
f.ls
`, `package main

type foo int

func (f foo) Ls(args ...string) {
}

var f foo

func main() {
	f.Ls()
}
`)
}

func TestCommandInPkg(t *testing.T) {
	gopClTest(t, `
func Ls(args ...string) {
}

ls
`, `package main

func Ls(args ...string) {
}
func main() {
	Ls()
}
`)
}

func TestFuncAlias(t *testing.T) {
	gopClTest(t, `
func Foo(a ...int) {}

foo 100
foo
`, `package main

func Foo(a ...int) {
}
func main() {
	Foo(100)
	Foo()
}
`)
}

func TestOverloadOp(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (a *foo) + (b *foo) *foo {
	println("a + b")
	return &foo{}
}

func (a foo) - (b foo) foo {
	println("a - b")
	return foo{}
}

func -(a foo) {
	println("-a")
}

func ++(a foo) {
	println("a++")
}

func (a foo) != (b foo) bool{
	println("a!=b")
	return true
}

var a, b foo
var c = a - b
var d = -a       // TODO: -a have no return value!
var e = a!=b
`, `package main

import "fmt"

type foo struct {
}

func (a *foo) Gop_Add(b *foo) *foo {
	fmt.Println("a + b")
	return &foo{}
}
func (a foo) Gop_Sub(b foo) foo {
	fmt.Println("a - b")
	return foo{}
}
func (a foo) Gop_NE(b foo) bool {
	fmt.Println("a!=b")
	return true
}
func (a foo) Gop_Neg() {
	fmt.Println("-a")
}
func (a foo) Gop_Inc() {
	fmt.Println("a++")
}

var a, b foo
var c = (foo).Gop_Sub(a, b)
var d = a.Gop_Neg()
var e = (foo).Gop_NE(a, b)
`)
}

func TestOverloadOp2(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (a foo) mulInt(b int) (ret foo) {
	return
}

func (a foo) mulFoo(b foo) (ret foo) {
	return
}

func intMulFoo(a int, b foo) (ret foo) {
	return
}

func (foo).* = (
	(foo).mulInt
	(foo).mulFoo
	intMulFoo
)

var a, b foo

println a * 10
println a * b
println 10 * a
`, `package main

import "fmt"

type foo struct {
}

const Gopo__foo__Gop_Mul = ".mulInt,.mulFoo,intMulFoo"

func (a foo) mulInt(b int) (ret foo) {
	return
}
func (a foo) mulFoo(b foo) (ret foo) {
	return
}
func intMulFoo(a int, b foo) (ret foo) {
	return
}

var a, b foo

func main() {
	fmt.Println((foo).mulInt(a, 10))
	fmt.Println((foo).mulFoo(a, b))
	fmt.Println(intMulFoo(10, a))
}
`)
}

func TestOverloadMethod(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (a *foo) mulInt(b int) *foo {
	println "mulInt"
	return a
}

func (a *foo) mulFoo(b *foo) *foo {
	println "mulFoo"
	return a
}

func (foo).mul = (
	(foo).mulInt
	(foo).mulFoo
)

var a, b foo
var c = a.mul(100)
var d = a.mul(c)
`, `package main

import "fmt"

type foo struct {
}

const Gopo_foo_mul = ".mulInt,.mulFoo"

func (a *foo) mulInt(b int) *foo {
	fmt.Println("mulInt")
	return a
}
func (a *foo) mulFoo(b *foo) *foo {
	fmt.Println("mulFoo")
	return a
}

var a, b foo
var c = a.mulInt(100)
var d = a.mulFoo(c)
`)
}

func TestOverloadFunc(t *testing.T) {
	gopClTest(t, `
func add = (
	func(a, b int) int {
		return a + b
	}
	func(a, b string) string {
		return a + b
	}
)

println add(100, 7)
println add("Hello", "World")
`, `package main

import "fmt"

func add__0(a int, b int) int {
	return a + b
}
func add__1(a string, b string) string {
	return a + b
}
func main() {
	fmt.Println(add__0(100, 7))
	fmt.Println(add__1("Hello", "World"))
}
`)
}

func TestOverloadFunc2(t *testing.T) {
	gopClTest(t, `
func mulInt(a, b int) int {
	return a * b
}

func mulFloat(a, b float64) float64 {
	return a * b
}

func mul = (
	mulInt
	mulFloat
)

println mul(100, 7)
println mul(1.2, 3.14)
`, `package main

import "fmt"

const Gopo_mul = "mulInt,mulFloat"

func mulInt(a int, b int) int {
	return a * b
}
func mulFloat(a float64, b float64) float64 {
	return a * b
}
func main() {
	fmt.Println(mulInt(100, 7))
	fmt.Println(mulFloat(1.2, 3.14))
}
`)
}

func TestOverloadFunc3(t *testing.T) {
	gopClTest(t, `
func addInt(a, b int) int {
	return a + b
}

func addFloat(a, b float64) float64 {
	return a * b
}

func add = (
	func (a,b string) string { return a + b }
	addInt
	addFloat
)

println add(100, 7)
println add(1.2, 3.14)
`, `package main

import "fmt"

const Gopo_add = ",addInt,addFloat"

func add__0(a string, b string) string {
	return a + b
}
func addInt(a int, b int) int {
	return a + b
}
func addFloat(a float64, b float64) float64 {
	return a * b
}
func main() {
	fmt.Println(addInt(100, 7))
	fmt.Println(addFloat(1.2, 3.14))
}
`)
}

func TestOverload(t *testing.T) {
	gopClTest(t, `
import "github.com/goplus/xgo/cl/internal/overload/foo"

type Mesh struct {
}

func (p *Mesh) Name() string {
	return "hello"
}

var (
	m1 = &Mesh{}
	m2 = &Mesh{}
)

foo.onKey "hello", => {
}
foo.onKey "hello", key => {
}
foo.onKey ["1"], => {
}
foo.onKey ["2"], key => {
}
foo.onKey [m1, m2], => {
}
foo.onKey [m1, m2], key => {
}
foo.onKey ["a"], ["b"], key => {
}
foo.onKey ["a"], [m1, m2], key => {
}
foo.onKey ["a"], nil, key => {
}
foo.onKey 100, 200

n := &foo.N{}
n.onKey "hello", => {
}
n.onKey "hello", key => {
}
n.onKey ["1"], => {
}
n.onKey ["2"], key => {
}
n.onKey [m1, m2], => {
}
n.onKey [m1, m2], key => {
}
n.onKey ["a"], ["b"], key => {
}
n.onKey ["a"], [m1, m2], key => {
}
n.onKey ["a"], nil, key => {
}
n.onKey 100, 200
`, `package main

import "github.com/goplus/xgo/cl/internal/overload/foo"

type Mesh struct {
}

func (p *Mesh) Name() string {
	return "hello"
}

var m1 = &Mesh{}
var m2 = &Mesh{}

func main() {
	foo.OnKey__0("hello", func() {
	})
	foo.OnKey__1("hello", func(key string) {
	})
	foo.OnKey__2([]string{"1"}, func() {
	})
	foo.OnKey__3([]string{"2"}, func(key string) {
	})
	foo.OnKey__4([]foo.Mesher{m1, m2}, func() {
	})
	foo.OnKey__5([]foo.Mesher{m1, m2}, func(key foo.Mesher) {
	})
	foo.OnKey__6([]string{"a"}, []string{"b"}, func(key string) {
	})
	foo.OnKey__7([]string{"a"}, []foo.Mesher{m1, m2}, func(key string) {
	})
	foo.OnKey__6([]string{"a"}, nil, func(key string) {
	})
	foo.OnKey__8(100, 200)
	n := &foo.N{}
	n.OnKey__0("hello", func() {
	})
	n.OnKey__1("hello", func(key string) {
	})
	n.OnKey__2([]string{"1"}, func() {
	})
	n.OnKey__3([]string{"2"}, func(key string) {
	})
	n.OnKey__4([]foo.Mesher{m1, m2}, func() {
	})
	n.OnKey__5([]foo.Mesher{m1, m2}, func(key foo.Mesher) {
	})
	n.OnKey__6([]string{"a"}, []string{"b"}, func(key string) {
	})
	n.OnKey__7([]string{"a"}, []foo.Mesher{m1, m2}, func(key string) {
	})
	n.OnKey__6([]string{"a"}, nil, func(key string) {
	})
	n.OnKey__8(100, 200)
}
`)
}

func TestMixedOverload(t *testing.T) {
	gopMixedClTest(t, "main", `
package main

type Mesher interface {
	Name() string
}

type N struct {
}

func (m *N) OnKey__0(a string, fn func()) {
}

func (m *N) OnKey__1(a string, fn func(key string)) {
}

func (m *N) OnKey__2(a []string, fn func()) {
}

func (m *N) OnKey__3(a []string, fn func(key string)) {
}

func (m *N) OnKey__4(a []Mesher, fn func()) {
}

func (m *N) OnKey__5(a []Mesher, fn func(key Mesher)) {
}

func (m *N) OnKey__6(a []string, b []string, fn func(key string)) {
}

func (m *N) OnKey__7(a []string, b []Mesher, fn func(key string)) {
}

func (m *N) OnKey__8(x int, y int) {
}


func OnKey__0(a string, fn func()) {
}

func OnKey__1(a string, fn func(key string)) {
}

func OnKey__2(a []string, fn func()) {
}

func OnKey__3(a []string, fn func(key string)) {
}

func OnKey__4(a []Mesher, fn func()) {
}

func OnKey__5(a []Mesher, fn func(key Mesher)) {
}

func OnKey__6(a []string, b []string, fn func(key string)) {
}

func OnKey__7(a []string, b []Mesher, fn func(key string)) {
}

func OnKey__8(x int, y int) {
}

func OnKey__9(a, b string, fn ...func(x int) int) {
}

func OnKey__a(a, b string, v ...int) {
}
`, `
type Mesh struct {
}

func (p *Mesh) Name() string {
	return "hello"
}

var (
	m1 = &Mesh{}
	m2 = &Mesh{}
)

OnKey "hello", => {
}
OnKey "hello", key => {
}
OnKey ["1"], => {
}
OnKey ["2"], key => {
}
OnKey [m1, m2], => {
}
OnKey [m1, m2], key => {
}
OnKey ["a"], ["b"], key => {
}
OnKey ["a"], [m1, m2], key => {
}
OnKey ["a"], nil, key => {
}
OnKey 100, 200
OnKey "a", "b", x => x * x, x => {
	return x * 2
}
OnKey "a", "b", 1, 2, 3
OnKey("a", "b", [1, 2, 3]...)

n := &N{}
n.onKey "hello", => {
}
n.onKey "hello", key => {
}
n.onKey ["1"], => {
}
n.onKey ["2"], key => {
}
n.onKey [m1, m2], => {
}
n.onKey [m1, m2], key => {
}
n.onKey ["a"], ["b"], key => {
}
n.onKey ["a"], [m1, m2], key => {
}
n.onKey ["a"], nil, key => {
}
n.onKey 100, 200
`, `package main

type Mesh struct {
}

func (p *Mesh) Name() string {
	return "hello"
}

var m1 = &Mesh{}
var m2 = &Mesh{}

func main() {
	OnKey__0("hello", func() {
	})
	OnKey__1("hello", func(key string) {
	})
	OnKey__2([]string{"1"}, func() {
	})
	OnKey__3([]string{"2"}, func(key string) {
	})
	OnKey__4([]Mesher{m1, m2}, func() {
	})
	OnKey__5([]Mesher{m1, m2}, func(key Mesher) {
	})
	OnKey__6([]string{"a"}, []string{"b"}, func(key string) {
	})
	OnKey__7([]string{"a"}, []Mesher{m1, m2}, func(key string) {
	})
	OnKey__6([]string{"a"}, nil, func(key string) {
	})
	OnKey__8(100, 200)
	OnKey__9("a", "b", func(x int) int {
		return x * x
	}, func(x int) int {
		return x * 2
	})
	OnKey__a("a", "b", 1, 2, 3)
	OnKey__a("a", "b", []int{1, 2, 3}...)
	n := &N{}
	n.OnKey__0("hello", func() {
	})
	n.OnKey__1("hello", func(key string) {
	})
	n.OnKey__2([]string{"1"}, func() {
	})
	n.OnKey__3([]string{"2"}, func(key string) {
	})
	n.OnKey__4([]Mesher{m1, m2}, func() {
	})
	n.OnKey__5([]Mesher{m1, m2}, func(key Mesher) {
	})
	n.OnKey__6([]string{"a"}, []string{"b"}, func(key string) {
	})
	n.OnKey__7([]string{"a"}, []Mesher{m1, m2}, func(key string) {
	})
	n.OnKey__6([]string{"a"}, nil, func(key string) {
	})
	n.OnKey__8(100, 200)
}
`)
}

func TestMixedOverloadOp(t *testing.T) {
	gopMixedClTest(t, "main", `package main

import "fmt"

type foo struct {
}

func (a *foo) Gop_Add(b *foo) *foo {
	fmt.Println("a + b")
	return &foo{}
}
func (a foo) Gop_Sub(b foo) foo {
	fmt.Println("a - b")
	return foo{}
}
func (a foo) Gop_NE(b foo) bool {
	fmt.Println("a!=b")
	return true
}
func (a foo) Gop_Neg() *foo {
	fmt.Println("-a")
	return &foo{}
}
func (a foo) Gop_Inc() {
	fmt.Println("a++")
}
`, `
var a, b foo
var c = a - b
var d = -a
var e = a!=b
`, `package main

var a, b foo
var c = (foo).Gop_Sub(a, b)
var d = a.Gop_Neg()
var e = (foo).Gop_NE(a, b)
`)
}

func TestMixedVector3(t *testing.T) {
	gopMixedClTest(t, "main", `package main
type Vector3 struct {
	x, y, z float64
}
func (a Vector3) Gop_Add__0(n int) Vector3 {
	return Vector3{}
}
func (a Vector3) Gop_Add__1(n float64) Vector3 {
	return Vector3{}
}
func (a Vector3) Gop_Add__2(n Vector3) Vector3 {
	return Vector3{}
}
func (a *Vector3) Gop_AddAssign(n Vector3) {
}

func (a Vector3) Gop_Rcast__0() int {
	return 0
}
func (a Vector3) Gop_Rcast__1() float64 {
	return 0
}

func Vector3_Cast__0(x int) Vector3 {
	return Vector3{}
}
func Vector3_Cast__1(x float64) Vector3 {
	return Vector3{}
}
func Vector3_Init__0(x int) Vector3 {
	return Vector3{}
}
func Vector3_Init__1(x float64) Vector3 {
	return Vector3{}
}
`, `
var a Vector3
var b int
var c float64
_ = a+b
_ = a+100
_ = a+c
_ = 100+a
_ = Vector3(b)+a
_ = b+int(a)
a += b
a += c
`, `package main

var a Vector3
var b int
var c float64

func main() {
	_ = (Vector3).Gop_Add__0(a, b)
	_ = (Vector3).Gop_Add__0(a, 100)
	_ = (Vector3).Gop_Add__1(a, c)
	_ = (Vector3).Gop_Add__2(Vector3_Init__0(100), a)
	_ = (Vector3).Gop_Add__2(Vector3_Cast__0(b), a)
	_ = b + a.Gop_Rcast__0()
	a.Gop_AddAssign(Vector3_Init__0(b))
	a.Gop_AddAssign(Vector3_Init__1(c))
}
`)
}

func TestMixedInterfaceOverload(t *testing.T) {
	gopMixedClTest(t, "main", `
package main

type N[T any] struct {
	v T
}

func (m *N[T]) OnKey__0(a string, fn func()) {
}

func (m *N[T]) OnKey__1(a string, fn func(key string)) {
}

func (m *N[T]) OnKey__2(a []string, fn func()) {
}

func (m *N[T]) OnKey__3(a []string, fn func(key string)) {
}

type I interface {
	OnKey__0(a string, fn func())
	OnKey__1(a string, fn func(key string))
	OnKey__2(a []string, fn func())
	OnKey__3(a []string, fn func(key string))
}
`, `
n := &N[int]{}
n.onKey "1", => {
}
keys := ["1","2"]
n.onKey keys, key => {
	println key
}
n.onKey keys, => {
	println keys
}

var i I = n
i.onKey "1", key => {
	println key
}
i.onKey ["1","2"], key => {
	println key
}
`, `package main

import "fmt"

func main() {
	n := &N[int]{}
	n.OnKey__0("1", func() {
	})
	keys := []string{"1", "2"}
	n.OnKey__3(keys, func(key string) {
		fmt.Println(key)
	})
	n.OnKey__2(keys, func() {
		fmt.Println(keys)
	})
	var i I = n
	i.OnKey__1("1", func(key string) {
		fmt.Println(key)
	})
	i.OnKey__3([]string{"1", "2"}, func(key string) {
		fmt.Println(key)
	})
}
`)
}

func TestMixedOverloadCommand(t *testing.T) {
	gopMixedClTest(t, "main", `package main

func Test__0() {
}
func Test__1(n int) {
}
type N struct {
}
func (p *N) Test__0() {
}
func (p *N) Test__1(n int) {
}`, `
Test
Test 100
var n N
n.test
n.test 100
`, `package main

func main() {
	Test__0()
	Test__1(100)
	var n N
	n.Test__0()
	n.Test__1(100)
}
`)
}

func TestOverloadNamed(t *testing.T) {
	var excepted string
	if gotypesalias {
		excepted = `package main

import "github.com/goplus/xgo/cl/internal/overload/bar"

var a bar.Var__0[int]
var b bar.Var__1[bar.M]

func main() {
	c := bar.Gopx_Var_Cast__0[string]()
	d := bar.Gopx_Var_Cast__1[bar.M]()
}
`
	} else {
		excepted = `package main

import "github.com/goplus/xgo/cl/internal/overload/bar"

var a bar.Var__0[int]
var b bar.Var__1[map[string]any]

func main() {
	c := bar.Gopx_Var_Cast__0[string]()
	d := bar.Gopx_Var_Cast__1[map[string]any]()
}
`
	}
	gopClTest(t, `
import "github.com/goplus/xgo/cl/internal/overload/bar"

var a bar.Var[int]
var b bar.Var[bar.M]
c := bar.Var(string)
d := bar.Var(bar.M)
`, excepted)
}

func TestMixedOverloadNamed(t *testing.T) {
	var expect string
	if gotypesalias {
		expect = `package main

var a Var__0[int]
var b Var__1[M]

func main() {
	c := Gopx_Var_Cast__0[string]()
	d := Gopx_Var_Cast__1[M]()
}
`
	} else {
		expect = `package main

var a Var__0[int]
var b Var__1[map[string]interface{}]

func main() {
	c := Gopx_Var_Cast__0[string]()
	d := Gopx_Var_Cast__1[map[string]interface{}]()
}
`
	}
	gopMixedClTest(t, "main", `package main

type M = map[string]any

type basetype interface {
	string | int | bool | float64
}

type Var__0[T basetype] struct {
	val T
}

type Var__1[T map[string]any] struct {
	val T
}

func Gopx_Var_Cast__0[T basetype]() *Var__0[T] {
	return new(Var__0[T])
}

func Gopx_Var_Cast__1[T map[string]any]() *Var__1[T] {
	return new(Var__1[T])
}
`, `
var a Var[int]
var b Var[M]
c := Var(string)
d := Var(M)
`, expect)
}

func TestStringLitBasic(t *testing.T) {
	gopClTest(t, `echo "$$"`, `package main

import "fmt"

func main() {
	fmt.Println("$")
}
`)
}

func TestStringLitVar(t *testing.T) {
	gopClTest(t, `
x := 1
println "Hi, " + "a${x}b"`, `package main

import (
	"fmt"
	"github.com/qiniu/x/stringutil"
	"strconv"
)

func main() {
	x := 1
	fmt.Println("Hi, " + stringutil.Concat("a", strconv.Itoa(x), "b"))
}
`)
}

func TestFileOpen(t *testing.T) {
	gopClTest(t, `
for line <- open("foo.txt")! {
	println line
}
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
	"github.com/qiniu/x/osx"
	"os"
)

func main() {
	for _xgo_it := osx.EnumLines(func() (_xgo_ret *os.File) {
		var _xgo_err error
		_xgo_ret, _xgo_err = os.Open("foo.txt")
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "open(\"foo.txt\")", "/foo/bar.xgo", 2, "main.main")
			panic(_xgo_err)
		}
		return
	}()); ; {
		var _xgo_ok bool
		line, _xgo_ok := _xgo_it.Next()
		if !_xgo_ok {
			break
		}
		fmt.Println(line)
	}
}
`)
}

func TestMixedGo(t *testing.T) {
	var expect string
	if gotypesalias {
		expect = `package main

var a [10]int
var b string = f(n)
var c foo2
var d int = c.v
var e = foo3{}
var x string = c.Str()
`
	} else {
		expect = `package main

var a [10]int
var b string = f(n)
var c foo
var d int = c.v
var e = foo3{}
var x string = c.Str()
`
	}
	gopMixedClTest(t, "main", `package main

import "strconv"

const n = 10

func f(v int) string {
	return strconv.Itoa(v)
}

type foo struct {
	v int
}

func (a foo) _() {
}

func (a foo) Str() string {
	return f(a.v)
}

func (a *foo) Bar() int {
	return 0
}

type foo2 = foo
type foo3 foo2
`, `
var a [n]int
var b string = f(n)
var c foo2
var d int = c.v
var e = foo3{}
var x string = c.str
`, expect, true)
	gopMixedClTest(t, "main", `package main
type Point struct {
	X int
	Y int
}
`, `
type T struct{}
println(&T{},&Point{10,20})
`, `package main

import "fmt"

type T struct {
}

func main() {
	fmt.Println(&T{}, &Point{10, 20})
}
`, false)
}

func TestTypeAsParamsFunc(t *testing.T) {
	gopMixedClTest(t, "main", `
package main

import (
	"fmt"
	"reflect"
)

type basetype interface {
	int | string
}

func Gopx_Row__0[T basetype](name string) {
}

func Gopx_Row__1[Array any](v int) {
}

func Gopx_Col[T any](name string) {
	fmt.Printf("%v: %s\n", reflect.TypeOf((*T)(nil)).Elem(), name)
}

type Table struct {
}

func Gopt_Table_Gopx_Col__0[T basetype](p *Table, name string) {
}

func Gopt_Table_Gopx_Col__1[Array any](p *Table, v int) {
}
`, `
var tbl *Table

col string, "name"
col int, "age"

row string, 100

tbl.col string, "foo"
tbl.col int, 100
`, `package main

var tbl *Table

func main() {
	Gopx_Col[string]("name")
	Gopx_Col[int]("age")
	Gopx_Row__1[string](100)
	Gopt_Table_Gopx_Col__0[string](tbl, "foo")
	Gopt_Table_Gopx_Col__1[int](tbl, 100)
}
`)
}

func TestYaptest(t *testing.T) {
	gopMixedClTest(t, "main", `package main

import (
	"github.com/goplus/xgo/cl/internal/test"
)

type Class struct {
	test.Case
}
`, `var c Class
var a int

c.match a, "b"
`, `package main

import "github.com/goplus/xgo/cl/internal/test"

var c Class
var a int

func main() {
	test.Gopt_Case_MatchAny(c, a, "b")
}
`)
}

func testRangeExpr(t *testing.T, codeTpl, expect string) {
	for k, s := range []string{" <- ", " := range ", " = range "} {
		if k == 2 {
			codeTpl = "i:=0\n" + codeTpl
			expect = strings.Replace(expect, "for i := ", "i := 0\n\tfor i = ", -1)
		}
		gopClTest(t, strings.Replace(codeTpl, "$", s, -1), expect)
	}
}

func TestRangeExpr(t *testing.T) {
	testRangeExpr(t, `
for i $ :10 {
	println(i)
}`, `package main

import "fmt"

func main() {
	for i := 0; i < 10; i += 1 {
		fmt.Println(i)
	}
}
`)
	testRangeExpr(t, `
for i $ 1:10:3 {
	println(i)
}`, `package main

import "fmt"

func main() {
	for i := 1; i < 10; i += 3 {
		fmt.Println(i)
	}
}
`)
}

func TestRangeExpr2(t *testing.T) {
	testRangeExpr(t, `
for i $ 1:10:2 {
	println(i)
}`, `package main

import "fmt"

func main() {
	for i := 1; i < 10; i += 2 {
		fmt.Println(i)
	}
}
`)
}

func TestRangeExpr3(t *testing.T) {
	testRangeExpr(t, `
for i $ 1:10 {
	println(i)
}`, `package main

import "fmt"

func main() {
	for i := 1; i < 10; i += 1 {
		fmt.Println(i)
	}
}
`)
}

func TestRangeExpr4(t *testing.T) {
	testRangeExpr(t, `
for i $ :10:2 {
	println(i)
}`, `package main

import "fmt"

func main() {
	for i := 0; i < 10; i += 2 {
		fmt.Println(i)
	}
}
`)
}

func TestRangeExpr5(t *testing.T) {
	gopClTest(t, `
for range :10 {
	println("Hi")
}`, `package main

import "fmt"

func main() {
	for _xgo_k := 0; _xgo_k < 10; _xgo_k += 1 {
		fmt.Println("Hi")
	}
}
`)
}

func TestRangeExpr6(t *testing.T) {
	gopClTest(t, `
for _ <- :10 {
	println("Hi")
}`, `package main

import "fmt"

func main() {
	for _xgo_k := 0; _xgo_k < 10; _xgo_k += 1 {
		fmt.Println("Hi")
	}
}
`)
}

func testRangeExpr8(t *testing.T, codeTpl, expect string) {
	for _, s := range []string{" <- ", " := range "} {
		gopClTest(t, strings.Replace(codeTpl, "$", s, -1), expect)
	}
}

func TestRangeExpr8(t *testing.T) {
	testRangeExpr8(t, `
type T struct{}

func (t T) start() int {
	return 0
}
func (t T) end() int{
	return 3
}
func (t T) step() int{
	return 1
}

t:=T{}

for i <- t.start():t.end():t.step(){
	println i
}
`, `package main

import "fmt"

type T struct {
}

func (t T) start() int {
	return 0
}
func (t T) end() int {
	return 3
}
func (t T) step() int {
	return 1
}
func main() {
	t := T{}
	for i, _xgo_end, _xgo_step := t.start(), t.end(), t.step(); i < _xgo_end; i += _xgo_step {
		fmt.Println(i)
	}
}
`)
}

func TestRangeExpr9(t *testing.T) {
	testRangeExpr8(t, `
type T struct{}

func (t T) start() int {
	return 0
}
func (t T) end() int{
	return 3
}
func (t T) step() int{
	return 1
}

t:=T{}
i:=0
for i =range t.start():t.end():t.step(){
	println i
}
`, `package main

import "fmt"

type T struct {
}

func (t T) start() int {
	return 0
}
func (t T) end() int {
	return 3
}
func (t T) step() int {
	return 1
}
func main() {
	t := T{}
	i := 0
	for _xgo_k, _xgo_end, _xgo_step := t.start(), t.end(), t.step(); _xgo_k < _xgo_end; _xgo_k += _xgo_step {
		i = _xgo_k
		fmt.Println(i)
	}
}
`)
}

func TestRangeExpr10(t *testing.T) {
	gopClTest(t, `
for :10 {
	echo "Hi"
}
`, `package main

import "fmt"

func main() {
	for _xgo_k := 0; _xgo_k < 10; _xgo_k += 1 {
		fmt.Println("Hi")
	}
}
`)
}

func Test_RangeExpressionIf_Issue1243(t *testing.T) {
	gopClTest(t, `
for i <- :10, i%3 == 0 {
	println i
}`, `package main

import "fmt"

func main() {
	for i := 0; i < 10; i += 1 {
		if i%3 == 0 {
			fmt.Println(i)
		}
	}
}
`)
}

func TestStaticMethod(t *testing.T) {
	gopClTest(t, `
type foo int

func foo.New(a int) *foo {
	return new(foo)
}

func foo._add() *foo {
	return new(foo)
}

a := foo.new(100)
`, `package main

type foo int

func Gops_foo_New(a int) *foo {
	return new(foo)
}
func Gops__foo___add() *foo {
	return new(foo)
}
func main() {
	a := Gops_foo_New(100)
}
`)
}

func TestOverlodOptions(t *testing.T) {
	gopMixedClTest(t, "main", `
package main

type PlayOptions struct {
	Action int
	Wait   bool
	Loop   bool
}
type Game struct {
}
func (g *Game) Play__0(options *PlayOptions) {
}
func (g *Game) Play__1(name string, options *PlayOptions) {
}
`, `
g := &Game{}
g.play "work", { Action: 0, Loop: true }
`, `package main

func main() {
	g := &Game{}
	g.Play__1("work", &PlayOptions{Action: 0, Loop: true})
}
`)
}

func TestEmbedField(t *testing.T) {
	gopClTest(t, `package main

type Info struct {
	id int
}
type T struct {
	Info
	id string
}
func demo(t *T) {
	t.id = "0"
}
func main() {
}
`, `package main

type Info struct {
	id int
}
type T struct {
	Info
	id string
}

func demo(t *T) {
	t.id = "0"
}
func main() {
}
`)
}

func TestOverloadUntyped(t *testing.T) {
	gopMixedClTest(t, "main", `
package main

type specialObj int
type SpriteName string

type SpriteImpl struct {
}

func (p *SpriteImpl) turn(v any) {
}
func (p *SpriteImpl) TurnTo__0(sprite *SpriteImpl) {
}
func (p *SpriteImpl) TurnTo__1(sprite SpriteName) {
}
func (p *SpriteImpl) TurnTo__2(obj specialObj) {
}
func (p *SpriteImpl) TurnTo__3(degree float64) {
}
`, `
p := &SpriteImpl{}
p.turnTo 180.0
p.turnTo 180.1
`, `package main

func main() {
	p := &SpriteImpl{}
	p.TurnTo__2(180.0)
	p.TurnTo__3(180.1)
}
`)
}

func TestOverloadUntyped2(t *testing.T) {
	gopMixedClTest(t, "main", `
package main

type SpriteImpl struct {
}

func (p *SpriteImpl) Rand__0(from int, to int) {
}
func (p *SpriteImpl) Rand__1(from float64, to float64) {
}
`, `
p := &SpriteImpl{}
p.rand(1.0,2.0)
p.rand(float64(1),float64(2))
`, `package main

func main() {
	p := &SpriteImpl{}
	p.Rand__0(1.0, 2.0)
	p.Rand__1(float64(1), float64(2))
}
`)
}

func TestSliceType(t *testing.T) {
	gopClTest(t, `
a := [1, "a"]
a[0] = [1, 2, 3]
echo a
`, `package main

import "fmt"

func main() {
	a := []interface{}{1, "a"}
	a[0] = []int{1, 2, 3}
	fmt.Println(a)
}
`)
}

func TestMapLitType(t *testing.T) {
	gopClTest(t, `
var a any = {
    "Monday": 1,
    "Tuesday": 2,
}
echo a
`, `package main

import "fmt"

var a interface{} = map[string]int{"Monday": 1, "Tuesday": 2}

func main() {
	fmt.Println(a)
}
`)
}
