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
	"os"
	"sync"
	"testing"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cl/cltest"
)

const (
	gopRootDir = ".."
)

var (
	gblConfLine  *cl.Config
	gotypesalias bool
	go1point     int
)

func init() {
	cltest.XGo.Root = gopRootDir
	conf := cltest.Conf
	gblConfLine = &cl.Config{
		Fset:          conf.Fset,
		Importer:      conf.Importer,
		Recorder:      cltest.Conf.Recorder,
		LookupClass:   cltest.LookupClass,
		NoFileLine:    false,
		NoAutoGenMain: true,
	}
	gotypesalias = cltest.EnableTypesalias()
	go1point = cltest.Go1Point()
}

func gopClNamedTest(t *testing.T, name string, gopcode, expected string) {
	cltest.Named(t, name, gopcode, expected)
}

func gopClTest(t *testing.T, gopcode, expected string) {
	cltest.DoExt(t, cltest.Conf, "main", gopcode, expected)
}

func gopClTestFile(t *testing.T, gopcode, expected string, fname string) {
	cltest.DoWithFname(t, gopcode, expected, fname)
}

func gopClTestEx(t *testing.T, conf *cl.Config, pkgname, gopcode, expected string) {
	cltest.DoExt(t, conf, pkgname, gopcode, expected)
}

func gopMixedClTest(t *testing.T, pkgname, gocode, gopcode, expected string, outline ...bool) {
	cltest.Mixed(t, pkgname, gocode, gopcode, expected, outline...)
}

func TestTypeDoc(t *testing.T) {
	gopClTest(t, `
type (
	// doc
	A int
)
`, `package main
// doc
type A int
`)
}

func TestUnsafe(t *testing.T) {
	gopClTest(t, `
import "unsafe"

println unsafe.Sizeof(0)
`, `package main

import (
	"fmt"
	"unsafe"
)

func main() {
	fmt.Println(unsafe.Sizeof(0))
}
`)
}

func Test_CastSlice_Issue1240(t *testing.T) {
	gopClTest(t, `
type fvec []float64
type foo float64
a := []float64([1, 2])
b := fvec([1, 2])
c := foo([1, 2])
d := fvec([])
println a, b, c, d
`, `package main

import "fmt"

type fvec []float64
type foo float64

func main() {
	a := []float64{1, 2}
	b := fvec{1, 2}
	c := foo([]int{1, 2})
	d := fvec{}
	fmt.Println(a, b, c, d)
}
`)
}

func TestUnderscoreRedeclared_Issue1197(t *testing.T) {
	gopClTest(t, `
func() (_ [2]int) { type _ int; return }()
`, `package main

func main() {
	func() (_ [2]int) {
		return
	}()
}
`)
}

func TestInterfaceBugNilUnderlying_Issue1198(t *testing.T) {
	gopClTest(t, `
import "runtime"

type Outer interface{ Inner }

type impl struct{}

func New() Outer { return &impl{} }

type Inner interface {
	DoStuff() error
}

func (a *impl) DoStuff() error {
	return nil
}

func main() {
	var outer Outer = New()
}
`, `package main

type Outer interface {
	Inner
}
type impl struct {
}
type Inner interface {
	DoStuff() error
}

func (a *impl) DoStuff() error {
	return nil
}
func New() Outer {
	return &impl{}
}
func main() {
	var outer Outer = New()
}
`)
}

func TestInterfaceBugNilUnderlying_Issue1196(t *testing.T) {
	gopClTest(t, `
func main() {
	i := I(A{})

	b := make(chan I, 1)
	b <- B{}

	var ok bool
	i, ok = <-b
}

type I interface{ M() int }

type T int

func (T) M() int { return 0 }

type A struct{ T }
type B struct{ T }
`, `package main

type I interface {
	M() int
}
type T int
type A struct {
	T
}
type B struct {
	T
}

func main() {
	i := I(A{})
	b := make(chan I, 1)
	b <- B{}
	var ok bool
	i, ok = <-b
}
func (T) M() int {
	return 0
}
`)
}

func TestMyIntInc_Issue1195(t *testing.T) {
	gopClTest(t, `
type MyInt int
var c MyInt
c++
`, `package main

type MyInt int

var c MyInt

func main() {
	c++
}
`)
}

func TestAutoPropMixedName_Issue1194(t *testing.T) {
	gopClTest(t, `
type Point struct {
	Min, Max int
}

type Obj struct {
	bbox Point
}

func (o *Obj) Bbox() Point {
	return o.bbox
}

func (o *Obj) Points() [2]int{
	return [2]int{o.bbox.Min, o.bbox.Max}
}
`, `package main

type Point struct {
	Min int
	Max int
}
type Obj struct {
	bbox Point
}

func (o *Obj) Bbox() Point {
	return o.bbox
}
func (o *Obj) Points() [2]int {
	return [2]int{o.bbox.Min, o.bbox.Max}
}
`)
}

func TestShiftUntypedInt_Issue1193(t *testing.T) {
	gopClTest(t, `
func GetValue(shift uint) uint {
	return 1 << shift
}`, `package main

func GetValue(shift uint) uint {
	return 1 << shift
}
`)
}

func TestInitFunc(t *testing.T) {
	gopClTest(t, `

func init() {}
func init() {}
`, `package main

func init() {
}
func init() {
}
`)
}

func TestSlogan(t *testing.T) {
	gopClTest(t, `
fields := ["engineering", "STEM education", "data science"]
println "The XGo Language for", fields.join(", ")
`, `package main

import (
	"fmt"
	"strings"
)

func main() {
	fields := []string{"engineering", "STEM education", "data science"}
	fmt.Println("The XGo Language for", strings.Join(fields, ", "))
}
`)
}

func TestAssignPrintln(t *testing.T) {
	gopClTest(t, `
p := println
p "Hello world"
`, `package main

import "fmt"

func main() {
	p := fmt.Println
	p("Hello world")
}
`)
}

func TestRedefineBuiltin(t *testing.T) {
	gopClTest(t, `
func main() {
	const a = append + len
}

const (
	append = iota
	len
)
`, `package main

const (
	append = iota
	len
)

func main() {
	const a = append + len
}
`)
}

func TestTypeConvIssue804(t *testing.T) {
	gopClTest(t, `
c := make(chan int)
d := (chan<- int)(c)
e := (<-chan int)(c)
f := (*int)(nil)
a := c == d
b := c == e
`, `package main

func main() {
	c := make(chan int)
	d := (chan<- int)(c)
	e := (<-chan int)(c)
	f := (*int)(nil)
	a := c == d
	b := c == e
}
`)
}

func TestUntypedFloatIssue798(t *testing.T) {
	gopClTest(t, `
func isPow10(x uint64) bool {
	switch x {
	case 1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
		1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19:
		return true
	}
	return false
}
`, `package main

func isPow10(x uint64) bool {
	switch x {
	case 1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19:
		return true
	}
	return false
}
`)
}

func TestInterfaceIssue795(t *testing.T) {
	gopClTest(t, `
type I interface {
	a(s string) I
	b(s string) string
}

type T1 int

func (t T1) a(s string) I {
	return t
}

func (T1) b(s string) string {
	return s
}
`, `package main

type I interface {
	a(s string) I
	b(s string) string
}
type T1 int

func (t T1) a(s string) I {
	return t
}
func (T1) b(s string) string {
	return s
}
`)
}

func TestChanRecvIssue789(t *testing.T) {
	gopClTest(t, `
func foo(ch chan int) (int, bool) {
	x, ok := (<-ch)
	return x, ok
}
`, `package main

func foo(ch chan int) (int, bool) {
	x, ok := <-ch
	return x, ok
}
`)
}

func TestNamedChanCloseIssue790(t *testing.T) {
	gopClTest(t, `
type XChan chan int

func foo(ch XChan) {
	close(ch)
}
`, `package main

type XChan chan int

func foo(ch XChan) {
	close(ch)
}
`)
}

func TestUntypedFloatIssue793(t *testing.T) {
	gopClTest(t, `
var a [1e1]int
`, `package main

var a [10]int
`)
}

func TestUntypedFloatIssue788(t *testing.T) {
	gopClTest(t, `
func foo(v int) bool {
    return v > 1.1e5
}
`, `package main

func foo(v int) bool {
	return v > 1.1e5
}
`)
}

func TestSwitchCompositeLitIssue801(t *testing.T) {
	gopClTest(t, `
type T struct {
	X int
}

switch (T{}) {
case T{1}:
	panic("bad")
}
`, `package main

type T struct {
	X int
}

func main() {
	switch (T{}) {
	case T{1}:
		panic("bad")
	}
}
`)
}

func TestConstIssue800(t *testing.T) {
	gopClTest(t, `
const (
	h0_0, h0_1 = 1.0 / (iota + 1), 1.0 / (iota + 2)
	h1_0, h1_1
)
`, `package main

const (
	h0_0, h0_1 = 1.0 / (iota + 1), 1.0 / (iota + 2)
	h1_0, h1_1
)
`)
}

func TestConstIssue805(t *testing.T) {
	gopClTest(t, `
const (
	n1 = +5
	d1 = +3

	q1 = +1
	r1 = +2
)

const (
	ret1 = n1/d1 != q1
	ret2 = n1%d1 != r1
	ret3 = n1/d1 != q1 || n1%d1 != r1
)
`, `package main

const (
	n1 = +5
	d1 = +3
	q1 = +1
	r1 = +2
)
const (
	ret1 = n1/d1 != q1
	ret2 = n1%d1 != r1
	ret3 = false
)
`)
}

func TestUntypedNilIssue806(t *testing.T) {
	gopClTest(t, `
switch f := func() {}; f {
case nil:
}
`, `package main

func main() {
	switch f := func() {
	}; f {
	case nil:
	}
}
`)
}

func TestSwitchIssue807(t *testing.T) {
	gopClTest(t, `
switch {
case interface{}(true):
}
`, `package main

func main() {
	switch {
	case interface{}(true):
	}
}
`)
}

func TestUntypedComplexIssue799(t *testing.T) {
	gopClTest(t, `
const ulp1 = imag(1i + 2i / 3 - 5i / 3)
const ulp2 = imag(1i + complex(0, 2) / 3 - 5i / 3)

func main() {
	const a = (ulp1 == ulp2)
}
`, `package main

const ulp1 = imag(1i + 2i/3 - 5i/3)
const ulp2 = imag(1i + complex(0, 2)/3 - 5i/3)

func main() {
	const a = ulp1 == ulp2
}
`)
}

func TestUnderscoreConstAndVar(t *testing.T) {
	gopClTest(t, `
const (
	c0 = 1 << iota
	_
	_
	_
	c4
)

func i() int {
	return 23
}

var (
	_ = i()
	_ = i()
)
`, `package main

const (
	c0 = 1 << iota
	_
	_
	_
	c4
)

func i() int {
	return 23
}

var _ = i()
var _ = i()
`)
}

func TestUnderscoreFuncAndMethod(t *testing.T) {
	gopClTest(t, `
func _() {
}

type T struct {
	_, _, _ int
}

func (T) _() {
}

func (T) _() {
}
`, `package main

type T struct {
	_ int
	_ int
	_ int
}

func (T) _() {
}
func (T) _() {
}
func _() {
}
`)
}

func TestErrWrapIssue772(t *testing.T) {
	gopClTest(t, `
package main

func t() (int,int,error){
	return 0, 0, nil
}

func main() {
	a, b := t()!
	println(a, b)
}`, `package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
)

func t() (int, int, error) {
	return 0, 0, nil
}
func main() {
	a, b := func() (_xgo_ret int, _xgo_ret2 int) {
		var _xgo_err error
		_xgo_ret, _xgo_ret2, _xgo_err = t()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "t()", "/foo/bar.xgo", 9, "main.main")
			panic(_xgo_err)
		}
		return
	}()
	fmt.Println(a, b)
}
`)
}

func TestErrWrapIssue778(t *testing.T) {
	gopClTest(t, `
package main

func t() error {
	return nil
}

func main() {
	t()!
}`, `package main

import "github.com/qiniu/x/errors"

func t() error {
	return nil
}
func main() {
	func() {
		var _xgo_err error
		_xgo_err = t()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "t()", "/foo/bar.xgo", 9, "main.main")
			panic(_xgo_err)
		}
		return
	}()
}
`)
}

func TestIssue774(t *testing.T) {
	gopClNamedTest(t, "InterfaceTypeAssert", `
package main

import "fmt"

func main() {
	var a AA = &A{str: "hello"}
	fmt.Println(a.(*A))
}

type AA interface {
	String() string
}

type A struct {
	str string
}

func (a *A) String() string {
	return a.str
}
`, `package main

import "fmt"

type AA interface {
	String() string
}
type A struct {
	str string
}

func main() {
	var a AA = &A{str: "hello"}
	fmt.Println(a.(*A))
}
func (a *A) String() string {
	return a.str
}
`)
	gopClNamedTest(t, "getInterface", `
package main

import "fmt"

func main() {
	a := get()
	fmt.Println(a.(*A))
}

type AA interface {
	String() string
}

func get() AA {
	var a AA
	return a
}

type A struct {
	str string
}

func (a *A) String() string {
	return a.str
}
`, `package main

import "fmt"

type AA interface {
	String() string
}
type A struct {
	str string
}

func main() {
	a := get()
	fmt.Println(a.(*A))
}
func get() AA {
	var a AA
	return a
}
func (a *A) String() string {
	return a.str
}
`)
}

func TestBlockStmt(t *testing.T) {
	gopClTest(t, `
package main

func main() {
	{
		type T int
		t := T(100)
		println(t)
	}
	{
		type T string
		t := "hello"
		println(t)
	}
}
`, `package main

import "fmt"

func main() {
	{
		type T int
		t := T(100)
		fmt.Println(t)
	}
	{
		type T string
		t := "hello"
		fmt.Println(t)
	}
}
`)
}

func TestConstTypeConvIssue792(t *testing.T) {
	gopClTest(t, `
const dots = ". . . " + ". . . . . "
const n = uint(len(dots))
`, `package main

const dots = ". . . " + ". . . . . "
const n = uint(len(dots))
`)
}

func TestVarInitTwoValueIssue791(t *testing.T) {
	gopClTest(t, `
var (
	m      = map[string]string{"a": "A"}
	a, ok  = m["a"]
)
`, `package main

var m = map[string]string{"a": "A"}
var a, ok = m["a"]
`)
}

func TestVarAfterMain(t *testing.T) {
	gopClTest(t, `
package main

func main() {
	println(i)
}

var i int
`, `package main

import "fmt"

func main() {
	fmt.Println(i)
}

var i int
`)
	gopClTest(t, `
package main

func f(v float64) float64 {
	return v
}
func main() {
	sink = f(100)
}

var sink float64
`, `package main

func f(v float64) float64 {
	return v
}
func main() {
	sink = f(100)
}

var sink float64
`)
}

func TestVarAfterMain2(t *testing.T) {
	gopClTest(t, `
package main

func main() {
	println(i)
}

var i = 100
`, `package main

import "fmt"

func main() {
	fmt.Println(i)
}

var i = 100
`)
}

func TestVarInMain(t *testing.T) {
	gopClTest(t, `
package main

func main() {
	v := []uint64{2, 3, 5}
	var n = len(v)
	println(n)
}`, `package main

import "fmt"

func main() {
	v := []uint64{2, 3, 5}
	var n = len(v)
	fmt.Println(n)
}
`)
}

func TestSelect(t *testing.T) {
	gopClTest(t, `

func consume(xchg chan int) {
	select {
	case c := <-xchg:
		println(c)
	case xchg <- 1:
		println("send ok")
	default:
		println(0)
	}
}
`, `package main

import "fmt"

func consume(xchg chan int) {
	select {
	case c := <-xchg:
		fmt.Println(c)
	case xchg <- 1:
		fmt.Println("send ok")
	default:
		fmt.Println(0)
	}
}
`)
}

func TestTypeSwitch(t *testing.T) {
	gopClTest(t, `

func bar(p *interface{}) {
}

func foo(v interface{}) {
	switch t := v.(type) {
	case int, string:
		bar(&v)
	case bool:
		var x bool = t
	default:
		bar(nil)
	}
}
`, `package main

func bar(p *interface{}) {
}
func foo(v interface{}) {
	switch t := v.(type) {
	case int, string:
		bar(&v)
	case bool:
		var x bool = t
	default:
		bar(nil)
	}
}
`)
}

func TestTypeSwitch2(t *testing.T) {
	gopClTest(t, `

func bar(p *interface{}) {
}

func foo(v interface{}) {
	switch bar(nil); v.(type) {
	case int, string:
		bar(&v)
	}
}
`, `package main

func bar(p *interface{}) {
}
func foo(v interface{}) {
	switch bar(nil); v.(type) {
	case int, string:
		bar(&v)
	}
}
`)
}

func TestTypeAssert(t *testing.T) {
	gopClTest(t, `

func foo(v interface{}) {
	x := v.(int)
	y, ok := v.(string)
}
`, `package main

func foo(v interface{}) {
	x := v.(int)
	y, ok := v.(string)
}
`)
}

func TestInterface(t *testing.T) {
	gopClTest(t, `

type Shape interface {
	Area() float64
}

func foo(shape Shape) {
	shape.Area()
}
`, `package main

type Shape interface {
	Area() float64
}

func foo(shape Shape) {
	shape.Area()
}
`)
}

func TestInterfaceEmbedded(t *testing.T) {
	gopClTest(t, `
type Shape interface {
	Area() float64
}

type Bar interface {
	Shape
}
`, `package main

type Shape interface {
	Area() float64
}
type Bar interface {
	Shape
}
`)
}

func TestInterfaceExample(t *testing.T) {
	gopClTest(t, `
type Shape interface {
	Area() float64
}

type Rect struct {
	x, y, w, h float64
}

func (p *Rect) Area() float64 {
	return p.w * p.h
}

type Circle struct {
	x, y, r float64
}

func (p *Circle) Area() float64 {
	return 3.14 * p.r * p.r
}

func Area(shapes ...Shape) float64 {
	s := 0.0
	for shape <- shapes {
		s += shape.Area()
	}
	return s
}

func main() {
	rect := &Rect{0, 0, 2, 5}
	circle := &Circle{0, 0, 3}
	println(Area(circle, rect))
}
`, `package main

import "fmt"

type Shape interface {
	Area() float64
}
type Rect struct {
	x float64
	y float64
	w float64
	h float64
}
type Circle struct {
	x float64
	y float64
	r float64
}

func (p *Rect) Area() float64 {
	return p.w * p.h
}
func (p *Circle) Area() float64 {
	return 3.14 * p.r * p.r
}
func Area(shapes ...Shape) float64 {
	s := 0.0
	for _, shape := range shapes {
		s += shape.Area()
	}
	return s
}
func main() {
	rect := &Rect{0, 0, 2, 5}
	circle := &Circle{0, 0, 3}
	fmt.Println(Area(circle, rect))
}
`)
}

func TestEmbeddField(t *testing.T) {
	gopClTest(t, `import "math/big"

type BigInt struct {
	*big.Int
}`, `package main

import "math/big"

type BigInt struct {
	*big.Int
}
`)
}

func TestAutoProperty(t *testing.T) {
	gopClTest(t, `import "github.com/goplus/xgo/ast/goptest"

func foo(script string) {
	doc := goptest.New(script)!

	echo doc.any.funcDecl.name
	echo doc.any.importSpec.name
}
`, `package main

import (
	"fmt"
	"github.com/goplus/xgo/ast/gopq"
	"github.com/goplus/xgo/ast/goptest"
	"github.com/qiniu/x/errors"
)

func foo(script string) {
	doc := func() (_xgo_ret gopq.NodeSet) {
		var _xgo_err error
		_xgo_ret, _xgo_err = goptest.New(script)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "goptest.New(script)", "/foo/bar.xgo", 4, "main.foo")
			panic(_xgo_err)
		}
		return
	}()
	fmt.Println(doc.Any().FuncDecl__0().Name())
	fmt.Println(doc.Any().ImportSpec().Name())
}
`)
}

func TestSimplifyAutoProperty(t *testing.T) {
	gopClTest(t, `import "gop/ast/goptest"

func foo(script string) {
	doc := goptest.New(script)!

	println(doc.any.funcDecl.name)
	println(doc.any.importSpec.name)
}
`, `package main

import (
	"fmt"
	"github.com/goplus/xgo/ast/gopq"
	"github.com/goplus/xgo/ast/goptest"
	"github.com/qiniu/x/errors"
)

func foo(script string) {
	doc := func() (_xgo_ret gopq.NodeSet) {
		var _xgo_err error
		_xgo_ret, _xgo_err = goptest.New(script)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "goptest.New(script)", "/foo/bar.xgo", 4, "main.foo")
			panic(_xgo_err)
		}
		return
	}()
	fmt.Println(doc.Any().FuncDecl__0().Name())
	fmt.Println(doc.Any().ImportSpec().Name())
}
`)
}

func TestErrWrapBasic(t *testing.T) {
	gopClTest(t, `
import "strconv"

func add(x, y string) (int, error) {
	return strconv.Atoi(x)? + strconv.Atoi(y)?, nil
}
`, `package main

import (
	"github.com/qiniu/x/errors"
	"strconv"
)

func add(x string, y string) (int, error) {
	var _autoGo_1 int
	{
		var _xgo_err error
		_autoGo_1, _xgo_err = strconv.Atoi(x)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "strconv.Atoi(x)", "/foo/bar.xgo", 5, "main.add")
			return 0, _xgo_err
		}
		goto _autoGo_2
	_autoGo_2:
	}
	var _autoGo_3 int
	{
		var _xgo_err error
		_autoGo_3, _xgo_err = strconv.Atoi(y)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "strconv.Atoi(y)", "/foo/bar.xgo", 5, "main.add")
			return 0, _xgo_err
		}
		goto _autoGo_4
	_autoGo_4:
	}
	return _autoGo_1 + _autoGo_3, nil
}
`)
}

func TestErrWrapDefVal(t *testing.T) {
	gopClTest(t, `
import "strconv"

func addSafe(x, y string) int {
	return strconv.Atoi(x)?:0 + strconv.Atoi(y)?:0
}
`, `package main

import "strconv"

func addSafe(x string, y string) int {
	return func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = strconv.Atoi(x)
		if _xgo_err != nil {
			return 0
		}
		return
	}() + func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = strconv.Atoi(y)
		if _xgo_err != nil {
			return 0
		}
		return
	}()
}
`)
}

func TestErrWrapPanic(t *testing.T) {
	gopClTest(t, `
var ret int = println("Hi")!
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
)

var ret int = func() (_xgo_ret int) {
	var _xgo_err error
	_xgo_ret, _xgo_err = fmt.Println("Hi")
	if _xgo_err != nil {
		_xgo_err = errors.NewFrame(_xgo_err, "println(\"Hi\")", "/foo/bar.xgo", 2, "main.main")
		panic(_xgo_err)
	}
	return
}()
`)
}

func TestErrWrapCommand(t *testing.T) {
	gopClTest(t, `
func mkdir(name string) error {
	return nil
}

mkdir! "foo"
`, `package main

import "github.com/qiniu/x/errors"

func mkdir(name string) error {
	return nil
}
func main() {
	func() {
		var _xgo_err error
		_xgo_err = mkdir("foo")
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "mkdir \"foo\"", "/foo/bar.xgo", 6, "main.main")
			panic(_xgo_err)
		}
		return
	}()
}
`)
}

func TestErrWrapCall(t *testing.T) {
	gopClTest(t, `
func foo() (func(), error) {
	return nil, nil
}

foo()!()
`, `package main

import "github.com/qiniu/x/errors"

func foo() (func(), error) {
	return nil, nil
}
func main() {
	func() (_xgo_ret func()) {
		var _xgo_err error
		_xgo_ret, _xgo_err = foo()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "foo()", "/foo/bar.xgo", 6, "main.main")
			panic(_xgo_err)
		}
		return
	}()()
}
`)
}

func TestMakeAndNew(t *testing.T) {
	gopClTest(t, `
var a *int = new(int)
var b map[string]int = make(map[string]int)
var c []byte = make([]byte, 0, 2)
`, `package main

var a *int = new(int)
var b map[string]int = make(map[string]int)
var c []byte = make([]byte, 0, 2)
`)
}

func TestVarDecl(t *testing.T) {
	gopClTest(t, `
var a int
var x, y = 1, "Hi"
`, `package main

var a int
var x, y = 1, "Hi"
`)
}

func TestUint128Add(t *testing.T) {
	gopClTest(t, `
var x, y uint128
var z uint128 = x + y
`, `package main

import "github.com/qiniu/x/xgo/ng"

var x, y ng.Uint128
var z ng.Uint128 = (ng.Uint128).XGo_Add__1(x, y)
`)
}

func TestInt128Add(t *testing.T) {
	gopClTest(t, `
var x, y int128
var z int128 = x + y
`, `package main

import "github.com/qiniu/x/xgo/ng"

var x, y ng.Int128
var z ng.Int128 = (ng.Int128).XGo_Add__1(x, y)
`)
}

func TestBigIntAdd(t *testing.T) {
	gopClTest(t, `
var x, y bigint
var z bigint = x + y
`, `package main

import "github.com/qiniu/x/xgo/ng"

var x, y ng.Bigint
var z ng.Bigint = (ng.Bigint).XGo_Add(x, y)
`)
}

func TestBigIntLit(t *testing.T) {
	gopClTest(t, `
var x = 1r
`, `package main

import (
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

var x = ng.Bigint_Init__1(big.NewInt(1))
`)
}

func TestUint128Lit(t *testing.T) {
	gopClTest(t, `
var x uint128 = 1
`, `package main

import "github.com/qiniu/x/xgo/ng"

var x ng.Uint128 = ng.Uint128_Init__0(1)
`)
}

func TestInt128Lit(t *testing.T) {
	gopClTest(t, `
var x int128 = 1
`, `package main

import "github.com/qiniu/x/xgo/ng"

var x ng.Int128 = ng.Int128_Init__0(1)
`)
}

func TestBigRatLit(t *testing.T) {
	gopClTest(t, `
var x = 1/2r
`, `package main

import (
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

var x = ng.Bigrat_Init__2(big.NewRat(1, 2))
`)
}

func TestBigRatLitAdd(t *testing.T) {
	gopClTest(t, `
var x = 3 + 1/2r
`, `package main

import (
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

var x = ng.Bigrat_Init__2(big.NewRat(7, 2))
`)
}

func TestBigRatAdd(t *testing.T) {
	gogen.SetDebug(gogen.DbgFlagAll)
	gopClTest(t, `
var x = 3 + 1/2r
var y = x + 100
var z = 100 + y
`, `package main

import (
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

var x = ng.Bigrat_Init__2(big.NewRat(7, 2))
var y = (ng.Bigrat).XGo_Add(x, ng.Bigrat_Init__0(100))
var z = (ng.Bigrat).XGo_Add(ng.Bigrat_Init__0(100), y)
`)
}

func TestTypeConv(t *testing.T) {
	gopClTest(t, `
var a = (*struct{})(nil)
var b = interface{}(nil)
var c = (func())(nil)
var x uint32 = uint32(0)
var y *uint32 = (*uint32)(nil)
`, `package main

var a = (*struct {
})(nil)
var b = interface{}(nil)
var c = (func())(nil)
var x uint32 = uint32(0)
var y *uint32 = (*uint32)(nil)
`)
}

func TestStar(t *testing.T) {
	gopClTest(t, `
var x *uint32 = (*uint32)(nil)
var y uint32 = *x
`, `package main

var x *uint32 = (*uint32)(nil)
var y uint32 = *x
`)
}

func TestLHS(t *testing.T) {
	gopClTest(t, `
type T struct {
	a int
}

func foo() *T {
	return nil
}

foo().a = 123
`, `package main

type T struct {
	a int
}

func foo() *T {
	return nil
}
func main() {
	foo().a = 123
}
`)
}

func TestSend(t *testing.T) {
	gopClTest(t, `
var x chan bool
x <- true
`, `package main

var x chan bool

func main() {
	x <- true
}
`)
}

func TestIncDec(t *testing.T) {
	gopClTest(t, `
var x uint32
x++
`, `package main

var x uint32

func main() {
	x++
}
`)
}

func TestAssignOp(t *testing.T) {
	gopClTest(t, `
var x uint32
x += 3
`, `package main

var x uint32

func main() {
	x += 3
}
`)
}

func TestBigIntAssignOp(t *testing.T) {
	gopClTest(t, `
var x bigint
x += 3
`, `package main

import "github.com/qiniu/x/xgo/ng"

var x ng.Bigint

func main() {
	x.XGo_AddAssign(ng.Bigint_Init__0(3))
}
`)
}

func TestBigIntAssignOp2(t *testing.T) {
	gopClTest(t, `
x := 3r
x *= 2
`, `package main

import (
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

func main() {
	x := ng.Bigint_Init__1(big.NewInt(3))
	x.XGo_MulAssign(ng.Bigint_Init__0(2))
}
`)
}

func TestBigIntAssignOp3(t *testing.T) {
	gopClTest(t, `
x := 3r
x *= 2r
`, `package main

import (
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

func main() {
	x := ng.Bigint_Init__1(big.NewInt(3))
	x.XGo_MulAssign(ng.Bigint_Init__1(big.NewInt(2)))
}
`)
}

func TestCompositeLit(t *testing.T) {
	gopClTest(t, `
x := []float64{1, 3.4, 5}
y := map[string]int{"Hello": 1, "XGo": 5}
z := [...]int{1, 3, 5}
a := {"Hello": 1, "XGo": 5.1}
`, `package main

func main() {
	x := []float64{1, 3.4, 5}
	y := map[string]int{"Hello": 1, "XGo": 5}
	z := [...]int{1, 3, 5}
	a := map[string]float64{"Hello": 1, "XGo": 5.1}
}
`)
}

func TestCompositeLit2(t *testing.T) {
	gopClTest(t, `
type foo struct {
	A int
}

x := []*struct{a int}{
	{1}, {3}, {5},
}
y := map[foo]struct{a string}{
	{1}: {"Hi"},
}
z := [...]foo{
	{1}, {3}, {5},
}
`, `package main

type foo struct {
	A int
}

func main() {
	x := []*struct {
		a int
	}{&struct {
		a int
	}{1}, &struct {
		a int
	}{3}, &struct {
		a int
	}{5}}
	y := map[foo]struct {
		a string
	}{foo{1}: struct {
		a string
	}{"Hi"}}
	z := [...]foo{foo{1}, foo{3}, foo{5}}
}
`)
}

// deduce struct type as parameters of a function call
func TestCompositeLit3(t *testing.T) {
	gopClTest(t, `
type Config struct {
	A int
}

func foo(conf *Config) {
}

func bar(conf ...Config) {
}

foo({A: 1})
bar({A: 2})
foo({})
bar({})
`, `package main

type Config struct {
	A int
}

func foo(conf *Config) {
}
func bar(conf ...Config) {
}
func main() {
	foo(&Config{A: 1})
	bar(Config{A: 2})
	foo(&Config{})
	bar(Config{})
}
`)
}

// deduce struct type as results of a function call
func TestCompositeLit4(t *testing.T) {
	gopClTest(t, `
type Result struct {
	A int
}

func foo() *Result {
	return {A: 1}
}
`, `package main

type Result struct {
	A int
}

func foo() *Result {
	return &Result{A: 1}
}
`)
}

func TestCompositeLit5(t *testing.T) {
	gopClTest(t, `
type mymap map[float64]string
var x = {1:"hello", 2:"world"}
var y map[float64]string = {1:"hello", 2:"world"}
var z mymap = {1:"hello", 2:"world"}
`, `package main

type mymap map[float64]string

var x = map[int]string{1: "hello", 2: "world"}
var y map[float64]string = map[float64]string{1: "hello", 2: "world"}
var z mymap = mymap{1: "hello", 2: "world"}
`)
}

func TestSliceLit(t *testing.T) {
	gopClTest(t, `
x := [1, 3.4, 5]
y := [1]
z := []
`, `package main

func main() {
	x := []float64{1, 3.4, 5}
	y := []int{1}
	z := []interface{}{}
}
`)
	gopClTest(t, `
type vector []float64
var x = [1, 2, 3]
var y []float64 = [1, 2, 3]
var z vector = [1, 2, 3]
`, `package main

type vector []float64

var x = []int{1, 2, 3}
var y []float64 = []float64{1, 2, 3}
var z vector = vector{1, 2, 3}
`)
}

func TestChan(t *testing.T) {
	gopClTest(t, `
a := make(chan int, 10)
a <- 3
var b int = <-a
x, ok := <-a
`, `package main

func main() {
	a := make(chan int, 10)
	a <- 3
	var b int = <-a
	x, ok := <-a
}
`)
}

func TestKeyValModeLit(t *testing.T) {
	gopClTest(t, `
a := [...]float64{1, 3: 3.4, 5}
b := []float64{2: 1.2, 3, 6: 4.5}
`, `package main

func main() {
	a := [...]float64{1, 3: 3.4, 5}
	b := []float64{2: 1.2, 3, 6: 4.5}
}
`)
}

func TestStructLit(t *testing.T) {
	gopClTest(t, `
type foo struct {
	A int
	B string "tag1:123"
}

a := struct {
	A int
	B string "tag1:123"
}{1, "Hello"}

b := foo{1, "Hello"}
c := foo{B: "Hi"}
`, `package main

type foo struct {
	A int
	B string `+"`tag1:123`"+`
}

func main() {
	a := struct {
		A int
		B string `+"`tag1:123`"+`
	}{1, "Hello"}
	b := foo{1, "Hello"}
	c := foo{B: "Hi"}
}
`)
}

func TestStructType(t *testing.T) {
	var expect string
	if gotypesalias {
		expect = `package main

type bar = foo
type foo struct {
	p *bar
	A int
	B string ` + "`tag1:123`" + `
}

func main() {
	type a struct {
		p *a
	}
	type b = a
}
`
	} else {
		expect = `package main

type bar = foo
type foo struct {
	p *foo
	A int
	B string ` + "`tag1:123`" + `
}

func main() {
	type a struct {
		p *a
	}
	type b = a
}
`
	}
	gopClTest(t, `
type bar = foo

type foo struct {
	p *bar
	A int
	B string "tag1:123"
}

func main() {
	type a struct {
		p *a
	}
	type b = a
}
`, expect)
}

func TestDeferGo(t *testing.T) {
	gopClTest(t, `
go println("Hi")
defer println("XGo")
`, `package main

import "fmt"

func main() {
	go fmt.Println("Hi")
	defer fmt.Println("XGo")
}
`)
}

func TestFor(t *testing.T) {
	gopClTest(t, `
a := [1, 3.4, 5]
for i := 0; i < 3; i=i+1 {
	println(i)
}
for {
	println("loop")
}
`, `package main

import "fmt"

func main() {
	a := []float64{1, 3.4, 5}
	for i := 0; i < 3; i = i + 1 {
		fmt.Println(i)
	}
	for {
		fmt.Println("loop")
	}
}
`)
}

func TestRangeStmt(t *testing.T) {
	gopClTest(t, `
a := [1, 3.4, 5]
for _, x := range a {
	println(x)
}
for i, x := range a {
	println(i, x)
}

var i int
var x float64
for _, x = range a {
	println(i, x)
}
for i, x = range a {
	println(i, x)
}
for range a {
	println("Hi")
}
`, `package main

import "fmt"

func main() {
	a := []float64{1, 3.4, 5}
	for _, x := range a {
		fmt.Println(x)
	}
	for i, x := range a {
		fmt.Println(i, x)
	}
	var i int
	var x float64
	for _, x = range a {
		fmt.Println(i, x)
	}
	for i, x = range a {
		fmt.Println(i, x)
	}
	for range a {
		fmt.Println("Hi")
	}
}
`)
}

func TestRangeStmtUDT(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (p *foo) Gop_Enum(c func(key int, val string)) {
}

for k, v := range new(foo) {
	println(k, v)
}
`, `package main

import "fmt"

type foo struct {
}

func (p *foo) Gop_Enum(c func(key int, val string)) {
}
func main() {
	new(foo).Gop_Enum(func(k int, v string) {
		fmt.Println(k, v)
	})
}
`)
}

func TestForPhraseUDT(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (p *foo) Gop_Enum(c func(val string)) {
}

for v <- new(foo) {
	println(v)
}
`, `package main

import "fmt"

type foo struct {
}

func (p *foo) Gop_Enum(c func(val string)) {
}
func main() {
	new(foo).Gop_Enum(func(v string) {
		fmt.Println(v)
	})
}
`)
}

func TestForPhraseUDT2(t *testing.T) {
	gopClTest(t, `
type fooIter struct {
}

func (p fooIter) Next() (key string, val int, ok bool) {
	return
}

type foo struct {
}

func (p *foo) Gop_Enum() fooIter {
	return fooIter{}
}

for k, v <- new(foo) {
	println(k, v)
}
`, `package main

import "fmt"

type fooIter struct {
}
type foo struct {
}

func (p fooIter) Next() (key string, val int, ok bool) {
	return
}
func (p *foo) Gop_Enum() fooIter {
	return fooIter{}
}
func main() {
	for _xgo_it := new(foo).Gop_Enum(); ; {
		var _xgo_ok bool
		k, v, _xgo_ok := _xgo_it.Next()
		if !_xgo_ok {
			break
		}
		fmt.Println(k, v)
	}
}
`)
}

func TestForPhraseUDT3(t *testing.T) {
	gopClTest(t, `
type foo struct {
}

func (p *foo) Gop_Enum(c func(val string)) {
}

println([v for v <- new(foo)])
`, `package main

import "fmt"

type foo struct {
}

func (p *foo) Gop_Enum(c func(val string)) {
}
func main() {
	fmt.Println(func() (_xgo_ret []string) {
		new(foo).Gop_Enum(func(v string) {
			_xgo_ret = append(_xgo_ret, v)
		})
		return
	}())
}
`)
}

func TestForPhraseUDT4(t *testing.T) {
	gopClTest(t, `
type fooIter struct {
	data *foo
	idx  int
}

func (p *fooIter) Next() (key int, val string, ok bool) {
	if p.idx < len(p.data.key) {
		key, val, ok = p.data.key[p.idx], p.data.val[p.idx], true
		p.idx++
	}
	return
}

type foo struct {
	key []int
	val []string
}

func newFoo() *foo {
	return &foo{key: [3, 7], val: ["Hi", "XGo"]}
}

func (p *foo) Gop_Enum() *fooIter {
	return &fooIter{data: p}
}

for k, v <- newFoo() {
	println(k, v)
}
`, `package main

import "fmt"

type fooIter struct {
	data *foo
	idx  int
}
type foo struct {
	key []int
	val []string
}

func (p *fooIter) Next() (key int, val string, ok bool) {
	if p.idx < len(p.data.key) {
		key, val, ok = p.data.key[p.idx], p.data.val[p.idx], true
		p.idx++
	}
	return
}
func (p *foo) Gop_Enum() *fooIter {
	return &fooIter{data: p}
}
func newFoo() *foo {
	return &foo{key: []int{3, 7}, val: []string{"Hi", "XGo"}}
}
func main() {
	for _xgo_it := newFoo().Gop_Enum(); ; {
		var _xgo_ok bool
		k, v, _xgo_ok := _xgo_it.Next()
		if !_xgo_ok {
			break
		}
		fmt.Println(k, v)
	}
}
`)
}

func TestForPhrase(t *testing.T) {
	gopClTest(t, `
sum := 0
for x <- [1, 3, 5, 7, 11, 13, 17], x > 3 {
	sum = sum + x
}
for i, x <- [1, 3, 5, 7, 11, 13, 17] {
	sum = sum + i*x
}
println("sum(5,7,11,13,17):", sum)
`, `package main

import "fmt"

func main() {
	sum := 0
	for _, x := range []int{1, 3, 5, 7, 11, 13, 17} {
		if x > 3 {
			sum = sum + x
		}
	}
	for i, x := range []int{1, 3, 5, 7, 11, 13, 17} {
		sum = sum + i*x
	}
	fmt.Println("sum(5,7,11,13,17):", sum)
}
`)
}

func TestExistsComprehension(t *testing.T) {
	gopClTest(t, `
hasFive := {for x <- ["1", "3", "5", "7", "11"], x == "5"}
`, `package main

func main() {
	hasFive := func() (_xgo_ok bool) {
		for _, x := range []string{"1", "3", "5", "7", "11"} {
			if x == "5" {
				return true
			}
		}
		return
	}()
}
`)
}

func TestSliceGet(t *testing.T) {
	gopClTest(t, `
a := [1, 3, 5, 7, 9]
b := a[:3]
c := a[1:]
d := a[1:2:3]
e := "Hello, XGo"[7:]
`, `package main

func main() {
	a := []int{1, 3, 5, 7, 9}
	b := a[:3]
	c := a[1:]
	d := a[1:2:3]
	e := "Hello, XGo"[7:]
}
`)
}

func TestIndexGetTwoValue(t *testing.T) {
	gopClTest(t, `
a := {"Hello": 1, "Hi": 3, "xsw": 5, "XGo": 7}
x, ok := a["Hi"]
y := a["XGo"]
`, `package main

func main() {
	a := map[string]int{"Hello": 1, "Hi": 3, "xsw": 5, "XGo": 7}
	x, ok := a["Hi"]
	y := a["XGo"]
}
`)
}

func TestIndexGet(t *testing.T) {
	gopClTest(t, `
a := [1, 3.4, 5]
b := a[1]
`, `package main

func main() {
	a := []float64{1, 3.4, 5}
	b := a[1]
}
`)
}

func TestIndexRef(t *testing.T) {
	gopClTest(t, `
a := [1, 3.4, 5]
a[1] = 2.1
`, `package main

func main() {
	a := []float64{1, 3.4, 5}
	a[1] = 2.1
}
`)
}

func TestIndexArrayPtrIssue784(t *testing.T) {
	gopClTest(t, `
type intArr [2]int

func foo(a *intArr) {
	a[1] = 10
}
`, `package main

type intArr [2]int

func foo(a *intArr) {
	a[1] = 10
}
`)
}

func TestMemberVal(t *testing.T) {
	gopClTest(t, `import "strings"

x := strings.NewReplacer("?", "!").Replace("hello, world???")
println("x:", x)
`, `package main

import (
	"fmt"
	"strings"
)

func main() {
	x := strings.NewReplacer("?", "!").Replace("hello, world???")
	fmt.Println("x:", x)
}
`)
}

func TestNamedPtrMemberIssue786(t *testing.T) {
	gopClTest(t, `
type foo struct {
	req int
}

type pfoo *foo

func bar(p pfoo) {
	println(p.req)
}
`, `package main

import "fmt"

type foo struct {
	req int
}
type pfoo *foo

func bar(p pfoo) {
	fmt.Println(p.req)
}
`)
}

func TestMember(t *testing.T) {
	gopClTest(t, `

import "flag"

a := &struct {
	A int
	B string
}{1, "Hello"}

x := a.A
a.B = "Hi"

flag.Usage = nil
`, `package main

import "flag"

func main() {
	a := &struct {
		A int
		B string
	}{1, "Hello"}
	x := a.A
	a.B = "Hi"
	flag.Usage = nil
}
`)
}

func TestElem(t *testing.T) {
	gopClTest(t, `

func foo(a *int, b int) {
	b = *a
	*a = b
}
`, `package main

func foo(a *int, b int) {
	b = *a
	*a = b
}
`)
}

func TestNamedPtrIssue797(t *testing.T) {
	gopClTest(t, `
type Bar *int

func foo(a Bar) {
	var b int = *a
}
`, `package main

type Bar *int

func foo(a Bar) {
	var b int = *a
}
`)
}

func TestMethod(t *testing.T) {
	gopClTest(t, `
type M int

func (m M) Foo() {
	println("foo", m)
}

func (M) Bar() {
	println("bar")
}
`, `package main

import "fmt"

type M int

func (m M) Foo() {
	fmt.Println("foo", m)
}
func (M) Bar() {
	fmt.Println("bar")
}
`)
}

func TestCmdlineNoEOL(t *testing.T) {
	gopClTest(t, `println "Hi"`, `package main

import "fmt"

func main() {
	fmt.Println("Hi")
}
`)
}

func TestImport(t *testing.T) {
	gopClTest(t, `import "fmt"

func main() {
	fmt.println "Hi"
}`, `package main

import "fmt"

func main() {
	fmt.Println("Hi")
}
`)
}

func TestDotImport(t *testing.T) {
	gopClTest(t, `import . "math"

var a = round(1.2)
`, `package main

import "math"

var a = math.Round(1.2)
`)
}

func TestLocalImport(t *testing.T) {
	gopClTest(t, `import "./internal/spx"

var a = spx.TestIntValue
`, `package main

import "github.com/goplus/xgo/cl/internal/spx"

var a = spx.TestIntValue
`)
}

func TestImportUnused(t *testing.T) {
	gopClTest(t, `import "fmt"

func main() {
}`, `package main

func main() {
}
`)
}

func TestImportForceUsed(t *testing.T) {
	gopClTest(t, `import _ "fmt"

func main() {
}`, `package main

import _ "fmt"

func main() {
}
`)
}

func TestAnonymousImport(t *testing.T) {
	gopClTest(t, `println("Hello")
printf("Hello XGo\n")
`, `package main

import "fmt"

func main() {
	fmt.Println("Hello")
	fmt.Printf("Hello XGo\n")
}
`)
}

func TestVarAndConst(t *testing.T) {
	gopClTest(t, `
const (
	i = 1
	x float64 = 1
)
var j int = i
`, `package main

const (
	i         = 1
	x float64 = 1
)

var j int = i
`)
}

func TestDeclStmt(t *testing.T) {
	gopClTest(t, `import "fmt"

func main() {
	const (
		i = 1
		x float64 = 1
	)
	var j int = i
	fmt.Println("Hi")
}`, `package main

import "fmt"

func main() {
	const (
		i         = 1
		x float64 = 1
	)
	var j int = i
	fmt.Println("Hi")
}
`)
}

func TestIf(t *testing.T) {
	gopClTest(t, `x := 0
if t := false; t {
	x = 3
} else if !t {
	x = 5
} else {
	x = 7
}
println("x:", x)
`, `package main

import "fmt"

func main() {
	x := 0
	if t := false; t {
		x = 3
	} else if !t {
		x = 5
	} else {
		x = 7
	}
	fmt.Println("x:", x)
}
`)
}

func TestSwitch(t *testing.T) {
	gopClTest(t, `x := 0
switch s := "Hello"; s {
default:
	x = 7
case "world", "hi":
	x = 5
case "xsw":
	x = 3
}
println("x:", x)

v := "Hello"
switch {
case v == "xsw":
	x = 3
case v == "hi", v == "world":
	x = 9
default:
	x = 11
}
println("x:", x)
`, `package main

import "fmt"

func main() {
	x := 0
	switch s := "Hello"; s {
	default:
		x = 7
	case "world", "hi":
		x = 5
	case "xsw":
		x = 3
	}
	fmt.Println("x:", x)
	v := "Hello"
	switch {
	case v == "xsw":
		x = 3
	case v == "hi", v == "world":
		x = 9
	default:
		x = 11
	}
	fmt.Println("x:", x)
}
`)
}

func TestSwitchFallthrough(t *testing.T) {
	gopClTest(t, `v := "Hello"
switch v {
case "Hello":
	println(v)
	fallthrough
case "hi":
	println(v)
	fallthrough
default:
	println(v)
}
`, `package main

import "fmt"

func main() {
	v := "Hello"
	switch v {
	case "Hello":
		fmt.Println(v)
		fallthrough
	case "hi":
		fmt.Println(v)
		fallthrough
	default:
		fmt.Println(v)
	}
}
`)
}

func TestBranchStmt(t *testing.T) {
	gopClTest(t, `
	a := [1, 3.4, 5]
label:
	for i := 0; i < 3; i=i+1 {
		println(i)
		break
		break label
		continue
		continue label
		goto label
	}
`, `package main

import "fmt"

func main() {
	a := []float64{1, 3.4, 5}
label:
	for i := 0; i < 3; i = i + 1 {
		fmt.Println(i)
		break
		break label
		continue
		continue label
		goto label
	}
}
`)
}

func TestReturn(t *testing.T) {
	gopClTest(t, `
func foo(format string, args ...interface{}) (int, error) {
	return printf(format, args...)
}

func main() {
}
`, `package main

import "fmt"

func foo(format string, args ...interface{}) (int, error) {
	return fmt.Printf(format, args...)
}
func main() {
}
`)
}

func TestReturnExpr(t *testing.T) {
	gopClTest(t, `
func foo(format string, args ...interface{}) (int, error) {
	return 0, nil
}

func main() {
}
`, `package main

func foo(format string, args ...interface{}) (int, error) {
	return 0, nil
}
func main() {
}
`)
}

func TestClosure(t *testing.T) {
	gopClTest(t, `import "fmt"

func(v string) {
	fmt.Println(v)
}("Hello")
`, `package main

import "fmt"

func main() {
	func(v string) {
		fmt.Println(v)
	}("Hello")
}
`)
}

func TestFunc(t *testing.T) {
	gopClTest(t, `func foo(format string, a [10]int, args ...interface{}) {
}

func main() {
}`, `package main

func foo(format string, a [10]int, args ...interface{}) {
}
func main() {
}
`)
}

func TestLambdaExpr(t *testing.T) {
	gopClTest(t, `
func Map(c []float64, t func(float64) float64) {
	// ...
}

func Map2(c []float64, t func(float64) (float64, float64)) {
	// ...
}

Map([1.2, 3.5, 6], x => x * x)
Map2([1.2, 3.5, 6], x => (x * x, x + x))
`, `package main

func Map(c []float64, t func(float64) float64) {
}
func Map2(c []float64, t func(float64) (float64, float64)) {
}
func main() {
	Map([]float64{1.2, 3.5, 6}, func(x float64) float64 {
		return x * x
	})
	Map2([]float64{1.2, 3.5, 6}, func(x float64) (float64, float64) {
		return x * x, x + x
	})
}
`)
	gopClTest(t, `type Foo struct {
	Plot func(x float64) (float64, float64)
}
foo := &Foo{
	Plot: x => (x * 2, x * x),
}`, `package main

type Foo struct {
	Plot func(x float64) (float64, float64)
}

func main() {
	foo := &Foo{Plot: func(x float64) (float64, float64) {
		return x * 2, x * x
	}}
}
`)
	gopClTest(t, `
type Fn func(x float64) (float64, float64)
type Foo struct {
	Plot Fn
}
foo := &Foo{
	Plot: x => (x * 2, x * x),
}`, `package main

type Fn func(x float64) (float64, float64)
type Foo struct {
	Plot Fn
}

func main() {
	foo := &Foo{Plot: func(x float64) (float64, float64) {
		return x * 2, x * x
	}}
}
`)
	gopClTest(t, `
type Fn func() (int, error)
func Do(fn Fn) {
}

Do => (100, nil)
`, `package main

type Fn func() (int, error)

func Do(fn Fn) {
}
func main() {
	Do(func() (int, error) {
		return 100, nil
	})
}
`)
	gopClTest(t, `
var fn func(int) (int,error) = x => (x*x, nil)
`, `package main

var fn func(int) (int, error) = func(x int) (int, error) {
	return x * x, nil
}
`)
	gopClTest(t, `
var fn func(int) (int,error)
fn = x => (x*x, nil)
`, `package main

var fn func(int) (int, error)

func main() {
	fn = func(x int) (int, error) {
		return x * x, nil
	}
}
`)
}

func TestLambdaExpr2(t *testing.T) {
	gopClTest(t, `
func Do(func()) {
	// ...
}

Do => {
	println "Hi"
}
`, `package main

import "fmt"

func Do(func()) {
}
func main() {
	Do(func() {
		fmt.Println("Hi")
	})
}
`)
	gopClTest(t, `
func Do(fn func() (int, error)) {
	// ...
}

Do => {
	return 100, nil
}
`, `package main

func Do(fn func() (int, error)) {
}
func main() {
	Do(func() (int, error) {
		return 100, nil
	})
}
`)
	gopClTest(t, `type Foo struct {
	Plot func(x float64) (float64, float64)
}
foo := &Foo{
	Plot: x => {
		return x * 2, x * x
	},
}`, `package main

type Foo struct {
	Plot func(x float64) (float64, float64)
}

func main() {
	foo := &Foo{Plot: func(x float64) (float64, float64) {
		return x * 2, x * x
	}}
}
`)
	gopClTest(t, `
type Fn func(x float64) (float64, float64)
type Foo struct {
	Plot Fn
}
foo := &Foo{
	Plot: x => {
		return x * 2, x * x
	},
}`, `package main

type Fn func(x float64) (float64, float64)
type Foo struct {
	Plot Fn
}

func main() {
	foo := &Foo{Plot: func(x float64) (float64, float64) {
		return x * 2, x * x
	}}
}
`)

	gopClTest(t, `
type Fn func() (int, error)
func Do(fn Fn) {
}

Do => {
	return 100, nil
}
`, `package main

type Fn func() (int, error)

func Do(fn Fn) {
}
func main() {
	Do(func() (int, error) {
		return 100, nil
	})
}
`)
	gopClTest(t, `
var fn func(int) (int,error) = x => {
	return x * x, nil
}
`, `package main

var fn func(int) (int, error) = func(x int) (int, error) {
	return x * x, nil
}
`)
	gopClTest(t, `
var fn func(int) (int,error)
fn = x => {
	return x * x, nil
}
`, `package main

var fn func(int) (int, error)

func main() {
	fn = func(x int) (int, error) {
		return x * x, nil
	}
}
`)
}

func TestLambdaExpr3(t *testing.T) {
	gopClTest(t, `
func intSeq() func() int {
	i := 0
	return => {
		i++
		return i
	}
}
`, `package main

func intSeq() func() int {
	i := 0
	return func() int {
		i++
		return i
	}
}
`)
	gopClTest(t, `
func intDouble() func(int) int {
	return i => i*2
}
`, `package main

func intDouble() func(int) int {
	return func(i int) int {
		return i * 2
	}
}
`)
}

func TestUnnamedMainFunc(t *testing.T) {
	gopClTest(t, `i := 1`, `package main

func main() {
	i := 1
}
`)
}

func TestFuncAsParam(t *testing.T) {
	gopClTest(t, `import "fmt"

func bar(foo func(string, ...interface{}) (int, error)) {
	foo("Hello, %v!\n", "XGo")
}

bar(fmt.Printf)
`, `package main

import "fmt"

func bar(foo func(string, ...interface{}) (int, error)) {
	foo("Hello, %v!\n", "XGo")
}
func main() {
	bar(fmt.Printf)
}
`)
}

func TestFuncAsParam2(t *testing.T) {
	gopClTest(t, `import (
	"fmt"
	"strings"
)

func foo(x string) string {
	return strings.NewReplacer("?", "!").Replace(x)
}

func printf(format string, args ...interface{}) (n int, err error) {
	n, err = fmt.Printf(format, args...)
	return
}

func bar(foo func(string, ...interface{}) (int, error)) {
	foo("Hello, %v!\n", "XGo")
}

bar(printf)
fmt.Println(foo("Hello, world???"))
fmt.Println(printf("Hello, %v\n", "XGo"))
`, `package main

import (
	"fmt"
	"strings"
)

func foo(x string) string {
	return strings.NewReplacer("?", "!").Replace(x)
}
func printf(format string, args ...interface{}) (n int, err error) {
	n, err = fmt.Printf(format, args...)
	return
}
func bar(foo func(string, ...interface{}) (int, error)) {
	foo("Hello, %v!\n", "XGo")
}
func main() {
	bar(printf)
	fmt.Println(foo("Hello, world???"))
	fmt.Println(printf("Hello, %v\n", "XGo"))
}
`)
}

func TestFuncCall(t *testing.T) {
	gopClTest(t, `import "fmt"

fmt.Println("Hello")`, `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`)
}

func TestFuncCallEllipsis(t *testing.T) {
	gopClTest(t, `import "fmt"

func foo(args ...interface{}) {
	fmt.Println(args...)
}

func main() {
}`, `package main

import "fmt"

func foo(args ...interface{}) {
	fmt.Println(args...)
}
func main() {
}
`)
}

func TestFuncCallCodeOrder(t *testing.T) {
	gopClTest(t, `import "fmt"

func main() {
	foo("Hello", 123)
}

func foo(args ...interface{}) {
	fmt.Println(args...)
}
`, `package main

import "fmt"

func main() {
	foo("Hello", 123)
}
func foo(args ...interface{}) {
	fmt.Println(args...)
}
`)
}

func TestInterfaceMethods(t *testing.T) {
	gopClTest(t, `package main

func foo(v ...interface { Bar() }) {
}

func main() {
}`, `package main

func foo(v ...interface {
	Bar()
}) {
}
func main() {
}
`)
}

func TestAssignUnderscore(t *testing.T) {
	gopClTest(t, `import log "fmt"

_, err := log.Println("Hello")
`, `package main

import "fmt"

func main() {
	_, err := fmt.Println("Hello")
}
`)
}

func TestOperator(t *testing.T) {
	gopClTest(t, `
a := "Hi"
b := a + "!"
c := 13
d := -c
`, `package main

func main() {
	a := "Hi"
	b := a + "!"
	c := 13
	d := -c
}
`)
}

var (
	autogen sync.Mutex
)

func removeAutogenFiles() {
	os.Remove("./internal/gop-in-go/foo/gop_autogen.go")
	os.Remove("./internal/gop-in-go/foo/gop_autogen_test.go")
	os.Remove("./internal/gop-in-go/foo/gop_autogen2_test.go")
}

func TestImportGopPkg(t *testing.T) {
	autogen.Lock()
	defer autogen.Unlock()

	removeAutogenFiles()
	gopClTest(t, `import "github.com/goplus/xgo/cl/internal/gop-in-go/foo"

rmap := foo.ReverseMap(map[string]int{"Hi": 1, "Hello": 2})
println(rmap)
`, `package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/gop-in-go/foo"
)

func main() {
	rmap := foo.ReverseMap(map[string]int{"Hi": 1, "Hello": 2})
	fmt.Println(rmap)
}
`)
}

func TestCallDep(t *testing.T) {
	for i := 0; i < 2; i++ {
		gopClTest(t, `
import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	ret := New()
	expected := Result{}
	if reflect.DeepEqual(ret, expected) {
		t.Fatal("Test failed:", ret, expected)
	}
}

type Repo struct {
	Title string
}

func newRepo() Repo {
	return {Title: "Hi"}
}

type Result struct {
	Repo Repo
}

func New() Result {
	repo := newRepo()
	return {Repo: repo}
}
`, `package main

import (
	"reflect"
	"testing"
)

type Repo struct {
	Title string
}
type Result struct {
	Repo Repo
}

func TestNew(t *testing.T) {
	ret := New()
	expected := Result{}
	if reflect.DeepEqual(ret, expected) {
		t.Fatal("Test failed:", ret, expected)
	}
}
func New() Result {
	repo := newRepo()
	return Result{Repo: repo}
}
func newRepo() Repo {
	return Repo{Title: "Hi"}
}
`)
	}
}

func TestGoFuncInstr(t *testing.T) {
	gopClTest(t, `package main

//go:noinline
//go:uintptrescapes
func test(s string, p, q uintptr, rest ...uintptr) int {
	return 0
}`, `package main
//go:noinline
//go:uintptrescapes
func test(s string, p uintptr, q uintptr, rest ...uintptr) int {
	return 0
}
`)
}

func TestGoTypeInstr(t *testing.T) {
	gopClTest(t, `package main

//go:notinheap
type S struct{ x int }
`, `package main
//go:notinheap
type S struct {
	x int
}
`)
}

func TestNoEntrypoint(t *testing.T) {
	gopClTest(t, `println("init")
`, `package main

import "fmt"

func main() {
	fmt.Println("init")
}
`)
	gopClTestEx(t, cltest.Conf, "bar", `package bar
println("init")
`, `package bar

import "fmt"

func init() {
	fmt.Println("init")
}
`)
}

func TestParentExpr(t *testing.T) {
	gopClTest(t, `var t1 *(int)
var t2 chan (int)
`, `package main

var t1 *int
var t2 chan int
`)
}

func TestCommandStyle(t *testing.T) {
	gopClTest(t, `
println []
println {}
`, `package main

import "fmt"

func main() {
	fmt.Println([]interface{}{})
	fmt.Println(map[string]interface{}{})
}
`)
}

func TestTypeLoader(t *testing.T) {
	gopClTest(t, `import "fmt"

func (p *Point) String() string {
	return fmt.Sprintf("%v-%v",p.X,p.Y)
}

type Point struct {
	X int
	Y int
}
`, `package main

import "fmt"

type Point struct {
	X int
	Y int
}

func (p *Point) String() string {
	return fmt.Sprintf("%v-%v", p.X, p.Y)
}
`)
}

func TestCallPrintln(t *testing.T) {
	gopClTest(t, `
print
print "hello"
print("hello")
println
println "hello"
println("hello")
`, `package main

import "fmt"

func main() {
	fmt.Print()
	fmt.Print("hello")
	fmt.Print("hello")
	fmt.Println()
	fmt.Println("hello")
	fmt.Println("hello")
}
`)
}

func TestAnyAlias(t *testing.T) {
	gopClTest(t, `
var a any = 100
println(a)
`, `package main

import "fmt"

var a interface{} = 100

func main() {
	fmt.Println(a)
}
`)
}

func TestMainEntry(t *testing.T) {
	conf := *cltest.Conf
	conf.NoAutoGenMain = false

	gopClTestEx(t, &conf, "main", `
`, `package main

func main() {
}
`)
	gopClTestEx(t, &conf, "main", `
func test() {
	println "hello"
}
`, `package main

import "fmt"

func test() {
	fmt.Println("hello")
}
func main() {
}
`)

	gopClTestEx(t, &conf, "main", `
func main() {
	println "hello"
}
`, `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)
}

func TestCommandNotExpr(t *testing.T) {
	gopClTest(t, `
println !true
`, `package main

import "fmt"

func main() {
	fmt.Println(false)
}
`)
	gopClTest(t, `
a := true
println !a
`, `package main

import "fmt"

func main() {
	a := true
	fmt.Println(!a)
}
`)
	gopClTest(t, `
println !func() bool { return true }()
`, `package main

import "fmt"

func main() {
	fmt.Println(!func() bool {
		return true
	}())
}
`)
}

func TestCommentLine(t *testing.T) {
	gopClTestEx(t, gblConfLine, "main", `
type Point struct {
	x int
	y int
}

func (pt *Point) Test() {
	println(pt.x, pt.y)
}

// testPoint is test point
func testPoint() {
	var pt Point
	pt.Test()
}

println "hello"
testPoint()
`, `package main

import "fmt"

type Point struct {
	x int
	y int
}
//line /foo/bar.xgo:7:1
func (pt *Point) Test() {
//line /foo/bar.xgo:8:1
	fmt.Println(pt.x, pt.y)
}
//line /foo/bar.xgo:11:1
// testPoint is test point
func testPoint() {
//line /foo/bar.xgo:13:1
	var pt Point
//line /foo/bar.xgo:14:1
	pt.Test()
}
//line /foo/bar.xgo:17
func main() {
//line /foo/bar.xgo:17:1
	fmt.Println("hello")
//line /foo/bar.xgo:18:1
	testPoint()
}
`)
}

func TestCommentLineRoot(t *testing.T) {
	conf := *cltest.Conf
	conf.NoFileLine = false
	conf.RelativeBase = "/foo/root"
	var src = `
type Point struct {
	x int
	y int
}

func (pt *Point) Test() {
	println(pt.x, pt.y)
}

// testPoint is test point
func testPoint() {
	var pt Point
	pt.Test()
}

println "hello"
testPoint()
`
	var expected = `package main

import "fmt"

type Point struct {
	x int
	y int
}
//line ../bar.xgo:7:1
func (pt *Point) Test() {
//line ../bar.xgo:8:1
	fmt.Println(pt.x, pt.y)
}
//line ../bar.xgo:11:1
// testPoint is test point
func testPoint() {
//line ../bar.xgo:13:1
	var pt Point
//line ../bar.xgo:14:1
	pt.Test()
}
//line ../bar.xgo:17
func main() {
//line ../bar.xgo:17:1
	fmt.Println("hello")
//line ../bar.xgo:18:1
	testPoint()
}
`
	gopClTestEx(t, &conf, "main", src, expected)
}

func TestRangeScope(t *testing.T) {
	gopClTest(t, `
ar := []int{100, 200}
for k, v := range ar {
	println(k, v, ar)
	var k, v, ar int
	println(ar, k, v)
}
`, `package main

import "fmt"

func main() {
	ar := []int{100, 200}
	for k, v := range ar {
		fmt.Println(k, v, ar)
		var k, v, ar int
		fmt.Println(ar, k, v)
	}
}
`)
}

func TestSelectScope(t *testing.T) {
	gopClTest(t, `
c1 := make(chan int)
c2 := make(chan int)
go func() {
	c1 <- 100
}()
select {
case i := <-c1:
	println i
case i := <-c2:
	println i
}
`, `package main

import "fmt"

func main() {
	c1 := make(chan int)
	c2 := make(chan int)
	go func() {
		c1 <- 100
	}()
	select {
	case i := <-c1:
		fmt.Println(i)
	case i := <-c2:
		fmt.Println(i)
	}
}
`)
}

func TestCommentVar(t *testing.T) {
	gopClTestEx(t, gblConfLine, "main", `
// doc a line2
var a int
println a

// doc b line6
var b int
println b

var c int
println c
`, `package main

import "fmt"
// doc a line2
var a int
//line /foo/bar.xgo:4
func main() {
//line /foo/bar.xgo:4:1
	fmt.Println(a)
//line /foo/bar.xgo:6:1
	// doc b line6
	var b int
//line /foo/bar.xgo:8:1
	fmt.Println(b)
//line /foo/bar.xgo:10:1
	var c int
//line /foo/bar.xgo:11:1
	fmt.Println(c)
}
`)

	gopClTestEx(t, gblConfLine, "main", `
func demo() {
	// doc a line3
	var a int
	println a
	
	// doc b line7
	var b int
	println b
	
	var c int
	println c
}
`, `package main

import "fmt"
//line /foo/bar.xgo:2:1
func demo() {
//line /foo/bar.xgo:3:1
	// doc a line3
	var a int
//line /foo/bar.xgo:5:1
	fmt.Println(a)
//line /foo/bar.xgo:7:1
	// doc b line7
	var b int
//line /foo/bar.xgo:9:1
	fmt.Println(b)
//line /foo/bar.xgo:11:1
	var c int
//line /foo/bar.xgo:12:1
	fmt.Println(c)
}
`)
}

func TestForPhraseScope(t *testing.T) {
	gopClTest(t, `sum := 0
for x <- [1, 3, 5, 7, 11, 13, 17] {
	sum = sum + x
	println x
	x := 200
	println x
}`, `package main

import "fmt"

func main() {
	sum := 0
	for _, x := range []int{1, 3, 5, 7, 11, 13, 17} {
		sum = sum + x
		fmt.Println(x)
		x := 200
		fmt.Println(x)
	}
}
`)
	gopClTest(t, `sum := 0
for x <- [1, 3, 5, 7, 11, 13, 17], x > 3 {
	sum = sum + x
	println x
	x := 200
	println x
}`, `package main

import "fmt"

func main() {
	sum := 0
	for _, x := range []int{1, 3, 5, 7, 11, 13, 17} {
		if x > 3 {
			sum = sum + x
			fmt.Println(x)
			x := 200
			fmt.Println(x)
		}
	}
}
`)
}

func TestAddress(t *testing.T) {
	gopClTest(t, `
type foo struct{ c int }

func (f foo) ptr() *foo { return &f }
func (f foo) clone() foo { return f }

type nested struct {
	f foo
	a [2]foo
	s []foo
}

func _() {
	getNested := func() nested { return nested{} }

	_ = getNested().f.c
	_ = getNested().a[0].c
	_ = getNested().s[0].c
	_ = getNested().f.ptr().c
	_ = getNested().f.clone().c
	_ = getNested().f.clone().ptr().c
}
`, `package main

type foo struct {
	c int
}
type nested struct {
	f foo
	a [2]foo
	s []foo
}

func (f foo) ptr() *foo {
	return &f
}
func (f foo) clone() foo {
	return f
}
func _() {
	getNested := func() nested {
		return nested{}
	}
	_ = getNested().f.c
	_ = getNested().a[0].c
	_ = getNested().s[0].c
	_ = getNested().f.ptr().c
	_ = getNested().f.clone().c
	_ = getNested().f.clone().ptr().c
}
`)
}

func TestSliceLitAssign(t *testing.T) {
	gopClTest(t, `
var n = 1
var a []any = [10, 3.14, 200]
n, a = 100, [10, 3.14, 200]
echo a, n
`, `package main

import "fmt"

var n = 1
var a []interface{} = []interface{}{10, 3.14, 200}

func main() {
	n, a = 100, []interface{}{10, 3.14, 200}
	fmt.Println(a, n)
}
`)
}

func TestSliceLitReturn(t *testing.T) {
	gopClTest(t, `
func anyslice() (int, []any) {
	return 100, [10, 3.14, 200]
}
n, a := anyslice()
echo n, a
`, `package main

import "fmt"

func anyslice() (int, []interface{}) {
	return 100, []interface{}{10, 3.14, 200}
}
func main() {
	n, a := anyslice()
	fmt.Println(n, a)
}
`)
}

func TestCompositeLitAssign(t *testing.T) {
	gopClTest(t, `
var a map[any]any = {10: "A", 3.14: "B", 200: "C"}
var b map[any]string = {10: "A", 3.14: "B", 200: "C"}
echo a
echo b
var n int
n, a = 1, {10: "A", 3.14: "B", 200: "C"}
echo a, n
n, b = 1, {10: "A", 3.14: "B", 200: "C"}
echo b, n
`, `package main

import "fmt"

var a map[interface{}]interface{} = map[interface{}]interface{}{10: "A", 3.14: "B", 200: "C"}
var b map[interface{}]string = map[interface{}]string{10: "A", 3.14: "B", 200: "C"}

func main() {
	fmt.Println(a)
	fmt.Println(b)
	var n int
	n, a = 1, map[interface{}]interface{}{10: "A", 3.14: "B", 200: "C"}
	fmt.Println(a, n)
	n, b = 1, map[interface{}]string{10: "A", 3.14: "B", 200: "C"}
	fmt.Println(b, n)
}
`)
}

func TestCompositeLitStruct(t *testing.T) {
	gopClTest(t, `
type T struct {
	s  []any
	m  map[any]any
	fn func(int) int
}

echo &T{[10, 3.14, 200], {10: "A", 3.14: "B", 200: "C"}, (x => x)}
echo &T{s: [10, 3.14, 200], m: {10: "A", 3.14: "B", 200: "C"}, fn: (x => x)}
`, `package main

import "fmt"

type T struct {
	s  []interface{}
	m  map[interface{}]interface{}
	fn func(int) int
}

func main() {
	fmt.Println(&T{[]interface{}{10, 3.14, 200}, map[interface{}]interface{}{10: "A", 3.14: "B", 200: "C"}, func(x int) int {
		return x
	}})
	fmt.Println(&T{s: []interface{}{10, 3.14, 200}, m: map[interface{}]interface{}{10: "A", 3.14: "B", 200: "C"}, fn: func(x int) int {
		return x
	}})
}
`)
}

func TestCompositeLitEx(t *testing.T) {
	gopClTest(t, `
var a [][]any = {[10, 3.14, 200], [100, 200]}
var m map[any][]any = {10: [10, 3.14, 200]}
var f map[any]func(int) int = {10: x => x}

echo a
echo m
echo f
`, `package main

import "fmt"

var a [][]interface{} = [][]interface{}{[]interface{}{10, 3.14, 200}, []interface{}{100, 200}}
var m map[interface{}][]interface{} = map[interface{}][]interface{}{10: []interface{}{10, 3.14, 200}}
var f map[interface{}]func(int) int = map[interface{}]func(int) int{10: func(x int) int {
	return x
}}

func main() {
	fmt.Println(a)
	fmt.Println(m)
	fmt.Println(f)
}
`)
}

func TestErrWrapNoArgs(t *testing.T) {
	gopClTest(t, `
func foo(v ...int) (func(), error) {
	return nil, nil
}
func Bar() (int, error) {
	return 100, nil
}
foo!()
foo(1)!()
echo foo!
echo bar!
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
)

func foo(v ...int) (func(), error) {
	return nil, nil
}
func Bar() (int, error) {
	return 100, nil
}
func main() {
	func() (_xgo_ret func()) {
		var _xgo_err error
		_xgo_ret, _xgo_err = foo()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "foo", "/foo/bar.xgo", 8, "main.main")
			panic(_xgo_err)
		}
		return
	}()()
	func() (_xgo_ret func()) {
		var _xgo_err error
		_xgo_ret, _xgo_err = foo(1)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "foo(1)", "/foo/bar.xgo", 9, "main.main")
			panic(_xgo_err)
		}
		return
	}()()
	fmt.Println(func() (_xgo_ret func()) {
		var _xgo_err error
		_xgo_ret, _xgo_err = foo()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "foo", "/foo/bar.xgo", 10, "main.main")
			panic(_xgo_err)
		}
		return
	}())
	fmt.Println(func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = Bar()
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "bar", "/foo/bar.xgo", 11, "main.main")
			panic(_xgo_err)
		}
		return
	}())
}
`)
}

func TestCommentFunc(t *testing.T) {
	gopClTestEx(t, gblConfLine, "main", `
import (
	"strconv"
)

func add(x, y string) (int, error) {
	return strconv.atoi(x)? + strconv.atoi(y)?, nil
}

func addSafe(x, y string) int {
	return strconv.atoi(x)?:0 + strconv.atoi(y)?:0
}

echo add("100", "23")!

sum, err := add("10", "abc")
echo sum, err

echo addSafe("10", "abc")
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
	"strconv"
)
//line /foo/bar.xgo:6:1
func add(x string, y string) (int, error) {
//line /foo/bar.xgo:7:1
	var _autoGo_1 int
//line /foo/bar.xgo:7:1
	{
//line /foo/bar.xgo:7:1
		var _xgo_err error
//line /foo/bar.xgo:7:1
		_autoGo_1, _xgo_err = strconv.Atoi(x)
//line /foo/bar.xgo:7:1
		if _xgo_err != nil {
//line /foo/bar.xgo:7:1
			_xgo_err = errors.NewFrame(_xgo_err, "strconv.atoi(x)", "/foo/bar.xgo", 7, "main.add")
//line /foo/bar.xgo:7:1
			return 0, _xgo_err
		}
//line /foo/bar.xgo:7:1
		goto _autoGo_2
	_autoGo_2:
//line /foo/bar.xgo:7:1
	}
//line /foo/bar.xgo:7:1
	var _autoGo_3 int
//line /foo/bar.xgo:7:1
	{
//line /foo/bar.xgo:7:1
		var _xgo_err error
//line /foo/bar.xgo:7:1
		_autoGo_3, _xgo_err = strconv.Atoi(y)
//line /foo/bar.xgo:7:1
		if _xgo_err != nil {
//line /foo/bar.xgo:7:1
			_xgo_err = errors.NewFrame(_xgo_err, "strconv.atoi(y)", "/foo/bar.xgo", 7, "main.add")
//line /foo/bar.xgo:7:1
			return 0, _xgo_err
		}
//line /foo/bar.xgo:7:1
		goto _autoGo_4
	_autoGo_4:
//line /foo/bar.xgo:7:1
	}
//line /foo/bar.xgo:7:1
	return _autoGo_1 + _autoGo_3, nil
}
//line /foo/bar.xgo:10:1
func addSafe(x string, y string) int {
//line /foo/bar.xgo:11:1
	return func() (_xgo_ret int) {
//line /foo/bar.xgo:11:1
		var _xgo_err error
//line /foo/bar.xgo:11:1
		_xgo_ret, _xgo_err = strconv.Atoi(x)
//line /foo/bar.xgo:11:1
		if _xgo_err != nil {
//line /foo/bar.xgo:11:1
			return 0
		}
//line /foo/bar.xgo:11:1
		return
	}() + func() (_xgo_ret int) {
//line /foo/bar.xgo:11:1
		var _xgo_err error
//line /foo/bar.xgo:11:1
		_xgo_ret, _xgo_err = strconv.Atoi(y)
//line /foo/bar.xgo:11:1
		if _xgo_err != nil {
//line /foo/bar.xgo:11:1
			return 0
		}
//line /foo/bar.xgo:11:1
		return
	}()
}
//line /foo/bar.xgo:14
func main() {
//line /foo/bar.xgo:14:1
	fmt.Println(func() (_xgo_ret int) {
//line /foo/bar.xgo:14:1
		var _xgo_err error
//line /foo/bar.xgo:14:1
		_xgo_ret, _xgo_err = add("100", "23")
//line /foo/bar.xgo:14:1
		if _xgo_err != nil {
//line /foo/bar.xgo:14:1
			_xgo_err = errors.NewFrame(_xgo_err, "add(\"100\", \"23\")", "/foo/bar.xgo", 14, "main.main")
//line /foo/bar.xgo:14:1
			panic(_xgo_err)
		}
//line /foo/bar.xgo:14:1
		return
	}())
//line /foo/bar.xgo:16:1
	sum, err := add("10", "abc")
//line /foo/bar.xgo:17:1
	fmt.Println(sum, err)
//line /foo/bar.xgo:19:1
	fmt.Println(addSafe("10", "abc"))
}
`)
}
