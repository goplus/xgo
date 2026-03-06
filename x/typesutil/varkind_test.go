//go:build go1.25

package typesutil_test

import (
	"fmt"
	"go/token"
	"go/types"
	"slices"
	"sort"
	"testing"

	"github.com/goplus/xgo/x/typesutil"
)

func testXGoVarKind(t *testing.T, name string, src any, want []string) {
	fset := token.NewFileSet()
	_, info, err := parseSource(fset, name, src, 0)
	if err != nil {
		t.Fatal("parserSource error", err)
	}
	testVarKind(t, info, want)
}

func testVarKind(t *testing.T, info *typesutil.Info, want []string) {
	var got []string
	for _, obj := range info.Defs {
		if v, ok := obj.(*types.Var); ok {
			got = append(got, fmt.Sprintf("%s: %v", v.Name(), v.Kind()))
		}
	}
	sort.Strings(got)
	if !slices.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestVarKind(t *testing.T) {
	src := `package p

var global1 = 100
var global2 int

type T struct { field int }

func (recv T) f(param int) (result int) {
	var local int
	var local1 = 100
	local2 := 0
	switch local3 := any(local).(type) {
	default:
		_ = local3
	}
	return local2
}
`
	want := []string{
		"field: FieldVar",
		"global1: PackageVar",
		"global2: PackageVar",
		"local1: LocalVar",
		"local2: LocalVar",
		"local: LocalVar",
		"param: ParamVar",
		"recv: RecvVar",
		"result: ResultVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestGoxVarKind(t *testing.T) {
	src := `
var (
	info  int
	title string
)

var global int

func addInt(a, b int) (c int) {
	return a + b
}
`
	want := []string{
		"a: ParamVar",
		"b: ParamVar",
		"c: ResultVar",
		"global: PackageVar",
		"info: FieldVar",
		"this: RecvVar",
		"title: FieldVar",
	}
	testXGoVarKind(t, "Rect.gox", src, want)
}

func TestSpxVarKind(t *testing.T) {
	src := `
var (
	a int
)

type info struct {
	x int
	y int
}

func onInit() {
	a = 1
	clone
	clone info{1,2}
	clone &info{1,2}
}

func onCloned() {
	say("Hi")
}

func demo(p1 int, p2 int) (r int) {
	return a + p1 + p2
}

`
	fset := token.NewFileSet()
	_, info, _, err := parseMixedSource(spxMod, fset, "Kai.tspx", src, "", "", spxParserConf(), false)
	if err != nil {
		t.Fatal("parserSource error", err)
	}
	want := []string{
		"a: FieldVar",
		"p1: ParamVar",
		"p2: ParamVar",
		"r: ResultVar",
		"this: RecvVar",
		"x: FieldVar",
		"y: FieldVar",
	}
	testVarKind(t, info, want)
}

func TestRangeVarKind(t *testing.T) {
	src := `
a := []int{100,200}
for k, v := range a {
	_ = k
	_ = v
}
var m map[int]string
for k, v := range m {
	_ = k
	_ = v
}
for v := range m {
	_ = v
}
`
	want := []string{
		"a: LocalVar",
		"k: LocalVar",
		"k: LocalVar",
		"m: LocalVar",
		"v: LocalVar",
		"v: LocalVar",
		"v: LocalVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestForPhraseVarKind(t *testing.T) {
	src := `
sum := 0
for x <- [1, 3, 5, 7, 11, 13, 17], x > 3 {
	sum = sum + x
}
println sum
`
	want := []string{
		"sum: LocalVar",
		"x: LocalVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestMapComprehensionVarKind(t *testing.T) {
	src := `
y := {x: i for i, x <- ["1", "3", "5", "7", "11"]}
println y
`
	want := []string{
		"i: LocalVar",
		"x: LocalVar",
		"y: LocalVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestErrorWrapVarKind(t *testing.T) {
	src := `
import (
	"strconv"
)

func add(x, y string) (r int, e error) {
	return strconv.atoi(x)? + strconv.atoi(y)?, nil
}

func addSafe(x, y string) int {
	return strconv.atoi(x)?:0 + strconv.atoi(y)?:0
}

// Error handling
// We reinvent the error handling specification in XGo. We call them ErrWrap expressions:

// expr! // panic if err
// expr? // return if err
// expr?:defval // use defval if err

n := add("100", "23")!
sum, err := add("10", "abc")
n2 := addSafe("10", "abc")
`
	want := []string{
		"e: ResultVar",
		"err: LocalVar",
		"n2: LocalVar",
		"n: LocalVar",
		"r: ResultVar",
		"sum: LocalVar",
		"x: ParamVar",
		"x: ParamVar",
		"y: ParamVar",
		"y: ParamVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestTupleVarKind(t *testing.T) {
	src := `
type Point (x, y int)

pt := Point(2, 3)
echo pt.x, pt.y

pt = (100, 200)
echo pt

pt2 := Point(pt)
echo pt2

pt3 := Point(y = 5, x = 3)
echo pt3.x, pt3.y
`
	want := []string{
		"pt2: LocalVar",
		"pt3: LocalVar",
		"pt: LocalVar",
		"x: FieldVar",
		"y: FieldVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestDQLXGoVarKind(t *testing.T) {
	src := `
doc := xgo` + "`" + `
x, y := "Hi", 123
echo x
print y
` + "`" + `!

stmts := doc.shadowEntry.body.list.*@(self.class == "ExprStmt")
for fn in stmts.x@(self.class == "CallExpr").fun@(self.class == "Ident") {
	echo fn.$name
}
`
	want := []string{
		"doc: LocalVar",
		"fn: LocalVar",
		"stmts: LocalVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}

func TestKwArgsVarKind(t *testing.T) {
	src := `
type Config (timeout, maxRetries int, debug bool)

func run(task int, cfg Config?) {
	if cfg.timeout == 0 {
		cfg.timeout = 30
	}
	if cfg.maxRetries == 0 {
		cfg.maxRetries = 3
	}
    echo "timeout:", cfg.timeout, "maxRetries:", cfg.maxRetries, "debug:", cfg.debug
	echo "task:", task
}

run 100, timeout = 60, maxRetries = 5
run 200
`
	want := []string{
		"cfg: VarKind(255)",
		"debug: FieldVar",
		"maxRetries: FieldVar",
		"task: ParamVar",
		"timeout: FieldVar",
	}
	testXGoVarKind(t, "main.xgo", src, want)
}
